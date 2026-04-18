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

// TestCacheWiring_SecondCallHitsDisk verifies the user-facing perf win: when
// cache is enabled and the request fingerprint matches, the second
// FetchRepo() does not call the transport. If this test regresses, caching
// is broken and the speed improvement is gone.
func TestCacheWiring_SecondCallHitsDisk(t *testing.T) {
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

	newClient := func() Client {
		co := buildClientOptions(Options{
			EnableCache: true,
			CacheTTL:    30 * time.Second,
			CacheDir:    cacheDir,
		})
		co.Host = "github.com"
		co.AuthToken = "test-token"
		co.Transport = rt
		gql, err := api.NewGraphQLClient(co)
		if err != nil {
			t.Fatalf("NewGraphQLClient: %v", err)
		}
		return newClientWith(gql, fakeLocator())
	}

	c1 := newClient()
	if _, err := c1.FetchRepo(context.Background(), filter.Set{}); err != nil {
		t.Fatalf("first FetchRepo: %v", err)
	}
	if got := atomic.LoadInt64(&hits); got != 1 {
		t.Fatalf("first call: want 1 transport hit, got %d", got)
	}

	// Fresh client (separate process simulation), same cache dir.
	c2 := newClient()
	if _, err := c2.FetchRepo(context.Background(), filter.Set{}); err != nil {
		t.Fatalf("second FetchRepo: %v", err)
	}
	if got := atomic.LoadInt64(&hits); got != 1 {
		t.Errorf("second call: want still 1 transport hit (cache hit), got %d", got)
	}
}

func TestCacheWiring_DisabledAlwaysHitsNetwork(t *testing.T) {
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
		co := buildClientOptions(Options{}) // cache off
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
