package render

import (
	"strings"
	"testing"

	"github.com/gertzgal/gh-prs/internal/model"
)

func TestCiStatus(t *testing.T) {
	cases := []struct {
		state model.CiState
		plain string
	}{
		{model.CiSuccess, "✓ ci"},
		{model.CiFailure, "✗ ci"},
		{model.CiError, "✗ ci"},
		{model.CiPending, "● ci"},
		{model.CiExpected, "● ci"},
		{"", "○ ci"},
	}
	for _, c := range cases {
		if got := ciStatus(c.state, newStyles(false)); got != c.plain {
			t.Errorf("ciStatus(%q, false) = %q, want %q", c.state, got, c.plain)
		}
		colored := ciStatus(c.state, newStyles(true))
		if !strings.Contains(colored, "\x1b[") {
			t.Errorf("ciStatus(%q, true) = %q, want ANSI escape", c.state, colored)
		}
		if !strings.Contains(colored, "ci") {
			t.Errorf("ciStatus(%q, true) = %q, want substring ci", c.state, colored)
		}
	}
}

func TestAdditions(t *testing.T) {
	p := model.PR{Additions: 123, Deletions: 45}
	if got := additions(p, newStyles(false)); got != "+123-45" {
		t.Errorf("additions plain = %q, want %q", got, "+123-45")
	}
	colored := additions(p, newStyles(true))
	if !strings.Contains(colored, "+123") {
		t.Errorf("colored = %q, want substring +123", colored)
	}
	if !strings.Contains(colored, "-45") {
		t.Errorf("colored = %q, want substring -45", colored)
	}
	if !strings.Contains(colored, "\x1b[") {
		t.Errorf("colored = %q, want ANSI escape", colored)
	}
	if strings.Contains(colored, "/") {
		t.Errorf("colored = %q, should not contain slash", colored)
	}
}
