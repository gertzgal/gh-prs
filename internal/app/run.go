package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/gertzgal/gh-prs/internal/github"
	"github.com/gertzgal/gh-prs/internal/model"
	"github.com/gertzgal/gh-prs/internal/render"
)

// Exit codes duplicated as literals here rather than imported from internal/cli
// to avoid the import cycle cli → app → cli (CONTRACTS.md import graph forbids
// app importing cli; the task file's sample import list was inconsistent with
// the graph). cli.ExitXxx remains the single source of truth for callers.
const (
	exitSuccess = 0
	exitGhError = 1
	exitNoRepo  = 2
	exitNoPRs   = 3
)

// Flags mirrors the caller's JSON intent. We intentionally do not duplicate the
// full cli.Flags surface to keep the app package free of cobra/cli concerns.
type Flags struct {
	JSON bool
}

// Deps is the injectable set of collaborators Run needs.
type Deps struct {
	Flags     Flags
	Client    github.Client
	Formatter render.Formatter
	FormatCtx render.Context
	Stdout    io.Writer
	Stderr    io.Writer
	Now       func() time.Time
}

// Run fetches, formats, writes. Returns the exit code to pass to os.Exit.
// Never panics.
func Run(ctx context.Context, d Deps) int {
	start := d.Now()
	repo, err := d.Client.FetchRepo(ctx)
	if err != nil {
		return reportFetchError(err, d.Stderr)
	}

	latencyMs := int(d.Now().Sub(start).Round(time.Millisecond) / time.Millisecond)
	ctx2 := d.FormatCtx
	ctx2.LatencyMs = latencyMs

	if !d.Flags.JSON && len(repo.PRs) == 0 {
		fmt.Fprintf(d.Stdout, "\nNo open PRs authored by @%s in %s/%s.\n\n", repo.ViewerLogin, repo.Owner, repo.Name)
		return exitNoPRs
	}
	_, _ = io.WriteString(d.Stdout, d.Formatter.Format(repo, ctx2))
	if len(repo.PRs) > 0 {
		return exitSuccess
	}
	return exitNoPRs
}

func reportFetchError(err error, stderr io.Writer) int {
	if errors.Is(err, model.ErrRepoNotFound) {
		fmt.Fprintln(stderr, "gh prs: not inside a GitHub repo")
		return exitNoRepo
	}
	var ghErr *model.GhError
	if errors.As(err, &ghErr) {
		fmt.Fprintf(stderr, "gh prs: %s\n", ghErr.Msg)
		if ghErr.Stderr != "" {
			fmt.Fprintln(stderr, ghErr.Stderr)
		}
		return exitGhError
	}
	fmt.Fprintf(stderr, "gh prs: unexpected error: %v\n", err)
	return exitGhError
}
