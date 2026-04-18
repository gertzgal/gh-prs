package cli

import (
	"fmt"
	"io"
	"strings"
)

// PrintRESTEquivalent writes the 9-line "DEBUG REST equivalent" block to stderr
// when --debug is set. Signature mirrors the TS debug-output.ts.
func PrintRESTEquivalent(owner, name string, stderr io.Writer) {
	p := "DEBUG REST equivalent:"
	lines := []string{
		p + " ─ fixed calls ────────────────────────────────────────────────",
		p + "   gh api user                                             # viewer",
		p + fmt.Sprintf("   gh api repos/%s/%s                                   # default branch", owner, name),
		p + fmt.Sprintf("   gh api \"search/issues?q=is:pr+is:open+author:@me+repo:%s/%s\"  # PR list", owner, name),
		p + " ─ per PR (run 3× for each authored open PR) ──────────────────",
		p + fmt.Sprintf("   gh api repos/%s/%s/pulls/<num>                        # additions/deletions", owner, name),
		p + fmt.Sprintf("   gh api repos/%s/%s/pulls/<num>/reviews                # reviewDecision", owner, name),
		p + fmt.Sprintf("   gh api repos/%s/%s/commits/<sha>/check-runs           # ciState", owner, name),
		p + " ─ total: 3 + 3N REST calls (vs 1 GraphQL call)",
	}
	fmt.Fprintln(stderr, strings.Join(lines, "\n"))
}
