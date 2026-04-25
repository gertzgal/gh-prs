package github

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/gertzgal/gh-prs/internal/filter"
	"github.com/gertzgal/gh-prs/internal/model"
)

type fakeInnerClient struct {
	repo      *model.Repo
	err       error
	callCount atomic.Int64
	blockChan chan struct{}
}

func (c *fakeInnerClient) FetchRepo(_ context.Context, _ filter.Set) (*model.Repo, error) {
	c.callCount.Add(1)
	if c.blockChan != nil {
		<-c.blockChan
	}
	if c.err != nil {
		return nil, c.err
	}
	return c.repo.Clone(), nil
}

func newFakeRepo() *model.Repo {
	return &model.Repo{
		Owner: "acme", Name: "widget",
		DefaultBranch: "main", ViewerLogin: "alice",
		PRs:       []model.PR{{Number: 1, Author: "alice"}},
		RateLimit: &model.RateLimit{Cost: 1, Remaining: 4999, ResetAt: "2026-04-17T17:21:19Z"},
	}
}

func TestSWR_ColdMiss_FetchesAndWrites(t *testing.T) {
	cacheDir := t.TempDir()
	inner := &fakeInnerClient{repo: newFakeRepo()}
	resolve := func() (repository.Repository, error) {
		return repository.Repository{Owner: "acme", Name: "widget"}, nil
	}
	swr := NewSWRClient(inner, resolve, cacheDir, 5*time.Minute)
	swr.accountID = func() string { return "testuser" }

	repo, err := swr.FetchRepo(context.Background(), filter.Set{})
	if err != nil {
		t.Fatalf("FetchRepo: %v", err)
	}
	if inner.callCount.Load() != 1 {
		t.Fatalf("want 1 inner call, got %d", inner.callCount.Load())
	}
	if repo.RateLimit.Cost != 1 {
		t.Fatalf("want cost 1 on cold miss, got %d", repo.RateLimit.Cost)
	}

	// Verify disk was written.
	store := newSWRStore(cacheDir)
	entry, err := store.read("testuser", "acme", "widget", cacheFilterKey(filter.Set{}))
	if err != nil {
		t.Fatalf("store.read: %v", err)
	}
	if entry == nil {
		t.Fatal("expected cache entry on disk, got nil")
	}
}

func TestSWR_WarmHit_ServesInstantly(t *testing.T) {
	cacheDir := t.TempDir()
	inner := &fakeInnerClient{repo: newFakeRepo()}
	resolve := func() (repository.Repository, error) {
		return repository.Repository{Owner: "acme", Name: "widget"}, nil
	}
	swr := NewSWRClient(inner, resolve, cacheDir, 5*time.Minute)
	swr.accountID = func() string { return "testuser" }

	// Cold miss to populate cache.
	_, _ = swr.FetchRepo(context.Background(), filter.Set{})

	// Warm hit should not call inner.
	repo, err := swr.FetchRepo(context.Background(), filter.Set{})
	if err != nil {
		t.Fatalf("FetchRepo: %v", err)
	}
	if inner.callCount.Load() != 1 {
		t.Fatalf("want 1 inner call, got %d", inner.callCount.Load())
	}
	if repo.RateLimit.Cost != 0 {
		t.Fatalf("want cost 0 on warm hit, got %d", repo.RateLimit.Cost)
	}
	if repo.CacheAge == 0 {
		t.Fatal("want CacheAge > 0 on warm hit")
	}
	if repo.IsStale {
		t.Fatal("want IsStale=false on warm hit")
	}
}

