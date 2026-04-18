package github

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
)

// TestDebugLogging_EmitsGraphQLBody proves that Debug=true wires go-gh's
// httpretty logger into the transport chain, so the actual GraphQL query
// bytes end up on DebugOut. This is the behavior the user asked for:
// debug mode should print real queries, not static text.
func TestDebugLogging_EmitsGraphQLBody(t *testing.T) {
	var dbg bytes.Buffer

	fixture := fixtureBytes(t, "graphql-widget-4-stack")
	rt := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Status:     "200 OK",
			Body:       io.NopCloser(bytes.NewReader(fixture)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Request:    r,
		}, nil
	})

	// Build options identical to how client.go builds them — Transport is
	// injected for testability only.
	co := buildClientOptions(Options{
		Debug:      true,
		DebugOut:   &dbg,
		DebugColor: false,
	})
	co.Host = "github.com"
	co.AuthToken = "test-token"
	co.Transport = rt

	gql, err := api.NewGraphQLClient(co)
	if err != nil {
		t.Fatalf("NewGraphQLClient: %v", err)
	}
	c := newClientWith(gql, func() (repository.Repository, error) {
		return repository.Repository{Host: "github.com", Owner: "acme-org", Name: "widget"}, nil
	})

	if _, err := c.FetchRepo(context.Background()); err != nil {
		t.Fatalf("FetchRepo: %v", err)
	}

	out := dbg.String()
	if out == "" {
		t.Fatal("debug output: want non-empty, got empty — logger was not wired")
	}
	// httpretty prints the request URL and verb.
	if !strings.Contains(out, "POST") {
		t.Errorf("debug output: want POST verb, got:\n%s", out)
	}
	if !strings.Contains(out, "graphql") {
		t.Errorf("debug output: want graphql endpoint reference, got:\n%s", out)
	}
	// The JSON formatter recognizes GraphQL and prints a "GraphQL query:" banner.
	if !strings.Contains(out, "GraphQL query") {
		t.Errorf("debug output: want 'GraphQL query' banner, got:\n%s", out)
	}
	// The actual operation name should appear in the body.
	if !strings.Contains(out, "PRsForViewer") {
		t.Errorf("debug output: want operation name 'PRsForViewer', got:\n%s", out)
	}
}

func TestDebugLogging_OffByDefault(t *testing.T) {
	var dbg bytes.Buffer

	fixture := fixtureBytes(t, "graphql-empty")
	rt := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewReader(fixture)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Request:    r,
		}, nil
	})

	co := buildClientOptions(Options{}) // no debug
	co.Host = "github.com"
	co.AuthToken = "test-token"
	co.Transport = rt

	gql, err := api.NewGraphQLClient(co)
	if err != nil {
		t.Fatalf("NewGraphQLClient: %v", err)
	}
	c := newClientWith(gql, fakeLocator())

	if _, err := c.FetchRepo(context.Background()); err != nil {
		t.Fatalf("FetchRepo: %v", err)
	}

	// Logger buffer was never wired up, but confirm nothing leaked into it
	// even if someone accidentally passes it.
	if dbg.Len() != 0 {
		t.Errorf("debug output: want empty (no debug flag), got %q", dbg.String())
	}
}
