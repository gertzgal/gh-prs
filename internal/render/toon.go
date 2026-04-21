package render

import (
	"github.com/gertzgal/gh-prs/internal/model"
	"github.com/toon-format/toon-go"
)

// toonPRRow is the per-PR shape emitted to TOON. All fields are primitive so the
// encoder collapses a []toonPRRow slice into the tabular header+rows form that
// gives TOON its token-efficiency advantage. Field order drives output column
// order and mirrors the JSON output for cross-format consistency.
//
// ReviewDecision and CiState are pointers because an empty string from a PR
// without a decision/check must emit as `null` rather than `""` — the encoder
// treats nil pointers as null while still keeping the row primitive for the
// tabular detector.
type toonPRRow struct {
	Number           int     `toon:"number"`
	Title            string  `toon:"title"`
	URL              string  `toon:"url"`
	IsDraft          bool    `toon:"isDraft"`
	HeadRefName      string  `toon:"headRefName"`
	BaseRefName      string  `toon:"baseRefName"`
	Additions        int     `toon:"additions"`
	Deletions        int     `toon:"deletions"`
	ChangedFiles     int     `toon:"changedFiles"`
	ReviewDecision   *string `toon:"reviewDecision"`
	CiState          *string `toon:"ciState"`
	MergeStateStatus string  `toon:"mergeStateStatus"`
	Author           string  `toon:"author"`
	StackID          *int    `toon:"stackId"`
	StackPos         *string `toon:"stackPos"`
}

type rateLimitView struct {
	Cost      int    `toon:"cost"`
	Remaining int    `toon:"remaining"`
	ResetAt   string `toon:"resetAt"`
}

type repoDoc struct {
	Owner         string         `toon:"owner"`
	Name          string         `toon:"name"`
	DefaultBranch string         `toon:"defaultBranch"`
	ViewerLogin   string         `toon:"viewerLogin"`
	PRs           []toonPRRow    `toon:"prs"`
	RateLimit     *rateLimitView `toon:"rateLimit"`
	LatencyMs     int            `toon:"latencyMs"`
	CacheAgeMs    int            `toon:"cacheAgeMs"`
	FromCache     bool           `toon:"fromCache"`
	IsStale       bool           `toon:"isStale"`
}

func (TOON) Format(repo *model.Repo, ctx Context) (string, error) {
	rows := make([]toonPRRow, len(repo.PRs))
	for i, p := range repo.PRs {
		rows[i] = toPRRow(p)
	}

	doc := repoDoc{
		Owner:         repo.Owner,
		Name:          repo.Name,
		DefaultBranch: repo.DefaultBranch,
		ViewerLogin:   repo.ViewerLogin,
		PRs:           rows,
		RateLimit:     toRateLimitView(repo.RateLimit),
		LatencyMs:     ctx.LatencyMs,
		CacheAgeMs:    int(repo.CacheAge.Milliseconds()),
		FromCache:     repo.CacheAge > 0,
		IsStale:       repo.IsStale,
	}

	out, err := toon.MarshalString(doc, toon.WithIndent(2))
	if err != nil {
		return "", err
	}
	if len(out) == 0 || out[len(out)-1] != '\n' {
		out += "\n"
	}
	return out, nil
}

func toPRRow(p model.PR) toonPRRow {
	return toonPRRow{
		Number:           p.Number,
		Title:            p.Title,
		URL:              p.URL,
		IsDraft:          p.IsDraft,
		HeadRefName:      p.HeadRefName,
		BaseRefName:      p.BaseRefName,
		Additions:        p.Additions,
		Deletions:        p.Deletions,
		ChangedFiles:     p.ChangedFiles,
		ReviewDecision:   nilOrString(string(p.ReviewDecision)),
		CiState:          nilOrString(string(p.CiState)),
		MergeStateStatus: p.MergeStateStatus,
		Author:           p.Author,
		StackID:          p.StackID,
		StackPos:         p.StackPos,
	}
}

func toRateLimitView(rl *model.RateLimit) *rateLimitView {
	if rl == nil {
		return nil
	}
	return &rateLimitView{
		Cost:      rl.Cost,
		Remaining: rl.Remaining,
		ResetAt:   rl.ResetAt,
	}
}

func nilOrString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
