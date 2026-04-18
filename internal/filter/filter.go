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
