package github

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
	graphql "github.com/cli/shurcooL-graphql"
	"github.com/gertzgal/gh-prs/internal/model"
)

// TeamResolver expands a GitHub team into the set of member logins.
//
// Implementations must:
//   - Treat the slug case-insensitively (GitHub team slugs are lowercase).
//   - Return logins in deterministic order (sorted) so callers can compare
//     and cache results without spurious diffs.
//   - Wrap upstream errors with model.GhError so the CLI exit-code mapper
//     handles them like any other gh failure.
type TeamResolver interface {
	ResolveTeam(ctx context.Context, org, slug string) ([]string, error)
}

// ErrTeamNotFound is returned when GitHub responds successfully but the team
// does not exist (or the viewer cannot see it). Distinct from a transport
// error so the CLI can produce a precise, user-friendly message.
var ErrTeamNotFound = errors.New("team not found or not visible to viewer")

// NewTeamResolver builds a TeamResolver backed by the same GraphQL transport
// as the main PR client. Kept as a separate factory (rather than expanding
// Client) so PR-fetch consumers don't take on a transitive dependency on
// team-resolution code, and so callers that don't need teams pay nothing.
func NewTeamResolver(opts Options) (TeamResolver, error) {
	gql, err := api.NewGraphQLClient(buildClientOptions(opts))
	if err != nil {
		return nil, translateError(err)
	}
	return &graphqlTeamResolver{gql: gql}, nil
}

// newTeamResolverWith constructs a resolver with an injected GraphQL client.
// Reserved for tests; mirrors the newClientWith pattern used by the PR client.
func newTeamResolverWith(gql *api.GraphQLClient) TeamResolver {
	return &graphqlTeamResolver{gql: gql}
}

type graphqlTeamResolver struct {
	gql *api.GraphQLClient
}

// teamMembersPage is the per-page projection of the team-members query.
// Kept as a private type so the GraphQL schema details don't leak.
type teamMembersPage struct {
	Organization *struct {
		Team *struct {
			Members struct {
				PageInfo struct {
					HasNextPage bool
					EndCursor   string
				}
				Nodes []struct {
					Login string
				}
			} `graphql:"members(first: 100, after: $cursor)"`
		} `graphql:"team(slug: $slug)"`
	} `graphql:"organization(login: $org)"`
}

// ResolveTeam fetches all members of org/slug, paginating until exhausted.
// Returns a sorted, de-duplicated slice of logins. The slug is lowercased
// before the query: GitHub stores team slugs as lowercase and accepts only
// lowercase values, so "Backend" and "backend" must produce the same result.
func (r *graphqlTeamResolver) ResolveTeam(ctx context.Context, org, slug string) ([]string, error) {
	org = strings.ToLower(strings.TrimSpace(org))
	slug = strings.ToLower(strings.TrimSpace(slug))
	if org == "" || slug == "" {
		return nil, &model.GhError{Msg: "team resolver requires non-empty org and slug"}
	}

	seen := make(map[string]struct{}, 16)
	var cursor *string
	for {
		var page teamMembersPage
		vars := map[string]any{
			"org":    graphql.String(org),
			"slug":   graphql.String(slug),
			"cursor": (*graphql.String)(nil),
		}
		if cursor != nil {
			s := graphql.String(*cursor)
			vars["cursor"] = &s
		}
		if err := r.gql.QueryWithContext(ctx, "TeamMembers", &page, vars); err != nil {
			return nil, translateError(err)
		}
		if page.Organization == nil || page.Organization.Team == nil {
			return nil, fmt.Errorf("%w: %s/%s", ErrTeamNotFound, org, slug)
		}
		for _, n := range page.Organization.Team.Members.Nodes {
			if n.Login == "" {
				continue
			}
			seen[n.Login] = struct{}{}
		}
		if !page.Organization.Team.Members.PageInfo.HasNextPage {
			break
		}
		end := page.Organization.Team.Members.PageInfo.EndCursor
		if end == "" {
			break // defensive: malformed page info, avoid infinite loop
		}
		cursor = &end
	}

	out := make([]string, 0, len(seen))
	for login := range seen {
		out = append(out, login)
	}
	sort.Strings(out)
	return out, nil
}
