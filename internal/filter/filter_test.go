package filter_test

import (
	"reflect"
	"testing"

	"github.com/gertzgal/gh-prs/internal/filter"
	"github.com/gertzgal/gh-prs/internal/model"
)

// ---------------------------------------------------------------------------
// AuthorFilter
// ---------------------------------------------------------------------------

func TestAuthorFilter_EmptyLogins_ProducesNoFragments(t *testing.T) {
	f := filter.NewAuthorFilter(nil)
	got := f.QueryFragments()
	if len(got) != 0 {
		t.Fatalf("want empty fragments, got %v", got)
	}
}

func TestAuthorFilter_SingleLogin(t *testing.T) {
	f := filter.NewAuthorFilter([]string{"alice"})
	got := f.QueryFragments()
	want := []string{"author:alice"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestAuthorFilter_MeAlias(t *testing.T) {
	f := filter.NewAuthorFilter([]string{"@me"})
	got := f.QueryFragments()
	want := []string{"author:@me"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestAuthorFilter_MultipleLogins_ProducesOneFragmentPerLogin(t *testing.T) {
	f := filter.NewAuthorFilter([]string{"alice", "bob", "carol"})
	got := f.QueryFragments()
	want := []string{"author:alice", "author:bob", "author:carol"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

// ---------------------------------------------------------------------------
// Set — zero value
// ---------------------------------------------------------------------------

func TestSet_ZeroValue_QueryFragments_Empty(t *testing.T) {
	var s filter.Set
	if frags := s.QueryFragments(); len(frags) != 0 {
		t.Fatalf("zero Set should produce no fragments, got %v", frags)
	}
}

func TestSet_ZeroValue_Apply_Passthrough(t *testing.T) {
	var s filter.Set
	prs := []model.PR{{Number: 1}, {Number: 2}}
	got := s.Apply(prs)
	if !reflect.DeepEqual(got, prs) {
		t.Fatalf("zero Set Apply should be a no-op, got %v", got)
	}
}

// ---------------------------------------------------------------------------
// Set — QueryFragments
// ---------------------------------------------------------------------------

func TestSet_QueryFragments_SingleFilter(t *testing.T) {
	s := filter.NewSet(
		[]filter.QueryFilter{filter.NewAuthorFilter([]string{"@me"})},
		nil,
	)
	got := s.QueryFragments()
	want := []string{"author:@me"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestSet_QueryFragments_MultipleFilters_Flattened(t *testing.T) {
	// Two separate author filters whose fragments should be merged.
	s := filter.NewSet(
		[]filter.QueryFilter{
			filter.NewAuthorFilter([]string{"alice"}),
			filter.NewAuthorFilter([]string{"bob"}),
		},
		nil,
	)
	got := s.QueryFragments()
	want := []string{"author:alice", "author:bob"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestSet_QueryFragments_EmptyFilterProducesNoFragment(t *testing.T) {
	s := filter.NewSet(
		[]filter.QueryFilter{filter.NewAuthorFilter(nil)},
		nil,
	)
	got := s.QueryFragments()
	if len(got) != 0 {
		t.Fatalf("empty AuthorFilter should contribute no fragments, got %v", got)
	}
}

// ---------------------------------------------------------------------------
// Set — defensive copy: mutating the original slice must not affect the Set
// ---------------------------------------------------------------------------

func TestSet_DefensiveCopy_QueryFilters(t *testing.T) {
	a := filter.NewAuthorFilter([]string{"alice"})
	original := []filter.QueryFilter{a}
	s := filter.NewSet(original, nil)

	// Mutate the original slice after construction.
	original[0] = filter.NewAuthorFilter([]string{"mutated"})

	got := s.QueryFragments()
	want := []string{"author:alice"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Set was mutated by caller; got %v, want %v", got, want)
	}
}

// ---------------------------------------------------------------------------
// Set — ListFilter chaining
// ---------------------------------------------------------------------------

// stubListFilter keeps every PR whose Number is in the allow-list.
type stubListFilter struct{ allow []int }

func (f stubListFilter) Apply(prs []model.PR) []model.PR {
	out := prs[:0]
	for _, pr := range prs {
		for _, n := range f.allow {
			if pr.Number == n {
				out = append(out, pr)
				break
			}
		}
	}
	return out
}

func TestSet_Apply_ListFilter_FiltersSlice(t *testing.T) {
	prs := []model.PR{{Number: 1}, {Number: 2}, {Number: 3}}
	s := filter.NewSet(nil, []filter.ListFilter{stubListFilter{allow: []int{1, 3}}})
	got := s.Apply(prs)
	want := []model.PR{{Number: 1}, {Number: 3}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestSet_Apply_MultipleListFilters_Chained(t *testing.T) {
	prs := []model.PR{{Number: 1}, {Number: 2}, {Number: 3}, {Number: 4}}
	s := filter.NewSet(nil, []filter.ListFilter{
		stubListFilter{allow: []int{1, 2, 3}}, // drop 4
		stubListFilter{allow: []int{1, 3}},    // drop 2
	})
	got := s.Apply(prs)
	want := []model.PR{{Number: 1}, {Number: 3}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestSet_Apply_NilInput_Safe(t *testing.T) {
	var s filter.Set
	got := s.Apply(nil)
	if got != nil {
		t.Fatalf("want nil, got %v", got)
	}
}
