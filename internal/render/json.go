package render

import (
	"encoding/json"

	"github.com/gertzgal/gh-prs/internal/model"
)

func (JSON) Format(repo *model.Repo, ctx Context) (string, error) {
	type out struct {
		*model.Repo
		LatencyMs int `json:"latencyMs"`
	}
	b, err := json.MarshalIndent(out{Repo: repo, LatencyMs: ctx.LatencyMs}, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b) + "\n", nil
}
