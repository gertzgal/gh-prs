package github

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	graphql "github.com/cli/shurcooL-graphql"
	"github.com/gertzgal/gh-prs/internal/model"
)

type Client interface {
	FetchRepo(ctx context.Context) (*model.Repo, error)
}

// Options configures the GitHub client. Zero-value is safe: no debug, no cache.
type Options struct {
	// Debug enables httpretty-style logging of the GraphQL request/response
	// (URL, headers, body, timing) to DebugOut. Honors DebugColor for ANSI.
	Debug      bool
	DebugOut   io.Writer
	DebugColor bool

	// EnableCache turns on go-gh's disk cache for the GraphQL POST. When on,
	// identical queries within CacheTTL are served from CacheDir without a
	// network round trip. CacheDir empty => go-gh default.
	EnableCache bool
	CacheTTL    time.Duration
	CacheDir    string
}

type githubClient struct {
	gql     *api.GraphQLClient
	resolve func() (repository.Repository, error)
}

func New(opts Options) (Client, error) {
	gql, err := api.NewGraphQLClient(buildClientOptions(opts))
	if err != nil {
		return nil, translateError(err)
	}
	return &githubClient{gql: gql, resolve: repository.Current}, nil
}

func buildClientOptions(opts Options) api.ClientOptions {
	co := api.ClientOptions{
		// LogIgnoreEnv: our --debug flag is the single source of truth. We do
		// not want GH_DEBUG to silently enable logging behind the user's back
		// (and vice versa: we want --debug to work even without GH_DEBUG).
		LogIgnoreEnv: true,
	}
	if opts.Debug && opts.DebugOut != nil {
		co.Log = opts.DebugOut
		co.LogVerboseHTTP = true
		co.LogColorize = opts.DebugColor
	}
	if opts.EnableCache {
		co.EnableCache = true
		if opts.CacheTTL > 0 {
			co.CacheTTL = opts.CacheTTL
		}
		if opts.CacheDir != "" {
			co.CacheDir = opts.CacheDir
		}
	}
	return co
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
