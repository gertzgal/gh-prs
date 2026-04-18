package github

import (
	"encoding/json"
	"fmt"

	"github.com/gertzgal/gh-prs/internal/model"
)

// DecodeForTests decodes a GraphQL response envelope captured as JSON
// (matching the shape of testdata/fixtures/*.json) into a *model.Repo.
// Exported solely so internal/render tests can drive golden comparisons.
// Do not use in production code.
func DecodeForTests(raw []byte) (*model.Repo, error) {
	var env struct {
		Data *struct {
			RateLimit  *model.RateLimit
			Viewer     struct{ Login string }
			Repository *struct {
				DefaultBranchRef *struct{ Name string }
			}
			Search struct {
				Nodes []struct {
					Number           int
					Title            string
					URL              string `json:"url"`
					IsDraft          bool
					HeadRefName      string
					BaseRefName      string
					Additions        int
					Deletions        int
					ChangedFiles     int
					ReviewDecision   model.ReviewDecision
					MergeStateStatus string
					Author           struct{ Login string }
					Commits          struct {
						Nodes []struct {
							Commit struct {
								StatusCheckRollup *struct{ State model.CiState }
							}
						}
					}
				}
			}
		}
		Errors []struct{ Message string }
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, fmt.Errorf("decode fixture: %w", err)
	}
	if len(env.Errors) > 0 {
		return nil, &model.GhError{Msg: "GraphQL error: " + env.Errors[0].Message}
	}
	if env.Data == nil || env.Data.Repository == nil {
		return nil, &model.GhError{Msg: "fixture missing repository"}
	}
	defaultBranch := "main"
	if env.Data.Repository.DefaultBranchRef != nil {
		defaultBranch = env.Data.Repository.DefaultBranchRef.Name
	}
	prs := make([]model.PR, 0, len(env.Data.Search.Nodes))
	for _, n := range env.Data.Search.Nodes {
		var ci model.CiState
		if len(n.Commits.Nodes) > 0 && n.Commits.Nodes[0].Commit.StatusCheckRollup != nil {
			ci = n.Commits.Nodes[0].Commit.StatusCheckRollup.State
		}
		prs = append(prs, model.PR{
			Number:           n.Number,
			Title:            n.Title,
			URL:              n.URL,
			IsDraft:          n.IsDraft,
			HeadRefName:      n.HeadRefName,
			BaseRefName:      n.BaseRefName,
			Additions:        n.Additions,
			Deletions:        n.Deletions,
			ChangedFiles:     n.ChangedFiles,
			ReviewDecision:   n.ReviewDecision,
			CiState:          ci,
			MergeStateStatus: n.MergeStateStatus,
			Author:           n.Author.Login,
		})
	}
	return &model.Repo{
		Owner:         "acme-org",
		Name:          "widget",
		DefaultBranch: defaultBranch,
		ViewerLogin:   env.Data.Viewer.Login,
		PRs:           prs,
		RateLimit:     env.Data.RateLimit,
	}, nil
}
