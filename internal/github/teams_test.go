package github

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/api"
)

// scriptedRoundTripper replies with a sequence of canned bodies, in order.
// Useful for exercising paginated GraphQL queries: each successive request
// gets the next fixture, regardless of body content.
type scriptedRoundTripper struct {
	responses [][]byte
	calls     int
	captured  []string
}

func (rt *scriptedRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	body, _ := io.ReadAll(r.Body)
	rt.captured = append(rt.captured, string(body))
	if rt.calls >= len(rt.responses) {
		return nil, fmt.Errorf("scriptedRoundTripper: unexpected extra call #%d", rt.calls+1)
	}
	out := rt.responses[rt.calls]
	rt.calls++
	return &http.Response{
		StatusCode: 200,
		Status:     "200 OK",
		Body:       io.NopCloser(bytes.NewReader(out)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Request:    r,
	}, nil
}

func newScriptedResolver(t *testing.T, responses [][]byte) (TeamResolver, *scriptedRoundTripper) {
	t.Helper()
	rt := &scriptedRoundTripper{responses: responses}
	gql, err := api.NewGraphQLClient(api.ClientOptions{
		Host:      "github.com",
		AuthToken: "test-token",
		Transport: rt,
	})
	if err != nil {
		t.Fatalf("NewGraphQLClient: %v", err)
	}
	return newTeamResolverWith(gql), rt
}

func teamPageBody(members []string, hasNext bool, endCursor string) []byte {
	nodes := make([]map[string]any, len(members))
	for i, m := range members {
		nodes[i] = map[string]any{"login": m}
	}
	payload := map[string]any{
		"data": map[string]any{
			"organization": map[string]any{
				"team": map[string]any{
					"members": map[string]any{
						"pageInfo": map[string]any{
							"hasNextPage": hasNext,
							"endCursor":   endCursor,
						},
						"nodes": nodes,
					},
				},
			},
		},
	}
	raw, _ := json.Marshal(payload)
	return raw
}

func TestResolveTeam_SinglePage(t *testing.T) {
	body := teamPageBody([]string{"alice", "bob", "carol"}, false, "")
	resolver, _ := newScriptedResolver(t, [][]byte{body})

	logins, err := resolver.ResolveTeam(context.Background(), "acme-org", "widgets")
	if err != nil {
		t.Fatalf("ResolveTeam: %v", err)
	}
	want := []string{"alice", "bob", "carol"}
	if !sameStrings(logins, want) {
		t.Errorf("logins: want %v, got %v", want, logins)
	}
}

func TestResolveTeam_Paginated(t *testing.T) {
	page1 := teamPageBody([]string{"alice", "bob"}, true, "cursor-1")
	page2 := teamPageBody([]string{"carol", "dave"}, false, "")
	resolver, rt := newScriptedResolver(t, [][]byte{page1, page2})

	logins, err := resolver.ResolveTeam(context.Background(), "acme-org", "widgets")
	if err != nil {
		t.Fatalf("ResolveTeam: %v", err)
	}
	if rt.calls != 2 {
		t.Errorf("calls: want 2 (paginated), got %d", rt.calls)
	}
	want := []string{"alice", "bob", "carol", "dave"}
	if !sameStrings(logins, want) {
		t.Errorf("logins: want %v, got %v", want, logins)
	}
	// Second request should carry the endCursor from page 1.
	if !strings.Contains(rt.captured[1], "cursor-1") {
		t.Errorf("page-2 request: expected to carry cursor-1, got:\n%s", rt.captured[1])
	}
}

func TestResolveTeam_LowercasesSlug(t *testing.T) {
	body := teamPageBody([]string{"alice"}, false, "")
	resolver, rt := newScriptedResolver(t, [][]byte{body})

	if _, err := resolver.ResolveTeam(context.Background(), "ACME-ORG", "Widgets"); err != nil {
		t.Fatalf("ResolveTeam: %v", err)
	}
	if len(rt.captured) == 0 {
		t.Fatal("no request captured")
	}
	req := rt.captured[0]
	// The GraphQL variables payload should contain the lowercased values.
	if !strings.Contains(req, `"org":"acme-org"`) {
		t.Errorf("expected org variable lowercased to acme-org, got:\n%s", req)
	}
	if !strings.Contains(req, `"slug":"widgets"`) {
		t.Errorf("expected slug variable lowercased to widgets, got:\n%s", req)
	}
}

func TestResolveTeam_NotFound(t *testing.T) {
	// organization or team is null -> ErrTeamNotFound.
	body := []byte(`{"data":{"organization":{"team":null}}}`)
	resolver, _ := newScriptedResolver(t, [][]byte{body})

	_, err := resolver.ResolveTeam(context.Background(), "acme-org", "ghost")
	if err == nil {
		t.Fatal("want error for missing team, got nil")
	}
	if !errors.Is(err, ErrTeamNotFound) {
		t.Errorf("want errors.Is(ErrTeamNotFound), got: %v", err)
	}
}

func TestResolveTeam_EmptyArgs(t *testing.T) {
	resolver, _ := newScriptedResolver(t, [][]byte{[]byte(`{}`)})
	for _, tc := range []struct{ org, slug string }{
		{"", "widgets"},
		{"acme-org", ""},
		{"  ", "widgets"},
	} {
		if _, err := resolver.ResolveTeam(context.Background(), tc.org, tc.slug); err == nil {
			t.Errorf("ResolveTeam(%q,%q): want error, got nil", tc.org, tc.slug)
		}
	}
}

func TestResolveTeam_DedupesAcrossPages(t *testing.T) {
	// GraphQL should not return duplicates, but the resolver still de-dupes
	// defensively so callers get a clean union when reused across queries.
	page1 := teamPageBody([]string{"alice", "bob"}, true, "c1")
	page2 := teamPageBody([]string{"bob", "carol"}, false, "")
	resolver, _ := newScriptedResolver(t, [][]byte{page1, page2})

	logins, err := resolver.ResolveTeam(context.Background(), "acme-org", "widgets")
	if err != nil {
		t.Fatalf("ResolveTeam: %v", err)
	}
	want := []string{"alice", "bob", "carol"}
	if !sameStrings(logins, want) {
		t.Errorf("logins: want %v, got %v", want, logins)
	}
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
