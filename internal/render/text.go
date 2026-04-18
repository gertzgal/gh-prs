package render

import (
	"fmt"
	"strings"

	"github.com/gertzgal/gh-prs/internal/model"
	"github.com/gertzgal/gh-prs/internal/stacks"
)

// lowerKey normalises a login for case-insensitive section matching.
// GitHub logins are case-preserving but comparison is case-insensitive.
func lowerKey(s string) string { return strings.ToLower(s) }

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
		switch i {
		case 0:
			glyph = "┬"
		case n - 1:
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

func repoHeader(repo *model.Repo, ctx Context, opts renderOpts) string {
	subject := ctx.FilterLabel
	if subject == "" {
		subject = "@" + repo.ViewerLogin
	}
	text := fmt.Sprintf("%s/%s · %s · %s", repo.Owner, repo.Name, repo.DefaultBranch, subject)
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

// authorSection groups stacks and standalone PRs for a single author.
type authorSection struct {
	Login      string // resolved login (never "@me")
	Stacks     []*stacks.Node
	Standalone []model.PR
}

// groupByAuthor partitions stacks and standalone PRs by author, preserving
// the order of authorOrder. "@me" is resolved to viewerLogin. Stacks are
// attributed to the root PR's author. Matching is case-insensitive to handle
// the common case where the user types a lowercase login but GitHub preserves
// the original case (e.g. "chenalon" matches PR author "ChenAlon").
func groupByAuthor(g stacks.Grouped, authorOrder []string, viewerLogin string) []authorSection {
	sections := make([]authorSection, len(authorOrder))
	// idx maps the lowercase login to the section index.
	idx := make(map[string]int, len(authorOrder))
	for i, login := range authorOrder {
		if login == "@me" {
			login = viewerLogin
		}
		sections[i] = authorSection{Login: login}
		idx[lowerKey(login)] = i
	}
	for _, node := range g.Stacks {
		if i, ok := idx[lowerKey(node.PR.Author)]; ok {
			sections[i].Stacks = append(sections[i].Stacks, node)
		}
	}
	for _, pr := range g.Standalone {
		if i, ok := idx[lowerKey(pr.Author)]; ok {
			sections[i].Standalone = append(sections[i].Standalone, pr)
		}
	}
	return sections
}

func standaloneLayout() rowLayout {
	return rowLayout{
		titlePrefix:  "  ",
		branchPrefix: strings.Repeat(" ", 11),
	}
}

// renderSections emits lines for stacks + standalone within a single logical
// section (used for both single-author and per-author-group paths).
func renderSections(stackNodes []*stacks.Node, standalone []model.PR, opts renderOpts) []string {
	var out []string

	stackedCount := 0
	for _, s := range stackNodes {
		stackedCount += len(flattenStack(s))
	}
	if stackedCount > 0 {
		out = append(out, sectionLabel("stack", stackedCount, opts), "")
		for _, s := range stackNodes {
			out = append(out, renderStack(s, opts)...)
			out = append(out, "")
		}
	}
	if len(standalone) > 0 {
		out = append(out, sectionLabel("standalone", len(standalone), opts), "")
		sl := standaloneLayout()
		for i, p := range standalone {
			if i > 0 {
				out = append(out, "")
			}
			out = append(out, prRow(p, sl, opts))
		}
		out = append(out, "")
	}
	return out
}

func (Text) Format(repo *model.Repo, ctx Context) (string, error) {
	opts := renderOpts{color: ctx.Color, osc8: ctx.OSC8}
	g := stacks.Group(repo.PRs, repo.DefaultBranch)
	var out []string
	out = append(out, "", repoHeader(repo, ctx, opts), "")

	if len(ctx.AuthorOrder) > 1 {
		// Multi-author mode: one @login · N PRs header per author, then that
		// author's stacks and standalone sub-sections.
		sections := groupByAuthor(g, ctx.AuthorOrder, repo.ViewerLogin)
		for _, sec := range sections {
			stackedCount := 0
			for _, node := range sec.Stacks {
				stackedCount += len(flattenStack(node))
			}
			prCount := stackedCount + len(sec.Standalone)
			if prCount == 0 {
				continue
			}
			authorHeader := "  " + fgGray(fmt.Sprintf("@%s · %d %s", sec.Login, prCount, pluralPR(prCount)), opts.color)
			out = append(out, authorHeader, "")
			out = append(out, renderSections(sec.Stacks, sec.Standalone, opts)...)
		}
	} else {
		out = append(out, renderSections(g.Stacks, g.Standalone, opts)...)
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
