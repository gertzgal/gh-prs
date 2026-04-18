package render

import "github.com/gertzgal/gh-prs/internal/model"

type Context struct {
	Color     bool
	OSC8      bool
	LatencyMs int
	// ShowStats toggles the compact stats footer ("Nms · ● Npt · N remaining").
	// Off by default: the line is noise for the common case and always visible
	// via `gh prs --stats`. Only consulted by the text formatter; the JSON
	// formatter always emits RateLimit as a structured field for machine
	// consumers regardless.
	ShowStats bool
}

type Formatter interface {
	Format(repo *model.Repo, ctx Context) string
}

type Name string

const (
	NameText Name = "text"
	NameJSON Name = "json"
)

type Text struct{}
type JSON struct{}

var _ Formatter = Text{}
var _ Formatter = JSON{}

func Formatters() map[Name]Formatter {
	return map[Name]Formatter{
		NameText: Text{},
		NameJSON: JSON{},
	}
}
