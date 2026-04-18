package github

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/gertzgal/gh-prs/internal/filter"
	"github.com/gertzgal/gh-prs/internal/model"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func fakeGraphQLClient(t *testing.T, body []byte, status int) *api.GraphQLClient {
	t.Helper()
	rt := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: status,
			Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
			Body:       io.NopCloser(bytes.NewReader(body)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Request:    r,
		}, nil
	})
	c, err := api.NewGraphQLClient(api.ClientOptions{
		Host:      "github.com",
		AuthToken: "test-token",
		Transport: rt,
	})
	if err != nil {
		t.Fatalf("NewGraphQLClient: %v", err)
	}
	return c
}

func fixtureBytes(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "testdata", "fixtures", name+".json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return raw
}

func fakeLocator() func() (repository.Repository, error) {
	return func() (repository.Repository, error) {
		return repository.Repository{Host: "github.com", Owner: "acme-org", Name: "widget"}, nil
	}
}

func TestFetchRepo_Widget4Stack(t *testing.T) {
	gql := fakeGraphQLClient(t, fixtureBytes(t, "graphql-widget-4-stack"), 200)
	c := newClientWith(gql, fakeLocator())

	repo, err := c.FetchRepo(context.Background(), filter.Set{})
	if err != nil {
		t.Fatalf("FetchRepo: %v", err)
	}

	if len(repo.PRs) != 4 {
		t.Fatalf("want 4 PRs, got %d", len(repo.PRs))
	}
	want := [][3]string{
		{"1001", "main", "ACME-100/feature-1-foundation"},
		{"1002", "ACME-100/feature-1-foundation", "ACME-100/feature-2-ui"},
		{"1003", "ACME-100/feature-2-ui", "ACME-100/feature-3-wiring"},
		{"1004", "ACME-100/feature-3-wiring", "ACME-100/feature-4-cleanup"},
	}
	for i, p := range repo.PRs {
		got := [3]string{itoa(p.Number), p.BaseRefName, p.HeadRefName}
		if got != want[i] {
			t.Errorf("PR[%d]: want %v, got %v", i, want[i], got)
		}
	}
	if repo.RateLimit == nil || repo.RateLimit.Cost != 1 {
		t.Errorf("rateLimit.cost: want 1, got %+v", repo.RateLimit)
	}
	if repo.ViewerLogin != "octocat" {
		t.Errorf("viewerLogin: want octocat, got %q", repo.ViewerLogin)
	}
	if repo.Owner != "acme-org" || repo.Name != "widget" {
		t.Errorf("repo identity: want acme-org/widget, got %s/%s", repo.Owner, repo.Name)
	}
}

func TestFetchRepo_GadgetStandalone(t *testing.T) {
	gql := fakeGraphQLClient(t, fixtureBytes(t, "graphql-gadget-standalone"), 200)
	c := newClientWith(gql, fakeLocator())

	repo, err := c.FetchRepo(context.Background(), filter.Set{})
	if err != nil {
		t.Fatalf("FetchRepo: %v", err)
	}

	if len(repo.PRs) != 2 {
		t.Fatalf("want 2 PRs, got %d", len(repo.PRs))
	}
	for _, p := range repo.PRs {
		if p.BaseRefName != "main" {
			t.Errorf("PR #%d: baseRefName want main, got %q", p.Number, p.BaseRefName)
		}
	}
	drafts := 0
	for _, p := range repo.PRs {
		if p.IsDraft {
			drafts++
		}
	}
	if drafts != 1 {
		t.Errorf("draft count: want 1, got %d", drafts)
	}
	if repo.DefaultBranch != "main" {
		t.Errorf("defaultBranch: want main, got %q", repo.DefaultBranch)
	}
}

func TestFetchRepo_Empty(t *testing.T) {
	gql := fakeGraphQLClient(t, fixtureBytes(t, "graphql-empty"), 200)
	c := newClientWith(gql, fakeLocator())

	repo, err := c.FetchRepo(context.Background(), filter.Set{})
	if err != nil {
		t.Fatalf("FetchRepo: %v", err)
	}
	if len(repo.PRs) != 0 {
		t.Errorf("want 0 PRs, got %d", len(repo.PRs))
	}
	if repo.DefaultBranch != "main" {
		t.Errorf("defaultBranch: want main, got %q", repo.DefaultBranch)
	}
}

