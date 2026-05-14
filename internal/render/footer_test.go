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
