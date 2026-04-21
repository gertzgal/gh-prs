// Package filter provides a composable, extensible filtering layer for PR
// queries. Filters are pure data structures with no side effects — policy
// (e.g. "default to @me") belongs in the caller (CLI layer), not here.
//
// New filter types are added by implementing QueryFilter or ListFilter; no
// existing code needs to change (Open/Closed principle). Each interface is
// intentionally narrow (Interface Segregation): a filter only implements
// what it actually does.
package filter

import "github.com/gertzgal/gh-prs/internal/model"

// QueryFilter contributes zero, one, or many GitHub search qualifier
// fragments that are appended to the issues-search query string.
//
// Examples of valid fragments: "author:alice", "author:alice author:bob".
// Returning an empty slice means this filter adds no qualifier.
type QueryFilter interface {
	QueryFragments() []string
}

// ListFilter post-processes a []model.PR slice after it has been fetched.
// Useful for logic that cannot be expressed as a GitHub search qualifier
// (e.g. filtering by stack position or local metadata).
type ListFilter interface {
	Apply(prs []model.PR) []model.PR
}

// Labeler is an optional interface a filter may implement to contribute a
// short human-readable label for display in UI contexts (e.g. the header).
//
// Returning an empty string signals "nothing to show" — the caller falls back
// to its own default (e.g. the viewer's real login). This is the designated
// mechanism for the @me sentinel: AuthorFilter{["@me"]}.Label() == "".
//
// Labeler is intentionally separate from QueryFilter and ListFilter so that
// filters which have no meaningful display label (e.g. an internal date-range
// filter) don't have to implement it.
type Labeler interface {
	Label() string
}

// Set is an immutable, zero-safe collection of filters to apply to a
// PR fetch operation. The zero value applies no filtering.
type Set struct {
	queries []QueryFilter
	lists   []ListFilter
}

// NewSet constructs a Set. Defensive copies are made so that mutations to
// the slices passed by the caller do not affect the returned Set.
func NewSet(queries []QueryFilter, lists []ListFilter) Set {
	q := make([]QueryFilter, len(queries))
	copy(q, queries)
	l := make([]ListFilter, len(lists))
	copy(l, lists)
	return Set{queries: q, lists: l}
}

// QueryFragments collects and flattens all fragments from every QueryFilter
// in the set. The result is ready to be appended to a GitHub search string.
func (s Set) QueryFragments() []string {
	var frags []string
	for _, f := range s.queries {
		frags = append(frags, f.QueryFragments()...)
	}
	return frags
}

// Apply runs every ListFilter in order, passing the output of each as the
// input to the next. Returns the final filtered slice.
func (s Set) Apply(prs []model.PR) []model.PR {
	for _, f := range s.lists {
		prs = f.Apply(prs)
	}
	return prs
}

// ResolveAndApply clones the Set, resolves any "@me" sentinel in AuthorFilter
// logins to the provided viewerLogin, then runs all ListFilters in order.
// The caller (app.Run) owns the policy of what "@me" means.
func (s Set) ResolveAndApply(prs []model.PR, viewerLogin string) []model.PR {
	lists := make([]ListFilter, len(s.lists))
	for i, f := range s.lists {
		if af, ok := f.(AuthorFilter); ok {
			resolved := make([]string, len(af.Logins))
			for j, login := range af.Logins {
				if login == "@me" {
					resolved[j] = viewerLogin
				} else {
					resolved[j] = login
				}
			}
			lists[i] = AuthorFilter{Logins: resolved}
			continue
		}
		lists[i] = f
	}
	return NewSet(s.queries, lists).Apply(prs)
}

// Label collects non-empty labels from every QueryFilter that implements
// Labeler and joins them with " · ".
//
// Order is deterministic because NewSet preserves insertion order and the
// CLI constructs the Set in a fixed sequence (author first, future filters
// appended). No sorting is applied: the construction site owns the order.
func (s Set) Label() string {
	var parts []string
	for _, q := range s.queries {
		if l, ok := q.(Labeler); ok {
			if v := l.Label(); v != "" {
				parts = append(parts, v)
			}
		}
	}
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for _, p := range parts[1:] {
		result += " · " + p
	}
	return result
}
