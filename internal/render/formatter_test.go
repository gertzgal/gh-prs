package render

import (
	"reflect"
	"testing"
)

func TestLookup_CanonicalNames(t *testing.T) {
	for _, name := range []string{FormatText, FormatJSON, FormatTOON} {
		if _, ok := Lookup(name); !ok {
			t.Errorf("Lookup(%q) = _, false; want a registered formatter", name)
		}
	}
}

func TestLookup_UnknownNameMiss(t *testing.T) {
	for _, name := range []string{"", "yaml", "TEXT", "tsv"} {
		if _, ok := Lookup(name); ok {
			t.Errorf("Lookup(%q) returned ok=true; want false", name)
		}
	}
}

func TestNames_ReturnsSortedCanonicalNames(t *testing.T) {
	got := Names()
	want := []string{FormatJSON, FormatText, FormatTOON} // alphabetical: json, text, toon
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Names() = %v, want %v", got, want)
	}
}

func TestRegistry_TypesSatisfyFormatter(t *testing.T) {
	// Compile-time guard enforced by var _ Formatter = ... declarations.
	// This test just confirms each registered value is non-nil.
	for name, f := range registry {
		if f == nil {
			t.Errorf("registry[%q] is nil", name)
		}
	}
}
