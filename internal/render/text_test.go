package render

import (
	"strings"
	"testing"

	"github.com/gertzgal/gh-prs/internal/model"
)

func samplePR(overrides model.PR) model.PR {
	base := model.PR{
		Number:           1,
		Title:            "Sample PR title",
		URL:              "https://github.com/acme/widget/pull/1",
		IsDraft:          false,
		HeadRefName:      "feature/sample",
		BaseRefName:      "main",
		Additions:        10,
		Deletions:        2,
		ChangedFiles:     3,
		ReviewDecision:   "",
		CiState:          model.CiSuccess,
		MergeStateStatus: "CLEAN",
		Author:           "alice",
	}
	if overrides.Number != 0 {
		base.Number = overrides.Number
	}
	if overrides.Title != "" {
		base.Title = overrides.Title
	}
	if overrides.URL != "" {
		base.URL = overrides.URL
	}
	if overrides.HeadRefName != "" {
		base.HeadRefName = overrides.HeadRefName
	}
	if overrides.BaseRefName != "" {
		base.BaseRefName = overrides.BaseRefName
	}
	if overrides.IsDraft {
		base.IsDraft = true
	}
	if overrides.Author != "" {
		base.Author = overrides.Author
	}
	return base
}

func repoWith(prs []model.PR, rl *model.RateLimit) *model.Repo {
	return &model.Repo{
		Owner:         "acme",
		Name:          "widget",
		DefaultBranch: "main",
		ViewerLogin:   "alice",
		PRs:           prs,
		RateLimit:     rl,
	}
}

func TestText_StackOfTwo(t *testing.T) {
	prs := []model.PR{
		samplePR(model.PR{Number: 10, HeadRefName: "feat/a", BaseRefName: "main", Title: "Base of two-stack"}),
		samplePR(model.PR{Number: 11, HeadRefName: "feat/b", BaseRefName: "feat/a", Title: "Tip of two-stack"}),
	}
	out := mustFormat(t, Text{}, repoWith(prs, nil), Context{Color: false, OSC8: false, LatencyMs: 5})

	if !strings.Contains(out, "stack · 2 PRs") {
		t.Errorf("want stack · 2 PRs; got:\n%s", out)
	}
	if !strings.Contains(out, "  ┬ #10") {
		t.Errorf("want ┬ #10")
	}
	if !strings.Contains(out, "  └ #11") {
		t.Errorf("want └ #11")
	}
	if strings.Contains(out, "├") {
		t.Errorf("stack of two should not contain ├")
	}
	if !strings.Contains(out, " 1/2") {
		t.Errorf("want 1/2")
	}
	if !strings.Contains(out, " 2/2") {
		t.Errorf("want 2/2")
	}
}

func TestText_SingularPluralization(t *testing.T) {
	out := mustFormat(
		t, Text{},
		repoWith([]model.PR{samplePR(model.PR{Number: 42, Title: "Lonely PR"})}, nil),
		Context{Color: false, OSC8: false, LatencyMs: 5},
	)
	if !strings.Contains(out, "standalone · 1 PR") {
		t.Errorf("want singular PR; got:\n%s", out)
	}
	if strings.Contains(out, "standalone · 1 PRs") {
		t.Errorf("must not pluralize PR for count=1")
	}
	if strings.Contains(out, "stack ·") {
		t.Errorf("stack section should be absent")
	}
}

func TestText_DraftInStackDimmed(t *testing.T) {
	prs := []model.PR{
		samplePR(model.PR{Number: 20, HeadRefName: "feat/base", BaseRefName: "main", Title: "Base"}),
		samplePR(model.PR{Number: 21, HeadRefName: "feat/draft-tip", BaseRefName: "feat/base", Title: "Draft tip", IsDraft: true}),
	}
	out := mustFormat(t, Text{}, repoWith(prs, nil), Context{Color: true, OSC8: false, LatencyMs: 5})
	dimIdx := strings.Index(out, "\x1b[2m")
	leadIdx := strings.Index(out, "#20")
	draftIdx := strings.Index(out, "#21")
	branchIdx := strings.Index(out, "feat/draft-tip")
	if dimIdx < 0 {
		t.Fatalf("expected dim SGR, got:\n%s", out)
	}
	if dimIdx <= leadIdx || dimIdx >= draftIdx || draftIdx >= branchIdx {
		t.Errorf("unexpected order: dim=%d lead=%d draft=%d branch=%d", dimIdx, leadIdx, draftIdx, branchIdx)
	}
}

