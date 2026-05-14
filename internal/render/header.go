package render

import (
	"fmt"
	"strings"

	"github.com/gertzgal/gh-prs/internal/model"
)

// renderHeader returns the two-line header block: a command-echo line
// (suppressed when Command is empty) and a metadata strip. Both lines are
// rendered in meta.secondary (gray).
func renderHeader(repo *model.Repo, ctx Context, s styles) string {
	var lines []string

	if ctx.Command != "" {
		lines = append(lines, s.Gray.Render("› "+ctx.Command))
	}

	parts := []string{
		"repo " + fmt.Sprintf("%s/%s", repo.Owner, repo.Name),
		"base " + repo.DefaultBranch,
	}
	if ctx.FilterLabel != "" {
		parts = append(parts, ctx.FilterLabel)
	}
	lines = append(lines, s.Gray.Render(strings.Join(parts, " · ")))

	return strings.Join(lines, "\n")
}
