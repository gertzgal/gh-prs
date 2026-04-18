package render

import (
	"fmt"
	"strings"

	"github.com/gertzgal/gh-prs/internal/model"
	"github.com/gertzgal/gh-prs/internal/stacks"
)

type rowLayout struct {
	titlePrefix  string
	branchPrefix string
	position     string
	titleBold    bool
}

type renderOpts struct {
	color bool
	osc8  bool
}

func prRow(pr model.PR, layout rowLayout, opts renderOpts) string {
	num := osc8Link(fmt.Sprintf("#%d", pr.Number), pr.URL, opts.osc8)
	numColored := fgBrightYellow(num, opts.color)
	ci := ciStatus(pr.CiState, opts.color)
	review := reviewStatus(pr.ReviewDecision, opts.color)
	diff := additions(pr, opts.color)
	title := pr.Title
	if layout.titleBold {
		title = styleBold(pr.Title, opts.color)
	}
	pos := ""
	if layout.position != "" {
		pos = "  " + fgGray(layout.position, opts.color)
	}

	titleLine := layout.titlePrefix + numColored + "  " + ci + "  " + review + "  " + diff + "  " + title + pos
	branchLine := layout.branchPrefix + fgGray(pr.HeadRefName, opts.color)
	row := titleLine + "\n" + branchLine
	if pr.IsDraft {
		return styleDim(row, opts.color)
	}
	return row
}

func flattenStack(node *stacks.Node) []model.PR {
	var out []model.PR
	for cur := node; cur != nil; cur = cur.Child {
		out = append(out, cur.PR)
	}
	return out
}

func renderStack(node *stacks.Node, opts renderOpts) []string {
	prs := flattenStack(node)
	n := len(prs)
	lines := make([]string, 0, n)
	for i, pr := range prs {
		var glyph string
		switch {
		case i == 0:
			glyph = "┬"
		case i == n-1:
			glyph = "└"
		default:
			glyph = "├"
		}
		connector := fgGray(glyph, opts.color)
		titlePrefix := "  " + connector + " "
		var branchPrefix string
		if i == n-1 {
			branchPrefix = strings.Repeat(" ", 13)
		} else {
			branchPrefix = "  " + fgGray("│", opts.color) + strings.Repeat(" ", 10)
		}
		lines = append(lines, prRow(pr, rowLayout{
			titlePrefix:  titlePrefix,
			branchPrefix: branchPrefix,
			position:     fmt.Sprintf("%d/%d", i+1, n),
			titleBold:    i == 0,
		}, opts))
	}
	return lines
}

func repoHeader(repo *model.Repo, opts renderOpts) string {
	text := fmt.Sprintf("%s/%s · %s · @%s", repo.Owner, repo.Name, repo.DefaultBranch, repo.ViewerLogin)
	return fgGray(text, opts.color)
}

func pluralPR(n int) string {
	if n == 1 {
		return "PR"
	}
	return "PRs"
}

func sectionLabel(kind string, n int, opts renderOpts) string {
	return "  " + fgGray(fmt.Sprintf("%s · %d %s", kind, n, pluralPR(n)), opts.color)
}

func (Text) Format(repo *model.Repo, ctx Context) (string, error) {
	opts := renderOpts{color: ctx.Color, osc8: ctx.OSC8}
	g := stacks.Group(repo.PRs, repo.DefaultBranch)
	var out []string
	out = append(out, "", repoHeader(repo, opts), "")

	stackedCount := 0
	for _, s := range g.Stacks {
		stackedCount += len(flattenStack(s))
	}
	if stackedCount > 0 {
		out = append(out, sectionLabel("stack", stackedCount, opts), "")
		for _, s := range g.Stacks {
			out = append(out, renderStack(s, opts)...)
			out = append(out, "")
		}
	}
	if len(g.Standalone) > 0 {
		out = append(out, sectionLabel("standalone", len(g.Standalone), opts), "")
		standaloneLayout := rowLayout{
			titlePrefix:  "  ",
			branchPrefix: strings.Repeat(" ", 11),
		}
		for i, p := range g.Standalone {
			if i > 0 {
				out = append(out, "")
			}
			out = append(out, prRow(p, standaloneLayout, opts))
		}
		out = append(out, "")
	}

	if ctx.ShowStats {
		footer := []string{fmt.Sprintf("%dms", ctx.LatencyMs)}
		if repo.RateLimit != nil {
			footer = append(footer, fmt.Sprintf("● %dpt", repo.RateLimit.Cost))
			footer = append(footer, fmt.Sprintf("%d remaining", repo.RateLimit.Remaining))
		}
		out = append(out, "  "+fgGray(strings.Join(footer, " · "), opts.color))
	}
	return strings.Join(out, "\n") + "\n", nil
}
