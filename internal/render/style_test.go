package render

import (
	"strings"
	"testing"
)

func TestStyles_ColorOff_PassesThrough(t *testing.T) {
	s := newStyles(false)
	cases := []struct {
		name string
		fn   func(string) string
	}{
		{"gray", func(in string) string { return s.Gray.Render(in) }},
		{"green", func(in string) string { return s.Green.Render(in) }},
		{"red", func(in string) string { return s.Red.Render(in) }},
		{"yellow", func(in string) string { return s.Yellow.Render(in) }},
		{"brightYellow", func(in string) string { return s.BrightYellow.Render(in) }},
		{"purple", func(in string) string { return s.Purple.Render(in) }},
		{"bold", func(in string) string { return s.Bold.Render(in) }},
		{"dim", func(in string) string { return s.Dim.Render(in) }},
		{"reviewChip", func(in string) string { return s.ReviewChip.Render(in) }},
		{"changesChip", func(in string) string { return s.ChangesChip.Render(in) }},
	}
	for _, c := range cases {
		got := c.fn("hi")
		if strings.Contains(got, "\x1b[") {
			t.Errorf("%s: expected no ANSI when color off, got %q", c.name, got)
		}
		if !strings.Contains(got, "hi") {
			t.Errorf("%s: lost content, got %q", c.name, got)
		}
	}
}

func TestStyles_ColorOn_EmitsAnsi(t *testing.T) {
	s := newStyles(true)
	got := s.Green.Render("hi")
	if !strings.Contains(got, "\x1b[") {
		t.Errorf("expected ANSI escape, got %q", got)
	}
	if !strings.Contains(got, "hi") {
		t.Errorf("lost content, got %q", got)
	}
}

func TestStyles_RenderChip_LiteralBracketsWhenNoColor(t *testing.T) {
	s := newStyles(false)
	if got := s.renderChip(s.ReviewChip, "review"); got != "[review]" {
		t.Errorf("review chip no-color = %q, want [review]", got)
	}
	if got := s.renderChip(s.ChangesChip, "changes"); got != "[changes]" {
		t.Errorf("changes chip no-color = %q, want [changes]", got)
	}
}

func TestStyles_RenderChip_StyledWhenColor(t *testing.T) {
	s := newStyles(true)
	got := s.renderChip(s.ReviewChip, "review")
	if strings.HasPrefix(got, "[") {
		t.Errorf("color-on chip should NOT use literal brackets; got %q", got)
	}
	if !strings.Contains(got, "review") {
		t.Errorf("lost label; got %q", got)
	}
	if !strings.Contains(got, "\x1b[") {
		t.Errorf("expected ANSI; got %q", got)
	}
}
