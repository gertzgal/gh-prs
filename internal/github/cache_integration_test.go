package github

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/gertzgal/gh-prs/internal/filter"
)

// TestSWRIntegration_WarmHit_SecondCallReadsDisk verifies that when cache is
// enabled, the second FetchRepo() serves from disk without a network round-trip.
func TestSWRIntegration_WarmHit_SecondCallReadsDisk(t *testing.T) {
	fixture := fixtureBytes(t, "graphql-widget-4-stack")

	var hits int64
	rt := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		atomic.AddInt64(&hits, 1)
		return &http.Response{
			StatusCode: 200,
			Status:     "200 OK",
			Body:       io.NopCloser(bytes.NewReader(fixture)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Request:    r,
		}, nil
	})

	cacheDir := filepath.Join(t.TempDir(), "cache")

	newInnerClient := func() Client {
		co := buildClientOptions(Options{})
		co.Host = "github.com"
		co.AuthToken = "test-token"
		co.Transport = rt
		gql, err := api.NewGraphQLClient(co)
		if err != nil {
			t.Fatalf("NewGraphQLClient: %v", err)
		}
		return newClientWith(gql, fakeLocator())
	}

	inner := newInnerClient()
	swr := NewSWRClient(inner, fakeLocator(), cacheDir, 30*time.Second)
	swr.accountID = func() string { return "testuser" }

	// First call: cold miss, hits network.
	repo1, err := swr.FetchRepo(context.Background(), filter.Set{})
	if err != nil {
		t.Fatalf("first FetchRepo: %v", err)
	}
	if got := atomic.LoadInt64(&hits); got != 1 {
		t.Fatalf("first call: want 1 transport hit, got %d", got)
	}
	if repo1.RateLimit == nil || repo1.RateLimit.Cost != 1 {
		t.Fatalf("first call: want cost 1, got %+v", repo1.RateLimit)
	}

	// Second call: warm hit, no network.
	inner2 := newInnerClient()
	swr2 := NewSWRClient(inner2, fakeLocator(), cacheDir, 30*time.Second)
	swr2.accountID = func() string { return "testuser" }
	repo2, err := swr2.FetchRepo(context.Background(), filter.Set{})
	if err != nil {
		t.Fatalf("second FetchRepo: %v", err)
	}
	if got := atomic.LoadInt64(&hits); got != 1 {
		t.Errorf("second call: want still 1 transport hit (cache hit), got %d", got)
	}
	if repo2.RateLimit == nil || repo2.RateLimit.Cost != 0 {
		t.Errorf("second call: want cost 0 (cache hit), got %+v", repo2.RateLimit)
	}
}

// TestSWRIntegration_DisabledAlwaysHitsNetwork verifies that when cache is
// disabled, every FetchRepo() hits the network.
func TestSWRIntegration_DisabledAlwaysHitsNetwork(t *testing.T) {
	fixture := fixtureBytes(t, "graphql-widget-4-stack")

	var hits int64
	rt := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		atomic.AddInt64(&hits, 1)
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewReader(fixture)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Request:    r,
		}, nil
	})

	newClient := func() Client {
		co := buildClientOptions(Options{})
		co.Host = "github.com"
		co.AuthToken = "test-token"
		co.Transport = rt
		gql, err := api.NewGraphQLClient(co)
		if err != nil {
			t.Fatalf("NewGraphQLClient: %v", err)
		}
		return newClientWith(gql, fakeLocator())
	}

	for i := 0; i < 3; i++ {
		if _, err := newClient().FetchRepo(context.Background(), filter.Set{}); err != nil {
			t.Fatalf("FetchRepo #%d: %v", i, err)
		}
	}
	if got := atomic.LoadInt64(&hits); got != 3 {
		t.Errorf("want 3 transport hits (cache disabled), got %d", got)
	}
}
