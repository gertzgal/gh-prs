package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/gertzgal/gh-prs/internal/filter"
	"github.com/gertzgal/gh-prs/internal/github"
	"github.com/gertzgal/gh-prs/internal/model"
	"github.com/gertzgal/gh-prs/internal/render"
	"github.com/gertzgal/gh-prs/internal/stacks"
)

const (
	exitSuccess = 0
	exitGhError = 1
	exitNoRepo  = 2
	exitNoPRs   = 3
)

// Flags captures the subset of CLI intent that affects Run's behaviour.
// Machine is true when the output format is for machine consumption
// (json/toon): Run skips the human-oriented "no open PRs" message and always
// delegates to the formatter. Kept minimal to avoid coupling app to cli.
type Flags struct {
	Machine bool
}

// Deps is the injectable set of collaborators Run needs.
type Deps struct {
	Flags     Flags
	Filters   filter.Set
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
	repo, err := d.Client.FetchRepo(ctx, d.Filters)
	if err != nil {
		return reportFetchError(err, d.Stderr)
	}

	// Apply list filters (post-fetch). ResolveAndApply substitutes "@me" with
	// the viewer login from the GraphQL response, then runs all ListFilters in
	// order.
	repo.PRs = d.Filters.ResolveAndApply(repo.PRs, repo.ViewerLogin)

	// Derive stack topology (stackId + stackPos) once, here, so every
	// formatter receives consistently annotated input. Keeps presentation
	// code free of the stacks package.
	repo.PRs = stacks.Annotate(repo.PRs, repo.DefaultBranch)

	// Resolve "@me" in AuthorOrder for render grouping.
	authorOrder := make([]string, len(d.FormatCtx.AuthorOrder))
	for i, login := range d.FormatCtx.AuthorOrder {
		if login == "@me" {
			authorOrder[i] = repo.ViewerLogin
		} else {
			authorOrder[i] = login
		}
	}

	latencyMs := int(d.Now().Sub(start).Round(time.Millisecond) / time.Millisecond)
	ctx2 := d.FormatCtx
	ctx2.LatencyMs = latencyMs
	ctx2.AuthorOrder = authorOrder

	if !d.Flags.Machine && len(repo.PRs) == 0 {
		_, _ = fmt.Fprintf(d.Stdout, "\nNo open PRs matching the applied filters in %s/%s.\n\n", repo.Owner, repo.Name)
		return exitNoPRs
	}
	out, err := d.Formatter.Format(repo, ctx2)
	if err != nil {
		_, _ = fmt.Fprintf(d.Stderr, "gh prs: format: %v\n", err)
		return exitGhError
	}
	_, _ = io.WriteString(d.Stdout, out)
	if len(repo.PRs) > 0 {
		return exitSuccess
	}
	return exitNoPRs
}

func reportFetchError(err error, stderr io.Writer) int {
	if errors.Is(err, model.ErrRepoNotFound) {
		_, _ = fmt.Fprintln(stderr, "gh prs: not inside a GitHub repo")
		return exitNoRepo
	}
	var ghErr *model.GhError
	if errors.As(err, &ghErr) {
		_, _ = fmt.Fprintf(stderr, "gh prs: %s\n", ghErr.Msg)
		if ghErr.Stderr != "" {
			_, _ = fmt.Fprintln(stderr, ghErr.Stderr)
		}
		return exitGhError
	}
	_, _ = fmt.Fprintf(stderr, "gh prs: unexpected error: %v\n", err)
	return exitGhError
}
