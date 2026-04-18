package render

import (
	"encoding/json"

	"github.com/gertzgal/gh-prs/internal/model"
	"github.com/gertzgal/gh-prs/internal/stacks"
)

func (JSON) Format(repo *model.Repo, ctx Context) string {
	annotated := *repo
	annotated.PRs = stacks.Annotate(repo.PRs, repo.DefaultBranch)

	type out struct {
		*model.Repo
		LatencyMs int `json:"latencyMs"`
	}
	b, err := json.MarshalIndent(out{Repo: &annotated, LatencyMs: ctx.LatencyMs}, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(b) + "\n"
}
