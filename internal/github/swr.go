package github

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/gertzgal/gh-prs/internal/filter"
	"github.com/gertzgal/gh-prs/internal/model"
)

const (
	defaultCacheTTL       = 5 * time.Minute
	hardExpireMultiplier  = 2
	defaultLingerCap      = 3 * time.Second
	defaultRefreshTimeout = 30 * time.Second
)

// SWRClient wraps a Client with stale-while-revalidate disk caching.
type SWRClient struct {
	inner      Client
	store      *swrStore
	resolve    func() (repository.Repository, error)
	accountID  func() string
	ttl        time.Duration
	lingerCap  time.Duration
	clock      func() time.Time
	mu         sync.Mutex
	refreshing map[string]bool
	wg         sync.WaitGroup
}

// NewSWRClient wraps the given client with SWR caching.
func NewSWRClient(inner Client, cacheDir string, ttl time.Duration) *SWRClient {
	if ttl <= 0 {
		ttl = defaultCacheTTL
	}
	return &SWRClient{
		inner:      inner,
		store:      newSWRStore(cacheDir),
		resolve:    repository.Current,
		accountID:  accountID,
		ttl:        ttl,
		lingerCap:  defaultLingerCap,
		clock:      time.Now,
		refreshing: make(map[string]bool),
	}
}

// FetchRepo implements Client. It serves from cache when possible and triggers
// background refreshes for stale entries.
func (c *SWRClient) FetchRepo(ctx context.Context, filters filter.Set) (*model.Repo, error) {
	cur, err := c.resolve()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", model.ErrRepoNotFound, err)
	}

	id := c.accountID()
	if id == "unknown" {
		// Can't determine account; skip caching entirely.
		return c.inner.FetchRepo(ctx, filters)
	}

	entry, storeErr := c.store.read(id, cur.Owner, cur.Name)
	if storeErr != nil {
		// Store read failed; fall back to fresh fetch and try to write.
		repo, err := c.inner.FetchRepo(ctx, filters)
		if err != nil {
			return nil, err
		}
		_ = c.store.write(id, cur.Owner, cur.Name, repo)
		return repo, nil
	}

	now := c.clock()
	var age time.Duration
	if entry != nil {
		age = now.Sub(entry.WrittenAt)
	}
	hardExpire := c.ttl * hardExpireMultiplier

	if entry == nil || age > hardExpire {
		// Cold miss or hard expired: blocking fetch.
		repo, err := c.inner.FetchRepo(ctx, filters)
		if err != nil {
			return nil, err
		}
		_ = c.store.write(id, cur.Owner, cur.Name, repo)
		return repo, nil
	}

	// Cache hit (warm or soft stale).
	repo := entry.Data.Clone()
	if repo.RateLimit != nil {
		repo.RateLimit.Cost = 0
	}
	repo.CacheAge = age
	repo.IsStale = age > c.ttl

	if repo.IsStale {
		// Soft stale: trigger background refresh.
		c.wg.Add(1)
		go c.refresh(id, cur.Owner, cur.Name, filters)
	}

	return repo, nil
}

func (c *SWRClient) refresh(id, owner, name string, filters filter.Set) {
	defer c.wg.Done()

	c.mu.Lock()
	key := fmt.Sprintf("%s/%s/%s", id, owner, name)
	if c.refreshing[key] {
		c.mu.Unlock()
		return
	}
	c.refreshing[key] = true
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		delete(c.refreshing, key)
		c.mu.Unlock()
	}()

	refreshCtx, cancel := context.WithTimeout(context.Background(), defaultRefreshTimeout)
	defer cancel()

	repo, err := c.inner.FetchRepo(refreshCtx, filters)
	if err != nil {
		return
	}
	_ = c.store.write(id, owner, name, repo)
}

// LingerWait blocks until all background refreshes complete or the linger cap
// fires. Call this after printing output and before exiting the process.
func (c *SWRClient) LingerWait() {
	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(c.lingerCap):
		// Linger cap fired. The refresh goroutine's context will cancel the
		// in-flight HTTP request, and the goroutine exits cleanly.
	}
}
