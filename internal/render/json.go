package render

import (
	"encoding/json"

	"github.com/gertzgal/gh-prs/internal/model"
)

func (JSON) Format(repo *model.Repo, ctx Context) (string, error) {
	type out struct {
		*model.Repo
		LatencyMs  int  `json:"latencyMs"`
		CacheAgeMs int  `json:"cacheAgeMs"`
		FromCache  bool `json:"fromCache"`
		IsStale    bool `json:"isStale"`
	}
	b, err := json.MarshalIndent(out{
		Repo:       repo,
		LatencyMs:  ctx.LatencyMs,
		CacheAgeMs: int(repo.CacheAge.Milliseconds()),
		FromCache:  repo.CacheAge > 0,
		IsStale:    repo.IsStale,
	}, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b) + "\n", nil
}
