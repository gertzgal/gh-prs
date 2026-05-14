package render

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
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
	Adds       int
	Dels       int
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
			for cur := node; cur != nil; cur = cur.Child {
				sections[i].Adds += cur.PR.Additions
				sections[i].Dels += cur.PR.Deletions
			}
		}
	}
	for _, pr := range g.Standalone {
		if i, ok := idx[lowerKey(pr.Author)]; ok {
			sections[i].Standalone = append(sections[i].Standalone, pr)
			sections[i].Adds += pr.Additions
			sections[i].Dels += pr.Deletions
		}
	}
	return sections
}

// sectionPRCount returns the total number of PRs in an author section,
// counting both stacked PRs (walking each chain) and standalone PRs.
func sectionPRCount(sec authorSection) int {
	n := 0
	for _, node := range sec.Stacks {
		n += len(flattenStack(node))
	}
	return n + len(sec.Standalone)
}

// authorHeaderLine renders the per-author header with `left` on the left,
// `right` right-aligned within the row when width >= 80, or stacked on two
// lines below that threshold. Both halves are already styled.
func authorHeaderLine(left, right string, width int, s styles) string {
	leftStyled := "  " + s.Gray.Render(left)
	if width < 80 {
		return leftStyled + "\n  " + right
	}
	gap := max(1, width-lipgloss.Width(leftStyled)-lipgloss.Width(right))
	return leftStyled + strings.Repeat(" ", gap) + right
}

// sectionDivider renders a dotted horizontal rule. Truncates to width-2 when
// width > 0 to leave breathing room; falls back to 80 dots otherwise — 80
// matches the narrow-terminal threshold used elsewhere in the layout.
func sectionDivider(width int, s styles) string {
	n := width - 2
	if n <= 0 {
		n = 80
	}
	return "  " + s.Gray.Render(strings.Repeat("·", n))
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
	if legend := renderLegend(ctx.Width, s); legend != "" {
		out = append(out, legend, "")
	}

	if len(ctx.AuthorOrder) > 1 {
		// Multi-author mode: one @login · N PRs header per author, then that
		// author's stacks and standalone sub-sections.
		sections := groupByAuthor(g, ctx.AuthorOrder)
		// Identify the last non-empty section so the divider only appears
		// between sections that will actually render — never trailing.
		lastNonEmpty := -1
		for i, sec := range sections {
			if sectionPRCount(sec) > 0 {
				lastNonEmpty = i
			}
		}
		for i, sec := range sections {
			prCount := sectionPRCount(sec)
			if prCount == 0 {
				continue
			}
			left := fmt.Sprintf("@%s · %d %s", sec.Login, prCount, pluralPR(prCount))
			// Totals are secondary — the author label is the dominant element.
			// Numbers carry diff color; the +/- sign-prefixes stay in meta gray
			// so the totals don't visually shout louder than the author label.
			right := s.Gray.Render("+") + s.Green.Render(fmt.Sprintf("%d", sec.Adds)) +
				" " + s.Gray.Render("-") + s.Red.Render(fmt.Sprintf("%d", sec.Dels))
			authorHeader := authorHeaderLine(left, right, ctx.Width, s)
			out = append(out, authorHeader, "")
			out = append(out, renderSections(sec.Stacks, sec.Standalone, s, ctx.OSC8)...)
			if i < lastNonEmpty {
				out = append(out, sectionDivider(ctx.Width, s), "")
			}
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
