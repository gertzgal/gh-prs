package render

import (
	"strings"
	"testing"
)

func TestSgrToggle(t *testing.T) {
	cases := []struct {
		name string
		fn   func(string, bool) string
		code string
	}{
		{"green", fgGreen, "\x1b[32m"},
		{"red", fgRed, "\x1b[31m"},
		{"yellow", fgYellow, "\x1b[33m"},
		{"gray", fgGray, "\x1b[90m"},
		{"brightYellow", fgBrightYellow, "\x1b[93m"},
		{"bold", styleBold, "\x1b[1m"},
		{"dim", styleDim, "\x1b[2m"},
	}
	for _, c := range cases {
		if got := c.fn("hi", false); got != "hi" {
			t.Errorf("%s off = %q, want bare %q", c.name, got, "hi")
		}
		got := c.fn("hi", true)
		if !strings.HasPrefix(got, c.code) {
			t.Errorf("%s on = %q, want prefix %q", c.name, got, c.code)
		}
		if !strings.HasSuffix(got, "\x1b[0m") {
			t.Errorf("%s on = %q, want suffix reset", c.name, got)
		}
		if !strings.Contains(got, "hi") {
			t.Errorf("%s on = %q, want substring hi", c.name, got)
		}
	}
}

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