func TestFetchRepo_ErrorFixture(t *testing.T) {
	gql := fakeGraphQLClient(t, fixtureBytes(t, "graphql-error"), 200)
	c := newClientWith(gql, fakeLocator())

	_, err := c.FetchRepo(context.Background(), filter.Set{})
	if err == nil {
		t.Fatal("want error, got nil")
	}
	var gh *model.GhError
	if !errors.As(err, &gh) {
		t.Fatalf("want *model.GhError, got %T: %v", err, err)
	}
	if !bytes.Contains([]byte(gh.Msg), []byte("rate limit")) {
		t.Errorf("Msg: want to contain %q, got %q", "rate limit", gh.Msg)
	}
}

func TestFetchRepo_NonJSONBody(t *testing.T) {
	gql := fakeGraphQLClient(t, []byte("not json!"), 200)
	c := newClientWith(gql, fakeLocator())

	_, err := c.FetchRepo(context.Background(), filter.Set{})
	if err == nil {
		t.Fatal("want error, got nil")
	}
	var gh *model.GhError
	if !errors.As(err, &gh) {
		t.Fatalf("want *model.GhError, got %T: %v", err, err)
	}
}

func TestFetchRepo_HTTP500(t *testing.T) {
	gql := fakeGraphQLClient(t, []byte(`{"message":"kaboom"}`), 500)
	c := newClientWith(gql, fakeLocator())

	_, err := c.FetchRepo(context.Background(), filter.Set{})
	if err == nil {
		t.Fatal("want error, got nil")
	}
	var gh *model.GhError
	if !errors.As(err, &gh) {
		t.Fatalf("want *model.GhError, got %T: %v", err, err)
	}
	if !bytes.Contains([]byte(gh.Msg), []byte("500")) {
		t.Errorf("Msg: want message reflecting status 500, got %q", gh.Msg)
	}
}

func TestFetchRepo_NullStatusCheckRollup(t *testing.T) {
	body := []byte(`{
		"data": {
			"rateLimit": null,
			"viewer": {"login": "someone"},
			"repository": {"defaultBranchRef": {"name": "main"}},
			"search": {
				"nodes": [
					{
						"number": 99,
						"title": "test pr",
						"url": "https://github.com/x/y/pull/99",
						"isDraft": false,
						"headRefName": "feat",
						"baseRefName": "main",
						"additions": 1,
						"deletions": 0,
						"changedFiles": 1,
						"reviewDecision": null,
						"mergeStateStatus": "CLEAN",
						"commits": {"nodes": [{"commit": {"statusCheckRollup": null}}]}
					}
				]
			}
		}
	}`)
	gql := fakeGraphQLClient(t, body, 200)
	c := newClientWith(gql, fakeLocator())

	repo, err := c.FetchRepo(context.Background(), filter.Set{})
	if err != nil {
		t.Fatalf("FetchRepo: %v", err)
	}
	if len(repo.PRs) != 1 {
		t.Fatalf("want 1 PR, got %d", len(repo.PRs))
	}
	if repo.PRs[0].CiState != "" {
		t.Errorf("CiState: want empty, got %q", repo.PRs[0].CiState)
	}
}

func TestFetchRepo_RepoResolveError(t *testing.T) {
	c := newClientWith(nil, func() (repository.Repository, error) {
		return repository.Repository{}, errors.New("no git remotes")
	})

	_, err := c.FetchRepo(context.Background(), filter.Set{})
	if err == nil {
		t.Fatal("want error, got nil")
	}
	if !errors.Is(err, model.ErrRepoNotFound) {
		t.Errorf("want errors.Is(ErrRepoNotFound), got %T: %v", err, err)
	}
}

// ---------------------------------------------------------------------------
// buildSearchQuery — unit tests (white-box, same package)
// ---------------------------------------------------------------------------

func TestBuildSearchQuery_BaseQualifiersAlwaysPresent(t *testing.T) {
	q := buildSearchQuery("acme", "widget", filter.Set{})
	for _, want := range []string{"is:pr", "is:open", "repo:acme/widget"} {
		if !strings.Contains(q, want) {
			t.Errorf("query %q missing %q", q, want)
		}
	}
}

func TestBuildSearchQuery_ZeroFilters_NoAuthorQualifier(t *testing.T) {
	q := buildSearchQuery("acme", "widget", filter.Set{})
	if strings.Contains(q, "author:") {
		t.Errorf("zero filter set should produce no author qualifier, got %q", q)
	}
}