func TestSWR_HardExpired_BlocksOnFetch(t *testing.T) {
	cacheDir := t.TempDir()
	inner := &fakeInnerClient{repo: newFakeRepo()}
	resolve := func() (repository.Repository, error) {
		return repository.Repository{Owner: "acme", Name: "widget"}, nil
	}
	swr := NewSWRClient(inner, resolve, cacheDir, 5*time.Minute)
	swr.accountID = func() string { return "testuser" }

	// Seed an old cache entry.
	store := newSWRStore(cacheDir)
	oldRepo := newFakeRepo()
	oldRepo.RateLimit.Cost = 1
	_ = store.write("testuser", "acme", "widget", cacheFilterKey(filter.Set{}), oldRepo)

	// Rewind WrittenAt by advancing the clock.
	swr.clock = func() time.Time {
		return time.Now().Add(11 * time.Minute)
	}

	repo, err := swr.FetchRepo(context.Background(), filter.Set{})
	if err != nil {
		t.Fatalf("FetchRepo: %v", err)
	}
	if inner.callCount.Load() != 1 {
		t.Fatalf("want 1 inner call (hard expire), got %d", inner.callCount.Load())
	}
	if repo.RateLimit.Cost != 1 {
		t.Fatalf("want cost 1 on hard expire, got %d", repo.RateLimit.Cost)
	}
}

func TestSWR_WrongVersion_TreatedAsMiss(t *testing.T) {
	cacheDir := t.TempDir()
	inner := &fakeInnerClient{repo: newFakeRepo()}
	resolve := func() (repository.Repository, error) {
		return repository.Repository{Owner: "acme", Name: "widget"}, nil
	}
	swr := NewSWRClient(inner, resolve, cacheDir, 5*time.Minute)
	swr.accountID = func() string { return "testuser" }

	// Write a v1 entry manually.
	p := filepath.Join(cacheDir, "swr", "testuser", "acme_widget_"+cacheFilterKey(filter.Set{})+".json")
	_ = os.MkdirAll(filepath.Dir(p), 0750)
	v1 := []byte(`{"v":1,"t":"2026-04-17T00:00:00Z","data":{"owner":"acme","name":"widget"}}`)
	_ = os.WriteFile(p, v1, 0640)

	repo, err := swr.FetchRepo(context.Background(), filter.Set{})
	if err != nil {
		t.Fatalf("FetchRepo: %v", err)
	}
	if inner.callCount.Load() != 1 {
		t.Fatalf("want 1 inner call (wrong version), got %d", inner.callCount.Load())
	}
	if repo.RateLimit.Cost != 1 {
		t.Fatalf("want cost 1, got %d", repo.RateLimit.Cost)
	}
}

func TestSWR_ConcurrentRefresh_Deduplicated(t *testing.T) {
	cacheDir := t.TempDir()
	block := make(chan struct{})
	inner := &fakeInnerClient{repo: newFakeRepo(), blockChan: block}
	resolve := func() (repository.Repository, error) {
		return repository.Repository{Owner: "acme", Name: "widget"}, nil
	}
	swr := NewSWRClient(inner, resolve, cacheDir, 5*time.Minute)
	swr.accountID = func() string { return "testuser" }

	// Seed a soft-stale entry.
	store := newSWRStore(cacheDir)
	_ = store.write("testuser", "acme", "widget", cacheFilterKey(filter.Set{}), newFakeRepo())
	swr.clock = func() time.Time {
		return time.Now().Add(6 * time.Minute)
	}

	// First stale serve triggers refresh.
	_, _ = swr.FetchRepo(context.Background(), filter.Set{})

	// Second stale serve while refresh is in flight should not spawn another.
	_, _ = swr.FetchRepo(context.Background(), filter.Set{})

	// Give the background goroutine time to hit the block.
	time.Sleep(20 * time.Millisecond)

	if inner.callCount.Load() != 1 {
		t.Fatalf("want 1 inner call (dedup), got %d", inner.callCount.Load())
	}

	close(block)
	swr.LingerWait()
}

