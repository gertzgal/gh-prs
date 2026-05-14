package render

import (
	"strings"
	"testing"

	"github.com/gertzgal/gh-prs/internal/model"
	"github.com/gertzgal/gh-prs/internal/stacks"
)

func TestSummaryFooter_MixedRepo(t *testing.T) {
	prs := []model.PR{
		// Stack of 2
		{Number: 1, Author: "a", HeadRefName: "feat/a-1", BaseRefName: "main", CiState: model.CiSuccess, Additions: 10, Deletions: 1},
		{Number: 2, Author: "a", HeadRefName: "feat/a-2", BaseRefName: "feat/a-1", CiState: model.CiPending, Additions: 5, Deletions: 0},
		// Standalone
		{Number: 3, Author: "b", HeadRefName: "feat/b-1", BaseRefName: "main", CiState: model.CiFailure, Additions: 1, Deletions: 1},
		{Number: 4, Author: "c", HeadRefName: "feat/c-1", BaseRefName: "main", CiState: model.CiSuccess, Additions: 2, Deletions: 0},
	}
	annotated := stacks.Annotate(prs, "main")
	repo := &model.Repo{DefaultBranch: "main", PRs: annotated}

	got := renderSummaryFooter(repo, newStyles(false))
	want := "4 PRs in 1 stack, 2 standalone   ✓ 2 passed  ● 1 pending  ✗ 1 failed   +18 -2"
	if !strings.Contains(got, want) {
		t.Errorf("footer mismatch.\ngot:  %q\nwant substring: %q", got, want)
	}
}

func TestSummaryFooter_Empty(t *testing.T) {
	repo := &model.Repo{DefaultBranch: "main", PRs: nil}
	got := renderSummaryFooter(repo, newStyles(false))
	want := "0 PRs in 0 stacks, 0 standalone   ✓ 0 passed  ● 0 pending  ✗ 0 failed   +0 -0"
	if !strings.Contains(got, want) {
		t.Errorf("empty-repo footer mismatch.\ngot:  %q\nwant substring: %q", got, want)
	}
}

// TestSummaryFooter_StackOfOneCountsAsStandalone exercises the invariant
// documented on renderSummaryFooter: a "stack" with only one PR must count
// toward standalone, not stacks. This is the same demotion rule applied
// during row rendering; the summary must stay in lock-step with it.
func TestSummaryFooter_StackOfOneCountsAsStandalone(t *testing.T) {
	// Construct a degenerate single-node stack by manually annotating: a PR
	// whose base points at another PR's head, but the parent is filtered out
	// in this test — meaning stacks.Group can produce a Stack with one node.
	// We simulate this by hand-crafting the Grouped result is not exported,
	// so instead use a base ref that isn't main but also isn't another PR.
	// The simpler equivalent: a single PR whose base equals another PR head
	// that ISN'T in the input. stacks.Group returns it as standalone in
	// practice, so we get the same observable behavior either way.
	prs := []model.PR{
		{Number: 100, Author: "a", HeadRefName: "feat/orphan", BaseRefName: "feat/missing-parent", CiState: model.CiSuccess, Additions: 7, Deletions: 0},
	}
	annotated := stacks.Annotate(prs, "main")
	repo := &model.Repo{DefaultBranch: "main", PRs: annotated}

	got := renderSummaryFooter(repo, newStyles(false))
	// Whether stacks.Group classifies the orphan as a 1-node stack or
	// directly as standalone, the summary must report it as standalone.
	want := "1 PR in 0 stacks, 1 standalone"
	if !strings.Contains(got, want) {
		t.Errorf("single-PR group must count as standalone in summary.\ngot:  %q\nwant substring: %q", got, want)
	}
}
