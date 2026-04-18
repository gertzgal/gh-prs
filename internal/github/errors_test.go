package github

import (
	"context"
	"errors"
	"testing"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/gertzgal/gh-prs/internal/model"
)

func TestTranslateError_HTTPError(t *testing.T) {
	src := &api.HTTPError{StatusCode: 502, Message: "Bad Gateway"}
	out := translateError(src)

	var gh *model.GhError
	if !errors.As(out, &gh) {
		t.Fatalf("want *model.GhError, got %T: %v", out, out)
	}
	if gh.Msg != "gh api returned HTTP 502" {
		t.Errorf("Msg: want %q, got %q", "gh api returned HTTP 502", gh.Msg)
	}
	if gh.Stderr != "Bad Gateway" {
		t.Errorf("Stderr: want %q, got %q", "Bad Gateway", gh.Stderr)
	}
}

func TestTranslateError_GraphQLError(t *testing.T) {
	src := &api.GraphQLError{
		Errors: []api.GraphQLErrorItem{
			{Message: "first problem"},
			{Message: "second problem"},
		},
	}
	out := translateError(src)

	var gh *model.GhError
	if !errors.As(out, &gh) {
		t.Fatalf("want *model.GhError, got %T: %v", out, out)
	}
	want := "GraphQL error: first problem; second problem"
	if gh.Msg != want {
		t.Errorf("Msg: want %q, got %q", want, gh.Msg)
	}
}

func TestTranslateError_ContextCanceled(t *testing.T) {
	out := translateError(context.Canceled)
	if !errors.Is(out, context.Canceled) {
		t.Errorf("want context.Canceled as-is, got %T: %v", out, out)
	}
}

func TestTranslateError_ContextDeadline(t *testing.T) {
	out := translateError(context.DeadlineExceeded)
	if !errors.Is(out, context.DeadlineExceeded) {
		t.Errorf("want context.DeadlineExceeded as-is, got %T: %v", out, out)
	}
}

func TestTranslateError_Generic(t *testing.T) {
	src := errors.New("boom")
	out := translateError(src)

	var gh *model.GhError
	if !errors.As(out, &gh) {
		t.Fatalf("want *model.GhError, got %T: %v", out, out)
	}
	if gh.Msg != "boom" {
		t.Errorf("Msg: want %q, got %q", "boom", gh.Msg)
	}
}

func TestTranslateError_Nil(t *testing.T) {
	if out := translateError(nil); out != nil {
		t.Errorf("want nil, got %v", out)
	}
}
