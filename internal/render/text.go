package render

import (
	"fmt"
	"strings"
	"time"

	"github.com/gertzgal/gh-prs/internal/model"
	"github.com/gertzgal/gh-prs/internal/stacks"
)

// lowerKey normalises a login for case-insensitive section matching.
// GitHub logins are case-preserving but comparison is case-insensitive.
func lowerKey(s string) string { return strings.ToLower(s) }

// formatShortDuration renders a duration as "2m", "30s", etc.
func formatShortDuration(d time.Duration) string {
	if d >= time.Minute {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%ds", int(d.Seconds()))
}

type rowLayout struct {
	titlePrefix  string
	branchPrefix string
	position     string
	titleBold    bool
}

func prRow(pr model.PR, layout rowLayout, s styles, osc8 bool) string {
	num := osc8Link(fmt.Sprintf("#%d", pr.Number), pr.URL, osc8)
	numColored := s.BrightYellow.Render(num)
	ci := ciStatus(pr.CiState, s)
	review := reviewStatus(pr.ReviewDecision, s)
	diff := additions(pr, s)
	title := pr.Title
	if layout.titleBold {
		title = s.Bold.Render(pr.Title)
	}
	pos := ""
	if layout.position != "" {
		pos = "  " + s.Gray.Render(layout.position)
	}

	titleLine := layout.titlePrefix + numColored + "  " + ci + "  " + review + "  " + diff + "  " + title + pos
	branchLine := layout.branchPrefix + s.Gray.Render(pr.HeadRefName)
	if pr.IsDraft {
		// Dim per-line, not over the joined row: lipgloss multi-line Render
		// pads short lines to max width, which would inject trailing spaces
		// and shift layout. The trade-off is two separate dim envelopes in
		// the output bytes — visually identical, but reflected in golden
		// files. Preserving layout is worth more than envelope count.
		return s.Dim.Render(titleLine) + "\n" + s.Dim.Render(branchLine)
	}
	return titleLine + "\n" + branchLine
}

func flattenStack(node *stacks.Node) []model.PR {
	var out []model.PR
	for cur := node; cur != nil; cur = cur.Child {
		out = append(out, cur.PR)
	}
	return out
}

func renderStack(node *stacks.Node, s styles, osc8 bool) []string {
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
		connector := s.Gray.Render(glyph)
		titlePrefix := "  " + connector + " "
		var branchPrefix string
		if i == n-1 {
			branchPrefix = strings.Repeat(" ", 13)
		} else {
			branchPrefix = "  " + s.Gray.Render("│") + strings.Repeat(" ", 10)
		}
		lines = append(lines, prRow(pr, rowLayout{
			titlePrefix:  titlePrefix,
			branchPrefix: branchPrefix,
			position:     fmt.Sprintf("%d/%d", i+1, n),
			titleBold:    i == 0,
		}, s, osc8))
	}
	return lines
}

func pluralPR(n int) string {
	if n == 1 {
		return "PR"
	}
	return "PRs"
}

func sectionLabel(kind string, n int, s styles) string {
	return "  " + s.Gray.Render(fmt.Sprintf("%s · %d %s", kind, n, pluralPR(n)))
}

// authorSection groups stacks and standalone PRs for a single author.
type authorSection struct {
	Login      string // resolved login (never "@me")
	Stacks     []*stacks.Node
	Standalone []model.PR
}

// groupByAuthor partitions stacks and standalone PRs by author, preserving
// the order of authorOrder. Logins are already resolved ("@me" substituted
// upstream in app.Run). Matching is case-insensitive to handle the common
// case where the user types a lowercase login but GitHub preserves the
// original case (e.g. "gertzgal" matches PR author "GertzGal").
func groupByAuthor(g stacks.Grouped, authorOrder []string) []authorSection {
	sections := make([]authorSection, len(authorOrder))
	// idx maps the lowercase login to the section index.
	idx := make(map[string]int, len(authorOrder))
	for i, login := range authorOrder {
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
func renderSections(stackNodes []*stacks.Node, standalone []model.PR, s styles, osc8 bool) []string {
	var out []string

	stackedCount := 0
	for _, sn := range stackNodes {
		stackedCount += len(flattenStack(sn))
	}
	if stackedCount > 0 {
		out = append(out, sectionLabel("stack", stackedCount, s), "")
		for _, sn := range stackNodes {
			out = append(out, renderStack(sn, s, osc8)...)
			out = append(out, "")
		}
	}
	if len(standalone) > 0 {
		out = append(out, sectionLabel("standalone", len(standalone), s), "")
		sl := standaloneLayout()
		for i, p := range standalone {
			if i > 0 {
				out = append(out, "")
			}
			out = append(out, prRow(p, sl, s, osc8))
		}
		out = append(out, "")
	}
	return out
}

func (Text) Format(repo *model.Repo, ctx Context) (string, error) {
	s := newStyles(ctx.Color)
	g := stacks.Group(repo.PRs, repo.DefaultBranch)
	var out []string
	out = append(out, "", renderHeader(repo, ctx, s), "")

	if len(ctx.AuthorOrder) > 1 {
		// Multi-author mode: one @login · N PRs header per author, then that
		// author's stacks and standalone sub-sections.
		sections := groupByAuthor(g, ctx.AuthorOrder)
		for _, sec := range sections {
			stackedCount := 0
			for _, node := range sec.Stacks {
				stackedCount += len(flattenStack(node))
			}
			prCount := stackedCount + len(sec.Standalone)
			if prCount == 0 {
				continue
			}
			authorHeader := "  " + s.Gray.Render(fmt.Sprintf("@%s · %d %s", sec.Login, prCount, pluralPR(prCount)))
			out = append(out, authorHeader, "")
			out = append(out, renderSections(sec.Stacks, sec.Standalone, s, ctx.OSC8)...)
		}
	} else {
		out = append(out, renderSections(g.Stacks, g.Standalone, s, ctx.OSC8)...)
	}

	if ctx.ShowStats {
		footer := []string{fmt.Sprintf("%dms", ctx.LatencyMs)}
		if repo.RateLimit != nil {
			footer = append(footer, fmt.Sprintf("● %dpt", repo.RateLimit.Cost))
			footer = append(footer, fmt.Sprintf("%d remaining", repo.RateLimit.Remaining))
		}
		if repo.CacheAge > 0 {
			age := formatShortDuration(repo.CacheAge)
			if repo.IsStale {
				footer = append(footer, fmt.Sprintf("stale %s ago", age))
			} else {
				footer = append(footer, fmt.Sprintf("cached %s ago", age))
			}
		}
		out = append(out, "  "+s.Gray.Render(strings.Join(footer, " · ")))
	}
	return strings.Join(out, "\n") + "\n", nil
}
