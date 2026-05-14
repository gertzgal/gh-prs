// Lipgloss styles, one per role in the visual contract. Color profile is
// chosen per-render-call so we can degrade to plain text when ctx.Color is
// false (NO_COLOR, non-TTY stdout, etc.).
//
// Color choices match the previous hand-rolled ANSI (basic 4-bit palette)
// so byte-level golden diffs remain interpretable: 90 (gray), 32 (green),
// 31 (red), 33 (yellow), 93 (bright yellow), plus 35 (purple) and 34 (blue)
// added by the redesign.
package render

import (
	"io"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

type styles struct {
	color        bool           // whether ANSI is emitted; gates literal-bracket fallback in chips
	Gray         lipgloss.Style // meta.secondary
	GrayDim      lipgloss.Style // meta.muted (drafts)
	Green        lipgloss.Style // ci.passed, diff.additions, stack.rail
	Red          lipgloss.Style // ci.failed, diff.deletions
	Yellow       lipgloss.Style // ci.pending
	BrightYellow lipgloss.Style // PR number
	Purple       lipgloss.Style // standalone.marker
	Bold         lipgloss.Style // emphasized titles
	Dim          lipgloss.Style // draft wrapper
	ReviewChip   lipgloss.Style // padded blue chip — reviewer action needed
	ChangesChip  lipgloss.Style // padded red chip — author action needed
}

// newStyles returns a fresh styles bundle. When color is false every style
// is no-op (returns input unchanged).
func newStyles(color bool) styles {
	profile := termenv.ANSI
	if !color {
		profile = termenv.Ascii
	}
	r := lipgloss.NewRenderer(io.Discard)
	r.SetColorProfile(profile)

	s := styles{color: color}
	s.Gray = r.NewStyle().Foreground(lipgloss.ANSIColor(8)) // bright black / gray
	s.GrayDim = s.Gray.Faint(true)
	s.Green = r.NewStyle().Foreground(lipgloss.ANSIColor(2))
	s.Red = r.NewStyle().Foreground(lipgloss.ANSIColor(1))
	s.Yellow = r.NewStyle().Foreground(lipgloss.ANSIColor(3))
	s.BrightYellow = r.NewStyle().Foreground(lipgloss.ANSIColor(11))
	s.Purple = r.NewStyle().Foreground(lipgloss.ANSIColor(5))
	s.Bold = r.NewStyle().Bold(true)
	s.Dim = r.NewStyle().Faint(true)
	s.ReviewChip = r.NewStyle().
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.ANSIColor(4)). // blue
		Padding(0, 1)
	s.ChangesChip = r.NewStyle().
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.ANSIColor(1)). // red
		Padding(0, 1)
	return s
}

// renderChip wraps `label` in either a padded background chip (color on)
// or the literal `[label]` form (color off). Callers must use this helper
// rather than calling .Render directly on a chip style so no-color output
// keeps the bracket distinctiveness for pipes, screenshots, and chat paste.
func (s styles) renderChip(style lipgloss.Style, label string) string {
	if !s.color {
		return "[" + label + "]"
	}
	return style.Render(label)
}
