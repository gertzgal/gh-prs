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
	if !(dimIdx > leadIdx && dimIdx < draftIdx && draftIdx < branchIdx) {
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