func TestText_BoldLeadOnly(t *testing.T) {
	prs := []model.PR{
		samplePR(model.PR{Number: 30, HeadRefName: "feat/lead", BaseRefName: "main", Title: "Lead PR title"}),
		samplePR(model.PR{Number: 31, HeadRefName: "feat/middle", BaseRefName: "feat/lead", Title: "Middle PR title"}),
		samplePR(model.PR{Number: 32, HeadRefName: "feat/tail", BaseRefName: "feat/middle", Title: "Tail PR title"}),
	}
	out := mustFormat(t, Text{}, repoWith(prs, nil), Context{Color: true, OSC8: false, LatencyMs: 5})
	if !strings.Contains(out, "\x1b[1mLead PR title\x1b[0m") {
		t.Errorf("lead title should be bold")
	}
	if strings.Contains(out, "\x1b[1mMiddle PR title\x1b[0m") {
		t.Errorf("middle title should not be bold")
	}
	if strings.Contains(out, "\x1b[1mTail PR title\x1b[0m") {
		t.Errorf("tail title should not be bold")
	}
}

func TestText_FooterNilRateLimit(t *testing.T) {
	out := mustFormat(t, Text{}, repoWith(nil, nil), Context{Color: false, OSC8: false, LatencyMs: 7, ShowStats: true})
	if !strings.Contains(out, "  7ms\n") {
		t.Errorf("want bare footer 7ms; got:\n%s", out)
	}
	if strings.Contains(out, "pt") {
		t.Errorf("footer should not contain pt")
	}
	if strings.Contains(out, "remaining") {
		t.Errorf("footer should not contain remaining")
	}
	if strings.Contains(out, "●") {
		t.Errorf("footer should not contain ●")
	}
}

func TestText_FooterWithRateLimit(t *testing.T) {
	rl := &model.RateLimit{Cost: 1, Remaining: 4655, ResetAt: "2026-04-17T20:00:00Z"}
	prs := []model.PR{samplePR(model.PR{})}
	out := mustFormat(t, Text{}, repoWith(prs, rl), Context{Color: false, OSC8: false, LatencyMs: 1408, ShowStats: true})
	if !strings.Contains(out, "  1408ms · ● 1pt · 4655 remaining") {
		t.Errorf("want compact footer; got:\n%s", out)
	}
}

func TestText_FooterHiddenByDefault(t *testing.T) {
	rl := &model.RateLimit{Cost: 1, Remaining: 4655, ResetAt: "2026-04-17T20:00:00Z"}
	prs := []model.PR{samplePR(model.PR{})}
	out := mustFormat(t, Text{}, repoWith(prs, rl), Context{Color: false, OSC8: false, LatencyMs: 1408})

	for _, needle := range []string{"1408ms", "1pt", "4655 remaining", "●"} {
		if strings.Contains(out, needle) {
			t.Errorf("footer leaked without ShowStats: found %q in:\n%s", needle, out)
		}
	}
}

// ---------------------------------------------------------------------------
// Multi-author section rendering
// ---------------------------------------------------------------------------

