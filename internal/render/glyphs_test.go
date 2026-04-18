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
		if got := ciStatus(c.state, false); got != c.plain {
			t.Errorf("ciStatus(%q, false) = %q, want %q", c.state, got, c.plain)
		}
		colored := ciStatus(c.state, true)
		if !strings.Contains(colored, "\x1b[") {
			t.Errorf("ciStatus(%q, true) = %q, want ANSI escape", c.state, colored)
		}
		if !strings.Contains(colored, "ci") {
			t.Errorf("ciStatus(%q, true) = %q, want substring ci", c.state, colored)
		}
	}
}

func TestReviewStatus(t *testing.T) {
	cases := []struct {
		decision model.ReviewDecision
		plain    string
	}{
		{model.ReviewApproved, "✓ review"},
		{model.ReviewChangesRequested, "✗ review"},
		{model.ReviewRequired, "● review"},
		{"", "○ review"},
	}
	for _, c := range cases {
		if got := reviewStatus(c.decision, false); got != c.plain {
			t.Errorf("reviewStatus(%q, false) = %q, want %q", c.decision, got, c.plain)
		}
		colored := reviewStatus(c.decision, true)
		if !strings.Contains(colored, "\x1b[") {
			t.Errorf("reviewStatus(%q, true) = %q, want ANSI escape", c.decision, colored)
		}
		if !strings.Contains(colored, "review") {
			t.Errorf("reviewStatus(%q, true) = %q, want substring review", c.decision, colored)
		}
	}
}

func TestAdditions(t *testing.T) {
	p := model.PR{Additions: 123, Deletions: 45}
	if got := additions(p, false); got != "+123-45" {
		t.Errorf("additions plain = %q, want %q", got, "+123-45")
	}
	colored := additions(p, true)
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
