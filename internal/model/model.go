package model

import "encoding/json"

type ReviewDecision string

const (
	ReviewApproved         ReviewDecision = "APPROVED"
	ReviewChangesRequested ReviewDecision = "CHANGES_REQUESTED"
	ReviewRequired         ReviewDecision = "REVIEW_REQUIRED"
)

type CiState string

const (
	CiSuccess  CiState = "SUCCESS"
	CiFailure  CiState = "FAILURE"
	CiPending  CiState = "PENDING"
	CiError    CiState = "ERROR"
	CiExpected CiState = "EXPECTED"
)

type PR struct {
	Number           int            `json:"number"`
	Title            string         `json:"title"`
	URL              string         `json:"url"`
	IsDraft          bool           `json:"isDraft"`
	HeadRefName      string         `json:"headRefName"`
	BaseRefName      string         `json:"baseRefName"`
	Additions        int            `json:"additions"`
	Deletions        int            `json:"deletions"`
	ChangedFiles     int            `json:"changedFiles"`
	ReviewDecision   ReviewDecision `json:"reviewDecision"`
	CiState          CiState        `json:"ciState"`
	MergeStateStatus string         `json:"mergeStateStatus"`
	// 1-based stack index, nil for standalone. Populated by stacks.Annotate.
	StackID *int `json:"stackId"`
	// "i/N" position within the stack (e.g. "2/3"), nil for standalone.
	StackPos *string `json:"stackPos"`
}

func (p PR) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Number           int     `json:"number"`
		Title            string  `json:"title"`
		URL              string  `json:"url"`
		IsDraft          bool    `json:"isDraft"`
		HeadRefName      string  `json:"headRefName"`
		BaseRefName      string  `json:"baseRefName"`
		Additions        int     `json:"additions"`
		Deletions        int     `json:"deletions"`
		ChangedFiles     int     `json:"changedFiles"`
		ReviewDecision   any     `json:"reviewDecision"`
		CiState          any     `json:"ciState"`
		MergeStateStatus string  `json:"mergeStateStatus"`
		StackID          *int    `json:"stackId"`
		StackPos         *string `json:"stackPos"`
	}{
		Number:           p.Number,
		Title:            p.Title,
		URL:              p.URL,
		IsDraft:          p.IsDraft,
		HeadRefName:      p.HeadRefName,
		BaseRefName:      p.BaseRefName,
		Additions:        p.Additions,
		Deletions:        p.Deletions,
		ChangedFiles:     p.ChangedFiles,
		ReviewDecision:   nilIfEmpty(string(p.ReviewDecision)),
		CiState:          nilIfEmpty(string(p.CiState)),
		MergeStateStatus: p.MergeStateStatus,
		StackID:          p.StackID,
		StackPos:         p.StackPos,
	})
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

type RateLimit struct {
	Cost      int    `json:"cost"`
	Remaining int    `json:"remaining"`
	ResetAt   string `json:"resetAt"`
}

type Repo struct {
	Owner         string     `json:"owner"`
	Name          string     `json:"name"`
	DefaultBranch string     `json:"defaultBranch"`
	ViewerLogin   string     `json:"viewerLogin"`
	PRs           []PR       `json:"prs"`
	RateLimit     *RateLimit `json:"rateLimit"`
}
