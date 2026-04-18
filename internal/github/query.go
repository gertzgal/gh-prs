package github

import (
	"github.com/gertzgal/gh-prs/internal/model"
)

type prSearchQuery struct {
	RateLimit struct {
		Cost      int
		Remaining int
		ResetAt   string
	}
	Viewer struct {
		Login string
	}
	Repository *struct {
		DefaultBranchRef *struct {
			Name string
		}
	} `graphql:"repository(owner: $owner, name: $name)"`
	Search struct {
		Nodes []struct {
			PullRequest struct {
				Number           int
				Title            string
				URL              string `graphql:"url"`
				IsDraft          bool
				HeadRefName      string
				BaseRefName      string
				Additions        int
				Deletions        int
				ChangedFiles     int
				ReviewDecision   string
				MergeStateStatus string
				Author           struct {
					Login    string
					Typename string `graphql:"__typename"`
				}
				Commits struct {
					Nodes []struct {
						Commit struct {
							StatusCheckRollup *struct {
								State string
							}
						}
					}
				} `graphql:"commits(last: 1)"`
			} `graphql:"... on PullRequest"`
		}
	} `graphql:"search(query: $q, type: ISSUE, first: 50)"`
}

// authorLogin returns the canonical GitHub login for a PR author.
//
// GitHub's GraphQL API returns the login of Bot actors without the "[bot]"
// suffix (e.g. "github-actions" instead of "github-actions[bot]"), while
// every other GitHub surface — the web UI, the REST search API, and the
// --author flag — uses the suffixed form. Appending it when __typename is
// "Bot" makes model.PR.Author consistent with what users type and see.
func authorLogin(login, typename string) string {
	if typename == "Bot" {
		return login + "[bot]"
	}
	return login
}

func translateQueryResult(q *prSearchQuery, owner, name string) *model.Repo {
	defaultBranch := "main"
	if q.Repository != nil && q.Repository.DefaultBranchRef != nil {
		defaultBranch = q.Repository.DefaultBranchRef.Name
	}

	prs := make([]model.PR, 0, len(q.Search.Nodes))
	for _, n := range q.Search.Nodes {
		node := n.PullRequest
		var ci model.CiState
		if len(node.Commits.Nodes) > 0 && node.Commits.Nodes[0].Commit.StatusCheckRollup != nil {
			ci = model.CiState(node.Commits.Nodes[0].Commit.StatusCheckRollup.State)
		}
		prs = append(prs, model.PR{
			Number:           node.Number,
			Title:            node.Title,
			URL:              node.URL,
			IsDraft:          node.IsDraft,
			HeadRefName:      node.HeadRefName,
			BaseRefName:      node.BaseRefName,
			Additions:        node.Additions,
			Deletions:        node.Deletions,
			ChangedFiles:     node.ChangedFiles,
			ReviewDecision:   model.ReviewDecision(node.ReviewDecision),
			CiState:          ci,
			MergeStateStatus: node.MergeStateStatus,
			Author:           authorLogin(node.Author.Login, node.Author.Typename),
		})
	}

	var rl *model.RateLimit
	if q.RateLimit.ResetAt != "" || q.RateLimit.Cost != 0 || q.RateLimit.Remaining != 0 {
		rl = &model.RateLimit{
			Cost:      q.RateLimit.Cost,
			Remaining: q.RateLimit.Remaining,
			ResetAt:   q.RateLimit.ResetAt,
		}
	}

	return &model.Repo{
		Owner:         owner,
		Name:          name,
		DefaultBranch: defaultBranch,
		ViewerLogin:   q.Viewer.Login,
		PRs:           prs,
		RateLimit:     rl,
	}
}