func TestBuildSearchQuery_SingleAuthor(t *testing.T) {
	s := filter.NewSet(
		[]filter.QueryFilter{filter.NewAuthorFilter([]string{"@me"})},
		nil,
	)
	q := buildSearchQuery("acme", "widget", s)
	want := "is:pr is:open repo:acme/widget author:@me"
	if q != want {
		t.Errorf("got %q, want %q", q, want)
	}
}

func TestBuildSearchQuery_MultipleAuthors_ORed(t *testing.T) {
	s := filter.NewSet(
		[]filter.QueryFilter{filter.NewAuthorFilter([]string{"alice", "bob"})},
		nil,
	)
	q := buildSearchQuery("acme", "widget", s)
	want := "is:pr is:open repo:acme/widget author:alice author:bob"
	if q != want {
		t.Errorf("got %q, want %q", q, want)
	}
}

// ---------------------------------------------------------------------------
// Dry-run: verify the correct "q" variable reaches the GitHub API
// ---------------------------------------------------------------------------

// captureRequestBody is a roundTripper that stores the raw request body and
// then replies with the provided fixture bytes so FetchRepo can complete.
type captureRoundTripper struct {
	capturedBody []byte
	fixture      []byte
}

func (rt *captureRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	rt.capturedBody = body
	return &http.Response{
		StatusCode: 200,
		Status:     "200 OK",
		Body:       io.NopCloser(bytes.NewReader(rt.fixture)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Request:    r,
	}, nil
}

func TestFetchRepo_DryRun_QueryContainsAuthorFragment(t *testing.T) {
	rt := &captureRoundTripper{fixture: fixtureBytes(t, "graphql-empty")}
	gql, err := api.NewGraphQLClient(api.ClientOptions{
		Host:      "github.com",
		AuthToken: "test-token",
		Transport: rt,
	})
	if err != nil {
		t.Fatalf("NewGraphQLClient: %v", err)
	}

	s := filter.NewSet(
		[]filter.QueryFilter{filter.NewAuthorFilter([]string{"alice"})},
		nil,
	)
	c := newClientWith(gql, fakeLocator())
	_, _ = c.FetchRepo(context.Background(), s)

	body := string(rt.capturedBody)
	if !strings.Contains(body, "author:alice") {
		t.Errorf("expected request body to contain author:alice, got:\n%s", body)
	}
}

func TestFetchRepo_DryRun_DefaultAuthorMe(t *testing.T) {
	rt := &captureRoundTripper{fixture: fixtureBytes(t, "graphql-empty")}
	gql, err := api.NewGraphQLClient(api.ClientOptions{
		Host:      "github.com",
		AuthToken: "test-token",
		Transport: rt,
	})
	if err != nil {
		t.Fatalf("NewGraphQLClient: %v", err)
	}

	// Simulate the CLI default: inject @me when no --author flag is given.
	s := filter.NewSet(
		[]filter.QueryFilter{filter.NewAuthorFilter([]string{"@me"})},
		nil,
	)
	c := newClientWith(gql, fakeLocator())
	_, _ = c.FetchRepo(context.Background(), s)

	body := string(rt.capturedBody)
	if !strings.Contains(body, "author:@me") {
		t.Errorf("expected request body to contain author:@me, got:\n%s", body)
	}
}

func TestFetchRepo_DryRun_MultipleAuthors(t *testing.T) {
	rt := &captureRoundTripper{fixture: fixtureBytes(t, "graphql-empty")}
	gql, err := api.NewGraphQLClient(api.ClientOptions{
		Host:      "github.com",
		AuthToken: "test-token",
		Transport: rt,
	})
	if err != nil {
		t.Fatalf("NewGraphQLClient: %v", err)
	}

	s := filter.NewSet(
		[]filter.QueryFilter{filter.NewAuthorFilter([]string{"alice", "bob"})},
		nil,
	)
	c := newClientWith(gql, fakeLocator())
	_, _ = c.FetchRepo(context.Background(), s)

	body := string(rt.capturedBody)
	for _, frag := range []string{"author:alice", "author:bob"} {
		if !strings.Contains(body, frag) {
			t.Errorf("expected request body to contain %q, got:\n%s", frag, body)
		}
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
