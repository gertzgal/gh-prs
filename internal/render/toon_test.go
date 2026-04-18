package render

import (
	"strings"
	"testing"

	"github.com/gertzgal/gh-prs/internal/model"
)

func TestTOON_EmitsTabularHeaderForNonEmpty(t *testing.T) {
	repo := loadRepo(t, "graphql-widget-4-stack")
	got := TOON{}.Format(repo, Context{LatencyMs: 0})

	if !strings.Contains(got, "prs[4]{") {
		t.Fatalf("expected tabular header prs[4]{...}, got:\n%s", got)
	}
	header := "prs[4]{number,title,url,isDraft,headRefName,baseRefName,additions,deletions,changedFiles,reviewDecision,ciState,mergeStateStatus,stackId,stackPos}:"
	if !strings.Contains(got, header) {
		t.Fatalf("header row mismatch.\nwant substring: %s\ngot:\n%s", header, got)
	}
}

func TestTOON_EmptyPRsEmitsEmptyArrayHeader(t *testing.T) {
	repo := loadRepo(t, "graphql-empty")
	got := TOON{}.Format(repo, Context{LatencyMs: 0})

	if !strings.Contains(got, "prs[0]:") {
		t.Fatalf("expected prs[0]: for empty, got:\n%s", got)
	}
	if strings.Contains(got, "prs[0]{") {
		t.Fatalf("empty array must not emit field list, got:\n%s", got)
	}
}

func TestTOON_StandalonePRsHaveNullStackFields(t *testing.T) {
	repo := loadRepo(t, "graphql-gadget-standalone")
	got := TOON{}.Format(repo, Context{LatencyMs: 0})

	// Every row should end with `,null,null` (stackId, stackPos both null).
	for _, line := range strings.Split(got, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "2") { // data rows start with PR numbers 2001,2002
			continue
		}
		if !strings.HasSuffix(trimmed, ",null,null") {
			t.Errorf("standalone row should end with ,null,null — got %q", trimmed)
		}
	}
}

func TestTOON_RateLimitEmittedAsNestedObject(t *testing.T) {
	repo := loadRepo(t, "graphql-widget-4-stack")
	got := TOON{}.Format(repo, Context{LatencyMs: 0})

	if !strings.Contains(got, "rateLimit:\n  cost: ") {
		t.Fatalf("expected nested rateLimit block, got:\n%s", got)
	}
}

func TestTOON_LatencyIncluded(t *testing.T) {
	repo := loadRepo(t, "graphql-widget-4-stack")
	got := TOON{}.Format(repo, Context{LatencyMs: 475})

	if !strings.Contains(got, "latencyMs: 475") {
		t.Fatalf("expected latencyMs: 475, got:\n%s", got)
	}
}

func TestTOON_EndsWithNewline(t *testing.T) {
	repo := loadRepo(t, "graphql-widget-4-stack")
	got := TOON{}.Format(repo, Context{LatencyMs: 0})

	if !strings.HasSuffix(got, "\n") {
		t.Fatalf("output must end with newline")
	}
}

func TestTOON_ColorDoesNotAffectOutput(t *testing.T) {
	repo := loadRepo(t, "graphql-widget-4-stack")
	withColor := TOON{}.Format(repo, Context{Color: true, OSC8: true, LatencyMs: 42})
	withoutColor := TOON{}.Format(repo, Context{Color: false, OSC8: false, LatencyMs: 42})

	if withColor != withoutColor {
		t.Fatalf("color/osc8 must not affect TOON output")
	}
}

func TestTOON_NullReviewDecisionAndCiState(t *testing.T) {
	repo := &model.Repo{
		Owner:         "o",
		Name:          "n",
		DefaultBranch: "main",
		ViewerLogin:   "u",
		PRs: []model.PR{
			{Number: 1, Title: "t", URL: "http://x", HeadRefName: "feat", BaseRefName: "main"},
		},
	}
	got := TOON{}.Format(repo, Context{})

	// Single-row table — the row should contain `,null,null,` for reviewDecision
	// and ciState sandwich before mergeStateStatus.
	if !strings.Contains(got, ",null,null,") {
		t.Fatalf("expected null cells for empty review/ci, got:\n%s", got)
	}
}

func TestTOON_TokenEfficiencyVsJSON(t *testing.T) {
	repo := loadRepo(t, "graphql-widget-4-stack")
	toonOut := TOON{}.Format(repo, Context{LatencyMs: 0})
	jsonOut := JSON{}.Format(repo, Context{LatencyMs: 0})

	if len(toonOut) >= len(jsonOut) {
		t.Fatalf("TOON (%d) should be smaller than JSON (%d)", len(toonOut), len(jsonOut))
	}
	// Sanity check the advertised ~40% reduction isn't wildly off for our 4-PR
	// fixture (it's currently ~52%). Regression guard — tighten if it drifts.
	if float64(len(toonOut))/float64(len(jsonOut)) > 0.75 {
		t.Fatalf("TOON/JSON ratio %.2f is worse than expected (~0.5)",
			float64(len(toonOut))/float64(len(jsonOut)))
	}
}
