package render

import "testing"

func TestOsc8Link(t *testing.T) {
	if got := osc8Link("label", "https://example.com", false); got != "label" {
		t.Errorf("osc8 disabled = %q, want bare label", got)
	}
	got := osc8Link("label", "https://example.com", true)
	want := "\x1b]8;;https://example.com\x1b\\label\x1b]8;;\x1b\\"
	if got != want {
		t.Errorf("osc8 enabled = %q, want %q", got, want)
	}
}
