package render

import "github.com/gertzgal/gh-prs/internal/model"

type Context struct {
	Color     bool
	OSC8      bool
	LatencyMs int
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