func TestText_MultiAuthor_SectionsGroupedByAuthor(t *testing.T) {
	prs := []model.PR{
		samplePR(model.PR{Number: 100, Title: "Alice PR 1", Author: "alice", HeadRefName: "alice/a", BaseRefName: "main"}),
		samplePR(model.PR{Number: 101, Title: "Bob PR 1", Author: "bob", HeadRefName: "bob/b", BaseRefName: "main"}),
	}
	ctx := Context{Color: false, OSC8: false, AuthorOrder: []string{"alice", "bob"}}
	out := mustFormat(t, Text{}, repoWith(prs, nil), ctx)

	if !strings.Contains(out, "@alice · 1 PR") {
		t.Errorf("want @alice · 1 PR header; got:\n%s", out)
	}
	if !strings.Contains(out, "@bob · 1 PR") {
		t.Errorf("want @bob · 1 PR header; got:\n%s", out)
	}
	// alice should appear before bob
	aliceIdx := strings.Index(out, "@alice")
	bobIdx := strings.Index(out, "@bob")
	if aliceIdx >= bobIdx {
		t.Errorf("alice section must precede bob section; alice=%d bob=%d in:\n%s", aliceIdx, bobIdx, out)
	}
}

func TestText_MultiAuthor_StackUnderRootAuthor(t *testing.T) {
	// alice has a stack of 2; bob has 1 standalone
	prs := []model.PR{
		samplePR(model.PR{Number: 10, Title: "Alice base", Author: "alice", HeadRefName: "alice/base", BaseRefName: "main"}),
		samplePR(model.PR{Number: 11, Title: "Alice tip", Author: "alice", HeadRefName: "alice/tip", BaseRefName: "alice/base"}),
		samplePR(model.PR{Number: 20, Title: "Bob feature", Author: "bob", HeadRefName: "bob/feat", BaseRefName: "main"}),
	}
	ctx := Context{Color: false, OSC8: false, AuthorOrder: []string{"alice", "bob"}}
	out := mustFormat(t, Text{}, repoWith(prs, nil), ctx)

	if !strings.Contains(out, "@alice · 2 PRs") {
		t.Errorf("want @alice · 2 PRs; got:\n%s", out)
	}
	if !strings.Contains(out, "@bob · 1 PR") {
		t.Errorf("want @bob · 1 PR; got:\n%s", out)
	}
	if !strings.Contains(out, "stack · 2 PRs") {
		t.Errorf("want stack · 2 PRs; got:\n%s", out)
	}
	if !strings.Contains(out, "standalone · 1 PR") {
		t.Errorf("want standalone · 1 PR; got:\n%s", out)
	}
	// alice's section (with the stack) must come before bob's section
	alicePos := strings.Index(out, "@alice")
	bobPos := strings.Index(out, "@bob")
	if alicePos >= bobPos {
		t.Errorf("alice section must precede bob; got positions alice=%d bob=%d", alicePos, bobPos)
	}
}

func TestText_MultiAuthor_AuthorOrderPreserved(t *testing.T) {
	// Ask for bob first, alice second. Use carol as the viewer (not alice/bob)
	// so the repo header "@carol" doesn't pollute position checks.
	repo := &model.Repo{
		Owner:         "acme",
		Name:          "widget",
		DefaultBranch: "main",
		ViewerLogin:   "carol",
		PRs: []model.PR{
			samplePR(model.PR{Number: 1, Title: "Alice PR", Author: "alice", HeadRefName: "alice/a", BaseRefName: "main"}),
			samplePR(model.PR{Number: 2, Title: "Bob PR", Author: "bob", HeadRefName: "bob/b", BaseRefName: "main"}),
		},
	}
	ctx := Context{Color: false, OSC8: false, AuthorOrder: []string{"bob", "alice"}}
	out := mustFormat(t, Text{}, repo, ctx)

	// Search for section headers specifically (include · to avoid false positives)
	bobIdx := strings.Index(out, "@bob ·")
	aliceIdx := strings.Index(out, "@alice ·")
	if bobIdx < 0 || aliceIdx < 0 {
		t.Fatalf("expected @bob · and @alice · section headers; got:\n%s", out)
	}
	if bobIdx >= aliceIdx {
		t.Errorf("bob section must precede alice when listed first in AuthorOrder; bob=%d alice=%d", bobIdx, aliceIdx)
	}
}

