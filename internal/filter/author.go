package filter

import (
	"strings"

	"github.com/gertzgal/gh-prs/internal/model"
)

// AuthorFilter restricts results to PRs authored by specific GitHub users.
//
// When multiple logins are provided, GitHub's search engine ORs them: a PR
// with exactly one author matches if that author appears in the list. An
// empty Logins slice produces no qualifier fragments — the caller is
// responsible for injecting a sensible default (e.g. "@me") before
// constructing the filter.
type AuthorFilter struct {
	Logins []string
}

// NewAuthorFilter constructs an AuthorFilter for the given logins.
// Policy (e.g. substituting "@me" when the list is empty) is intentionally
// left to the caller so this package stays policy-blind.
func NewAuthorFilter(logins []string) AuthorFilter {
	return AuthorFilter{Logins: logins}
}

// QueryFragments implements QueryFilter. Each login becomes an independent
// "author:<login>" qualifier; GitHub ORs them in the search results.
func (f AuthorFilter) QueryFragments() []string {
	frags := make([]string, len(f.Logins))
	for i, login := range f.Logins {
		frags[i] = "author:" + login
	}
	return frags
}

// Apply implements ListFilter. Keeps PRs whose author (case-insensitive)
// matches any of the resolved logins. An empty Logins slice is a no-op.
func (f AuthorFilter) Apply(prs []model.PR) []model.PR {
	if len(f.Logins) == 0 {
		return prs
	}
	allowed := make(map[string]struct{}, len(f.Logins))
	for _, login := range f.Logins {
		allowed[strings.ToLower(login)] = struct{}{}
	}
	out := make([]model.PR, 0, len(prs))
	for _, pr := range prs {
		if _, ok := allowed[strings.ToLower(pr.Author)]; ok {
			out = append(out, pr)
		}
	}
	return out
}

// Label implements Labeler. Returns a compact display string for use in UI
// headers (e.g. "@alice, @bob").
//
// Returns "" when the filter is the bare @me sentinel — callers interpret
// an empty label as "nothing to override; show the viewer's real login
// instead." This is the only policy-aware behaviour in this package: @me
// always means the authenticated viewer, whoever that is, so displaying it
// literally ("@me") would be less informative than the viewer's actual login.
//
// Examples:
//
//	["@me"]         → ""           (fall back to viewer login)
//	["alice"]       → "@alice"
//	["alice","bob"] → "@alice, @bob"
func (f AuthorFilter) Label() string {
	if len(f.Logins) == 1 && f.Logins[0] == "@me" {
		return ""
	}
	parts := make([]string, len(f.Logins))
	for i, login := range f.Logins {
		if !strings.HasPrefix(login, "@") {
			login = "@" + login
		}
		parts[i] = login
	}
	return strings.Join(parts, ", ")
}
