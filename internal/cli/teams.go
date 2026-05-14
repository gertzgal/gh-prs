package cli

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/gertzgal/gh-prs/internal/github"
)

// teamResolveTimeout caps each team-resolution GraphQL call. Generous enough
// to cover paginated rosters on a slow link, tight enough to fail fast when
// the network is broken.
const teamResolveTimeout = 15 * time.Second

// resolveTeamsForCurrentRepo expands flags.Teams into GitHub logins using
// the current repo's owner as the org. Returns a sorted, de-duplicated
// slice. An empty Teams slice short-circuits with no error and no network.
//
// Cache reuse: when caching is enabled (flags.NoCache is false), wraps the
// resolver in cachedTeamResolver so repeated invocations within the TTL
// window skip the network entirely.
//
// Construction is duplicated from runOnce rather than threaded through Deps
// because team resolution happens before app.Run is called, and lifting it
// into app would put a network operation in the orchestration layer that
// AGENTS.md says should stay free of fetch policy.
func resolveTeamsForCurrentRepo(flags Flags, clientOpts github.Options, stderr io.Writer) ([]string, error) {
	if len(flags.Teams) == 0 {
		return nil, nil
	}

	cur, err := repository.Current()
	if err != nil {
		return nil, fmt.Errorf("--team requires a GitHub repo: %w", err)
	}

	resolver, err := github.NewTeamResolver(clientOpts)
	if err != nil {
		return nil, err
	}
	if !flags.NoCache {
		resolver = github.NewCachedTeamResolver(resolver, github.DefaultCacheDir(), github.DefaultTeamCacheTTL)
	}

	seen := make(map[string]struct{}, 32)
	for _, slug := range flags.Teams {
		ctx, cancel := context.WithTimeout(context.Background(), teamResolveTimeout)
		logins, err := resolver.ResolveTeam(ctx, cur.Owner, slug)
		cancel()
		if err != nil {
			return nil, fmt.Errorf("resolving team %q in %s: %w", slug, cur.Owner, err)
		}
		if flags.Debug {
			_, _ = fmt.Fprintf(stderr, "resolved team %s/%s -> %d members\n", cur.Owner, strings.ToLower(slug), len(logins))
		}
		for _, login := range logins {
			seen[login] = struct{}{}
		}
	}

	return sortedKeys(seen), nil
}

// unionLogins merges author logins with team-resolved logins, preserving
// the author order first (callers may use it as display order) and then
// appending any team-only logins in sorted order. Case-insensitive dedup.
func unionLogins(authors, teamLogins []string) []string {
	seen := make(map[string]struct{}, len(authors)+len(teamLogins))
	out := make([]string, 0, len(authors)+len(teamLogins))
	for _, a := range authors {
		key := strings.ToLower(a)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, a)
	}
	for _, t := range teamLogins {
		key := strings.ToLower(t)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, t)
	}
	return out
}

func sortedKeys(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
