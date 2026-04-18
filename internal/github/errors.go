package github

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/gertzgal/gh-prs/internal/model"
)

func translateError(err error) error {
	if err == nil {
		return nil
	}

	var httpErr *api.HTTPError
	if errors.As(err, &httpErr) {
		return &model.GhError{
			Msg:    fmt.Sprintf("gh api returned HTTP %d", httpErr.StatusCode),
			Stderr: httpErr.Message,
		}
	}

	var gqlErr *api.GraphQLError
	if errors.As(err, &gqlErr) {
		msgs := make([]string, 0, len(gqlErr.Errors))
		for _, e := range gqlErr.Errors {
			msgs = append(msgs, e.Message)
		}
		return &model.GhError{Msg: "GraphQL error: " + strings.Join(msgs, "; ")}
	}

	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}

	return &model.GhError{Msg: err.Error()}
}
