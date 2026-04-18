package render

import (
	"sort"

	"github.com/gertzgal/gh-prs/internal/model"
)

type Context struct {
	Color     bool
	OSC8      bool
	LatencyMs int
	// ShowStats toggles the compact stats footer ("Nms · ● Npt · N remaining").
	// Off by default: the line is noise for the common case and always visible
	// via `gh prs --stats`. Only consulted by the text formatter; machine
	// formats (json, toon) always emit RateLimit as a structured field.
	ShowStats bool
	// FilterLabel is a short human-readable summary of the active filters for
	// display in the repo header (e.g. "@alice, @bob"). An empty string means
	// "no override" — the header falls back to showing the viewer's real login.
	// Populated by filter.Set.Label() in the CLI layer; the render package has
	// no knowledge of the filter package.
	FilterLabel string
}

// Formatter renders a fetched repo into its target output form. Implementations
// return (output, nil) on success. A non-nil error indicates an encoder failure
// the caller must surface — the output string is then undefined and must not
// be written to stdout.
type Formatter interface {
	Format(repo *model.Repo, ctx Context) (string, error)
}

// FormatText, FormatJSON, and FormatTOON are the canonical format names used
// as registry keys and CLI flag values. Keep this list in sync with registry.
const (
	FormatText = "text"
	FormatJSON = "json"
	FormatTOON = "toon"
)

type Text struct{}
type JSON struct{}
type TOON struct{}

var _ Formatter = Text{}
var _ Formatter = JSON{}
var _ Formatter = TOON{}

// registry is the single source of truth for format-name → Formatter. Adding
// a new format means registering it here (and nothing else in the cli layer).
var registry = map[string]Formatter{
	FormatText: Text{},
	FormatJSON: JSON{},
	FormatTOON: TOON{},
}

// Lookup returns the Formatter for the given canonical name and whether it
// is registered.
func Lookup(name string) (Formatter, bool) {
	f, ok := registry[name]
	return f, ok
}

// Names returns the registered format names in stable (sorted) order. Used
// for help text and error messages.
func Names() []string {
	out := make([]string, 0, len(registry))
	for name := range registry {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}