func TestText_MultiAuthor_MeResolvedToViewerLogin(t *testing.T) {
	// ViewerLogin is "alice"; @me in AuthorOrder should resolve to alice's section
	prs := []model.PR{
		samplePR(model.PR{Number: 5, Title: "Alice PR", Author: "alice", HeadRefName: "alice/feat", BaseRefName: "main"}),
		samplePR(model.PR{Number: 6, Title: "Bob PR", Author: "bob", HeadRefName: "bob/feat", BaseRefName: "main"}),
	}
	ctx := Context{Color: false, OSC8: false, AuthorOrder: []string{"@me", "bob"}}
	out := mustFormat(t, Text{}, repoWith(prs, nil), ctx)

	// @me → resolves to viewerLogin "alice"; header should show @alice
	if !strings.Contains(out, "@alice · 1 PR") {
		t.Errorf("want @alice · 1 PR (resolved from @me); got:\n%s", out)
	}
	if !strings.Contains(out, "@bob · 1 PR") {
		t.Errorf("want @bob · 1 PR; got:\n%s", out)
	}
}

func TestText_MultiAuthor_SingleAuthorFallsBackToFlatLayout(t *testing.T) {
	// len(AuthorOrder) == 1 → flat layout, no @author headers
	prs := []model.PR{
		samplePR(model.PR{Number: 42, Title: "Alice PR", Author: "alice"}),
	}
	ctx := Context{Color: false, OSC8: false, AuthorOrder: []string{"alice"}}
	out := mustFormat(t, Text{}, repoWith(prs, nil), ctx)

	if strings.Contains(out, "@alice ·") {
		t.Errorf("single AuthorOrder should not produce @author section headers; got:\n%s", out)
	}
	if !strings.Contains(out, "standalone · 1 PR") {
		t.Errorf("want flat standalone section; got:\n%s", out)
	}
}

func TestText_MultiAuthor_CaseInsensitiveLoginMatch(t *testing.T) {
	// GitHub preserves the original casing of logins (e.g. "OctoDev"), but the
	// --author flag is typically typed lowercase ("octodev"). The section must
	// still group correctly via case-insensitive matching.
	prs := []model.PR{
		samplePR(model.PR{Number: 99, Title: "Add widget endpoint", Author: "OctoDev", HeadRefName: "octodev/widget-endpoint", BaseRefName: "main"}),
	}
	ctx := Context{Color: false, OSC8: false, AuthorOrder: []string{"devbot", "octodev"}}
	repo := &model.Repo{
		Owner:         "acme",
		Name:          "widget",
		DefaultBranch: "main",
		ViewerLogin:   "devbot",
		PRs:           prs,
	}
	out := mustFormat(t, Text{}, repo, ctx)

	// Section header uses the flag-typed login ("octodev"), PR must appear under it
	if !strings.Contains(out, "@octodev · 1 PR") {
		t.Errorf("want @octodev · 1 PR (case-insensitive match); got:\n%s", out)
	}
	if !strings.Contains(out, "#99") {
		t.Errorf("PR #99 must be rendered; got:\n%s", out)
	}
}

func TestText_MultiAuthor_EmptyAuthorSectionOmitted(t *testing.T) {
	// carol has no PRs — her section should not appear
	prs := []model.PR{
		samplePR(model.PR{Number: 7, Title: "Alice PR", Author: "alice", HeadRefName: "alice/x", BaseRefName: "main"}),
	}
	ctx := Context{Color: false, OSC8: false, AuthorOrder: []string{"alice", "carol"}}
	out := mustFormat(t, Text{}, repoWith(prs, nil), ctx)

	if strings.Contains(out, "@carol") {
		t.Errorf("author with no PRs must not appear; got:\n%s", out)
	}
	if !strings.Contains(out, "@alice · 1 PR") {
		t.Errorf("want @alice · 1 PR; got:\n%s", out)
	}
}
