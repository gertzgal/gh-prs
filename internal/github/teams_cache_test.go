package github

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

type stubResolver struct {
	calls   atomic.Int32
	logins  []string
	err     error
	lastOrg string
}

func (s *stubResolver) ResolveTeam(_ context.Context, org, _ string) ([]string, error) {
	s.calls.Add(1)
	s.lastOrg = org
	if s.err != nil {
		return nil, s.err
	}
	out := make([]string, len(s.logins))
	copy(out, s.logins)
	return out, nil
}

func newCachedFor(t *testing.T, inner TeamResolver, ttl time.Duration) *cachedTeamResolver {
	t.Helper()
	c := &cachedTeamResolver{
		inner:     inner,
		store:     newTeamStore(t.TempDir()),
		accountID: func() string { return "testuser" },
		ttl:       ttl,
		clock:     time.Now,
	}
	return c
}

func TestCachedResolver_FirstCallHitsInnerSecondServesFromCache(t *testing.T) {
	stub := &stubResolver{logins: []string{"alice", "bob"}}
	cached := newCachedFor(t, stub, time.Hour)

	got1, err := cached.ResolveTeam(context.Background(), "acme-org", "widgets")
	if err != nil {
		t.Fatalf("first ResolveTeam: %v", err)
	}
	if stub.calls.Load() != 1 {
		t.Errorf("inner calls after first: want 1, got %d", stub.calls.Load())
	}
	if !sameStrings(got1, []string{"alice", "bob"}) {
		t.Errorf("first call result: want [alice bob], got %v", got1)
	}

	got2, err := cached.ResolveTeam(context.Background(), "acme-org", "widgets")
	if err != nil {
		t.Fatalf("second ResolveTeam: %v", err)
	}
	if stub.calls.Load() != 1 {
		t.Errorf("inner calls after second: want still 1 (cache hit), got %d", stub.calls.Load())
	}
	if !sameStrings(got2, []string{"alice", "bob"}) {
		t.Errorf("second call result: want [alice bob], got %v", got2)
	}
}

func TestCachedResolver_ExpiredEntryRefetches(t *testing.T) {
	stub := &stubResolver{logins: []string{"alice"}}
	cached := newCachedFor(t, stub, 50*time.Millisecond)

	if _, err := cached.ResolveTeam(context.Background(), "acme-org", "widgets"); err != nil {
		t.Fatalf("first ResolveTeam: %v", err)
	}

	// Advance the cache's notion of "now" past the TTL.
	written := time.Now()
	cached.clock = func() time.Time { return written.Add(time.Hour) }

	if _, err := cached.ResolveTeam(context.Background(), "acme-org", "widgets"); err != nil {
		t.Fatalf("second ResolveTeam: %v", err)
	}
	if stub.calls.Load() != 2 {
		t.Errorf("inner calls after expiry: want 2, got %d", stub.calls.Load())
	}
}

func TestCachedResolver_DifferentSlugIsSeparateEntry(t *testing.T) {
	stub := &stubResolver{logins: []string{"alice"}}
	cached := newCachedFor(t, stub, time.Hour)

	if _, err := cached.ResolveTeam(context.Background(), "acme-org", "widgets"); err != nil {
		t.Fatalf("widgets: %v", err)
	}
	if _, err := cached.ResolveTeam(context.Background(), "acme-org", "gadgets"); err != nil {
		t.Fatalf("gadgets: %v", err)
	}
	if stub.calls.Load() != 2 {
		t.Errorf("inner calls: want 2 (different slugs), got %d", stub.calls.Load())
	}
}

func TestCachedResolver_SlugCaseInsensitive(t *testing.T) {
	stub := &stubResolver{logins: []string{"alice"}}
	cached := newCachedFor(t, stub, time.Hour)

	if _, err := cached.ResolveTeam(context.Background(), "ACME-ORG", "Widgets"); err != nil {
		t.Fatalf("first: %v", err)
	}
	if _, err := cached.ResolveTeam(context.Background(), "acme-org", "widgets"); err != nil {
		t.Fatalf("second: %v", err)
	}
	if stub.calls.Load() != 1 {
		t.Errorf("inner calls: want 1 (case-insensitive hit), got %d", stub.calls.Load())
	}
}

func TestCachedResolver_ErrorDoesNotCache(t *testing.T) {
	stub := &stubResolver{err: errors.New("network down")}
	cached := newCachedFor(t, stub, time.Hour)

	if _, err := cached.ResolveTeam(context.Background(), "acme-org", "widgets"); err == nil {
		t.Fatal("want error, got nil")
	}
	if _, err := cached.ResolveTeam(context.Background(), "acme-org", "widgets"); err == nil {
		t.Fatal("want error on second call, got nil")
	}
	if stub.calls.Load() != 2 {
		t.Errorf("inner calls: want 2 (errors not cached), got %d", stub.calls.Load())
	}
}

func TestCachedResolver_UnknownAccountBypassesCache(t *testing.T) {
	stub := &stubResolver{logins: []string{"alice"}}
	cached := newCachedFor(t, stub, time.Hour)
	cached.accountID = func() string { return "unknown" }

	if _, err := cached.ResolveTeam(context.Background(), "acme-org", "widgets"); err != nil {
		t.Fatalf("first: %v", err)
	}
	if _, err := cached.ResolveTeam(context.Background(), "acme-org", "widgets"); err != nil {
		t.Fatalf("second: %v", err)
	}
	if stub.calls.Load() != 2 {
		t.Errorf("inner calls: want 2 (unknown account bypasses cache), got %d", stub.calls.Load())
	}
}
