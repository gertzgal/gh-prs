package github

import (
	"context"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	graphql "github.com/cli/shurcooL-graphql"
	"github.com/gertzgal/gh-prs/internal/model"
)

type Client interface {
	FetchRepo(ctx context.Context) (*model.Repo, error)
}

type githubClient struct {
	gql     *api.GraphQLClient
	resolve func() (repository.Repository, error)
}

func New() (Client, error) {
	gql, err := api.DefaultGraphQLClient()
	if err != nil {
		return nil, translateError(err)
	}
	return &githubClient{gql: gql, resolve: repository.Current}, nil
}

func newClientWith(gql *api.GraphQLClient, resolve func() (repository.Repository, error)) Client {
	return &githubClient{gql: gql, resolve: resolve}
}

func (c *githubClient) FetchRepo(ctx context.Context) (*model.Repo, error) {
	cur, err := c.resolve()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", model.ErrRepoNotFound, err)
	}

	var q prSearchQuery
	vars := map[string]any{
		"owner": graphql.String(cur.Owner),
		"name":  graphql.String(cur.Name),
		"q":     graphql.String(fmt.Sprintf("is:pr is:open author:@me repo:%s/%s", cur.Owner, cur.Name)),
	}

	if err := c.gql.QueryWithContext(ctx, "PRsForViewer", &q, vars); err != nil {
		return nil, translateError(err)
	}
	return translateQueryResult(&q, cur.Owner, cur.Name), nil
}
