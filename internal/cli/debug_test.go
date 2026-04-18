package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestPrintRESTEquivalent(t *testing.T) {
	var buf bytes.Buffer
	PrintRESTEquivalent("acme", "widget", &buf)

	out := strings.TrimRight(buf.String(), "\n")
	lines := strings.Split(out, "\n")
	if len(lines) != 9 {
		t.Fatalf("got %d lines, want 9:\n%s", len(lines), out)
	}

	checks := []struct {
		idx      int
		contains []string
	}{
		{0, []string{"fixed calls"}},
		{1, []string{"gh api user"}},
		{2, []string{"gh api repos/acme/widget"}},
		{3, []string{"search/issues", "repo:acme/widget"}},
		{4, []string{"per PR"}},
		{5, []string{"pulls/<num>", "additions/deletions"}},
		{6, []string{"reviewDecision"}},
		{7, []string{"check-runs"}},
		{8, []string{"3 + 3N REST calls"}},
	}
	for _, c := range checks {
		for _, sub := range c.contains {
			if !strings.Contains(lines[c.idx], sub) {
				t.Errorf("line %d missing %q: %q", c.idx, sub, lines[c.idx])
			}
		}
	}
}
