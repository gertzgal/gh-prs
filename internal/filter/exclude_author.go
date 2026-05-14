package filter

import (
	"strings"

	"github.com/gertzgal/gh-prs/internal/model"
)

// ExcludeAuthorFilter removes PRs authored by any of the given GitHub logins
// from the result set. It composes with AuthorFilter: when both are present,
// exclusion wins (matching GitHub's "-author:" search semantics).
//
// An empty Logins slice is a no-op in every method. Policy decisions such as
// "treat empty as 'do not exclude anyone'" live in the caller, not here.
type ExcludeAuthorFilter struct {
	Logins []string
}

// NewExcludeAuthorFilter constructs an ExcludeAuthorFilter for the given logins.
func NewExcludeAuthorFilter(logins []string) ExcludeAuthorFilter {
	return ExcludeAuthorFilter{Logins: logins}
}

// QueryFragments implements QueryFilter. Each login becomes a "-author:<login>"
// qualifier; GitHub treats these as hard subtractions, so PRs by those authors
// are removed even when matched by a positive author:/team-derived author:
// qualifier.
func (f ExcludeAuthorFilter) QueryFragments() []string {
	frags := make([]string, len(f.Logins))
	for i, login := range f.Logins {
		frags[i] = "-author:" + login
	}
	return frags
}

// Apply implements ListFilter. Drops PRs whose author (case-insensitive)
// matches any login in the filter. Acts as a safety net for the query-time
// negation; an empty Logins slice is a no-op.
func (f ExcludeAuthorFilter) Apply(prs []model.PR) []model.PR {
	if len(f.Logins) == 0 {
		return prs
	}
	denied := make(map[string]struct{}, len(f.Logins))
	for _, login := range f.Logins {
		denied[strings.ToLower(login)] = struct{}{}
	}
	out := make([]model.PR, 0, len(prs))
	for _, pr := range prs {
		if _, blocked := denied[strings.ToLower(pr.Author)]; blocked {
			continue
		}
		out = append(out, pr)
	}
	return out
}

// Label implements Labeler. Renders excluded logins prefixed with "!@",
// e.g. "!@alice" or "!@alice, !@bob".
//
// Unlike AuthorFilter, the bare "@me" sentinel is NOT mapped to the empty
// string. Exclusion is always an explicit user act and must remain visible
// in the header — otherwise a user excluding their own PRs would see a
// header indistinguishable from the default.
func (f ExcludeAuthorFilter) Label() string {
	if len(f.Logins) == 0 {
		return ""
	}
	parts := make([]string, len(f.Logins))
	for i, login := range f.Logins {
		if !strings.HasPrefix(login, "@") {
			login = "@" + login
		}
		parts[i] = "!" + login
	}
	return strings.Join(parts, ", ")
}

// resolveMe implements the package-internal meResolver interface so that
// Set.ResolveAndApply substitutes the "@me" sentinel with the viewer login
// without any type-switch growth at the call site.
func (f ExcludeAuthorFilter) resolveMe(viewerLogin string) ListFilter {
	resolved := make([]string, len(f.Logins))
	for i, login := range f.Logins {
		if login == "@me" {
			resolved[i] = viewerLogin
		} else {
			resolved[i] = login
		}
	}
	return ExcludeAuthorFilter{Logins: resolved}
}