func TestSWR_WriteFailure_Silent(t *testing.T) {
	// Use a file path as cacheDir so MkdirAll fails.
	badDir := filepath.Join(t.TempDir(), "notadir")
	_ = os.WriteFile(badDir, []byte("x"), 0644)

	inner := &fakeInnerClient{repo: newFakeRepo()}
	resolve := func() (repository.Repository, error) {
		return repository.Repository{Owner: "acme", Name: "widget"}, nil
	}
	swr := NewSWRClient(inner, resolve, badDir, 5*time.Minute)
	swr.accountID = func() string { return "testuser" }

	// Should not panic even though cache write fails.
	repo, err := swr.FetchRepo(context.Background(), filter.Set{})
	if err != nil {
		t.Fatalf("FetchRepo: %v", err)
	}
	if repo.RateLimit.Cost != 1 {
		t.Fatalf("want cost 1, got %d", repo.RateLimit.Cost)
	}
}

func TestSWR_LingerCap_CancelsInFlightRequest(t *testing.T) {
	cacheDir := t.TempDir()
	block := make(chan struct{})
	inner := &fakeInnerClient{repo: newFakeRepo(), blockChan: block}
	resolve := func() (repository.Repository, error) {
		return repository.Repository{Owner: "acme", Name: "widget"}, nil
	}
	swr := NewSWRClient(inner, resolve, cacheDir, 5*time.Minute)
	swr.accountID = func() string { return "testuser" }
	swr.lingerCap = 50 * time.Millisecond

	// Seed a soft-stale entry.
	store := newSWRStore(cacheDir)
	_ = store.write("testuser", "acme", "widget", cacheFilterKey(filter.Set{}), newFakeRepo())
	swr.clock = func() time.Time {
		return time.Now().Add(6 * time.Minute)
	}

	// Stale serve triggers refresh.
	_, _ = swr.FetchRepo(context.Background(), filter.Set{})

	// LingerWait should return quickly because the cap fires.
	start := time.Now()
	swr.LingerWait()
	elapsed := time.Since(start)

	if elapsed > 200*time.Millisecond {
		t.Fatalf("LingerWait took too long: %v (cap should have fired)", elapsed)
	}

	close(block)
	// Ensure the background goroutine finishes writing before t.TempDir cleanup.
	swr.LingerWait()
}

func TestSWR_InnerError_Propagated(t *testing.T) {
	cacheDir := t.TempDir()
	inner := &fakeInnerClient{err: errors.New("network error")}
	resolve := func() (repository.Repository, error) {
		return repository.Repository{Owner: "acme", Name: "widget"}, nil
	}
	swr := NewSWRClient(inner, resolve, cacheDir, 5*time.Minute)
	swr.accountID = func() string { return "testuser" }

	_, err := swr.FetchRepo(context.Background(), filter.Set{})
	if err == nil {
		t.Fatal("want error, got nil")
	}
}

func TestSWR_FilteredViewsUseDistinctCacheKeys(t *testing.T) {
	cacheDir := t.TempDir()
	inner := &fakeInnerClient{repo: newFakeRepo()}
	resolve := func() (repository.Repository, error) {
		return repository.Repository{Owner: "acme", Name: "widget"}, nil
	}
	swr := NewSWRClient(inner, resolve, cacheDir, 5*time.Minute)
	swr.accountID = func() string { return "testuser" }

	alice := filter.NewSet(
		[]filter.QueryFilter{filter.NewAuthorFilter([]string{"alice"})},
		nil,
	)
	bob := filter.NewSet(
		[]filter.QueryFilter{filter.NewAuthorFilter([]string{"bob"})},
		nil,
	)

	if _, err := swr.FetchRepo(context.Background(), alice); err != nil {
		t.Fatalf("alice FetchRepo: %v", err)
	}
	if _, err := swr.FetchRepo(context.Background(), bob); err != nil {
		t.Fatalf("bob FetchRepo: %v", err)
	}
	if inner.callCount.Load() != 2 {
		t.Fatalf("want 2 inner calls for distinct filter keys, got %d", inner.callCount.Load())
	}
}
