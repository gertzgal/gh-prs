// Always-on summary footer for the text formatter. Renders a single line
// with: PR count, stack count, standalone count, CI pass/pending/fail
// breakdown, and aggregate +adds/-dels totals. Sits above the optional
// `--stats` line emitted by Text.Format when ctx.ShowStats is true.
package render

import (
	"fmt"
	"strings"

	"github.com/gertzgal/gh-prs/internal/model"
	"github.com/gertzgal/gh-prs/internal/stacks"
)

// renderSummaryFooter returns the always-on summary line. It does not
// duplicate the optional --stats latency/rate/cache line, which the
// caller emits separately below this one.
//
// Stack count only includes chains with >= 2 PRs to match the stack-of-one
// fallback rule applied during section rendering — a single-node "stack"
// is demoted to standalone in the row output, so it must also count as
// standalone here for the summary to stay internally consistent.
func renderSummaryFooter(repo *model.Repo, s styles) string {
	g := stacks.Group(repo.PRs, repo.DefaultBranch)

	stackCount := 0
	stackedPRs := 0
	for _, node := range g.Stacks {
		flat := flattenStack(node)
		if len(flat) >= 2 {
			stackCount++
			stackedPRs += len(flat)
		}
	}
	standalonePRs := len(repo.PRs) - stackedPRs
	totalPRs := len(repo.PRs)

	var passed, pending, failed int
	var adds, dels int
	for _, pr := range repo.PRs {
		switch pr.CiState {
		case model.CiSuccess:
			passed++
		case model.CiFailure, model.CiError:
			failed++
		case model.CiPending, model.CiExpected:
			pending++
		}
		adds += pr.Additions
		dels += pr.Deletions
	}

	counts := fmt.Sprintf("%d %s in %d %s, %d standalone",
		totalPRs, pluralPR(totalPRs),
		stackCount, pluralStack(stackCount),
		standalonePRs,
	)
	ci := s.Green.Render("✓ "+fmt.Sprintf("%d passed", passed)) +
		"  " + s.Yellow.Render("● "+fmt.Sprintf("%d pending", pending)) +
		"  " + s.Red.Render("✗ "+fmt.Sprintf("%d failed", failed))
	totals := s.Green.Render(fmt.Sprintf("+%d", adds)) + " " + s.Red.Render(fmt.Sprintf("-%d", dels))

	return "  " + strings.Join([]string{counts, ci, totals}, "   ")
}

// pluralStack returns "stack" or "stacks" based on count.
func pluralStack(n int) string {
	if n == 1 {
		return "stack"
	}
	return "stacks"
}
