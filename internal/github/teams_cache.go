package github

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DefaultTeamCacheTTL is the freshness window for team membership lookups.
// Team rosters change rarely (org admin action), so a long TTL keeps the
// fast path instant for repeated invocations while still picking up changes
// within a working day.
const DefaultTeamCacheTTL = 24 * time.Hour

const teamCacheVersion = 1

// cachedTeamResolver wraps another TeamResolver with a disk-backed cache.
// Cache hits serve immediately and skip the network. Cache misses delegate
// to the inner resolver and persist the result. Write failures are best-
// effort: a corrupt or unwritable cache must never block a working lookup.
type cachedTeamResolver struct {
	inner     TeamResolver
	store     *teamStore
	accountID func() string
	ttl       time.Duration
	clock     func() time.Time
}

// NewCachedTeamResolver wraps inner with a disk-backed cache rooted under
// cacheDir. A non-positive ttl falls back to DefaultTeamCacheTTL.
func NewCachedTeamResolver(inner TeamResolver, cacheDir string, ttl time.Duration) TeamResolver {
	if ttl <= 0 {
		ttl = DefaultTeamCacheTTL
	}
	return &cachedTeamResolver{
		inner:     inner,
		store:     newTeamStore(cacheDir),
		accountID: accountID,
		ttl:       ttl,
		clock:     time.Now,
	}
}

func (c *cachedTeamResolver) ResolveTeam(ctx context.Context, org, slug string) ([]string, error) {
	org = strings.ToLower(strings.TrimSpace(org))
	slug = strings.ToLower(strings.TrimSpace(slug))

	id := c.accountID()
	if id == "unknown" {
		// No stable identity to scope the cache; bypass entirely.
		return c.inner.ResolveTeam(ctx, org, slug)
	}

	if entry, err := c.store.read(id, org, slug); err == nil && entry != nil {
		if c.clock().Sub(entry.WrittenAt) < c.ttl {
			out := make([]string, len(entry.Logins))
			copy(out, entry.Logins)
			return out, nil
		}
	}

	logins, err := c.inner.ResolveTeam(ctx, org, slug)
	if err != nil {
		return nil, err
	}
	_ = c.store.write(id, org, slug, logins)
	return logins, nil
}

// teamStore handles disk I/O for the team-membership cache. Kept separate
// from swrStore because the data shape and lifecycle (long TTL, no SWR
// background refresh) are different enough that conflating them would force
// shared code to carry two policies at once.
type teamStore struct {
	baseDir string
}

func newTeamStore(baseDir string) *teamStore {
	return &teamStore{baseDir: baseDir}
}

func (s *teamStore) path(accountID, org, slug string) string {
	safe := func(v string) string {
		return strings.ReplaceAll(strings.ReplaceAll(v, "/", "_"), "..", "_")
	}
	return filepath.Join(
		s.baseDir,
		"teams",
		safe(accountID),
		fmt.Sprintf("%s_%s.json", safe(org), safe(slug)),
	)
}

func (s *teamStore) read(accountID, org, slug string) (*teamEntry, error) {
	p := s.path(accountID, org, slug)
	raw, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var entry teamEntry
	if err := json.Unmarshal(raw, &entry); err != nil {
		return nil, nil // corrupt file, treat as miss
	}
	if entry.Version != teamCacheVersion {
		return nil, nil // old format, treat as miss
	}
	return &entry, nil
}

func (s *teamStore) write(accountID, org, slug string, logins []string) error {
	p := s.path(accountID, org, slug)
	if err := os.MkdirAll(filepath.Dir(p), 0750); err != nil {
		return err
	}
	entry := teamEntry{
		Version:   teamCacheVersion,
		WrittenAt: time.Now(),
		Org:       org,
		Slug:      slug,
		Logins:    logins,
	}
	raw, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	return os.WriteFile(p, raw, 0640)
}

// teamEntry is the on-disk envelope for cached team membership.
type teamEntry struct {
	Version   int       `json:"v"`
	WrittenAt time.Time `json:"t"`
	Org       string    `json:"org"`
	Slug      string    `json:"slug"`
	Logins    []string  `json:"logins"`
}
