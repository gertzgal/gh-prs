package render

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// legendMinWidth is the minimum terminal width at which the legend is
// rendered. Below this threshold the legend is suppressed entirely so
// narrow terminals don't see a jumbled, broken box. width=0 (no TTY,
// unknown size) is treated as "wide": the legend renders and lipgloss
// is free to lay it out on a single line.
const legendMinWidth = 80

// renderLegend returns the bordered legend strip, or "" when width is
// below legendMinWidth. width=0 is treated as "wide" (no suppression,
// no wrap forcing). Between legendMinWidth and the natural rendered
// width of the token set (~155 cols) lipgloss .Width(...) soft-wraps
// the body inside the box. At wider widths the body fits on one line.
func renderLegend(width int, s styles) string {
	if width > 0 && width < legendMinWidth {
		return ""
	}

	tokens := []string{
		s.Green.Render("┬─●") + " stack",
		s.Purple.Render("●") + " standalone",
		s.Green.Render("✓") + " CI passed",
		s.Yellow.Render("●") + " CI pending",
		s.Red.Render("✗") + " CI failed",
		s.Gray.Render("○") + " CI unknown",
		s.Green.Render("+") + " additions",
		s.Red.Render("-") + " deletions",
		s.renderChip(s.ReviewChip, "review") + " reviewer",
		s.renderChip(s.ChangesChip, "changes") + " author",
	}

	body := strings.Join(tokens, "   ")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.ANSIColor(8)).
		Padding(0, 1)

	if width > 0 {
		// Inner width = total - 4 (2 border chars + 2 padding chars).
		box = box.Width(width - 4)
	}
	return box.Render(body)
}
