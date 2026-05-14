package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/gertzgal/gh-prs/internal/app"
	"github.com/gertzgal/gh-prs/internal/filter"
	"github.com/gertzgal/gh-prs/internal/github"
	"github.com/gertzgal/gh-prs/internal/render"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const USAGE = `Usage: gh prs [--author <login>] [--team <slug>] [--exclude-author <login>] [--format <text|json|toon>] [--debug] [--no-cache] [--cache-ttl <dur>] [--stats] [--help]

  --author <login> Filter by PR author login. Repeatable: --author alice --author bob
                   shows PRs by alice OR bob. Defaults to @me (the authenticated user)
                   unless --team is set. Also honored via GH_PRS_AUTHOR
                   (comma-separated: "alice,bob").
  --team <slug>    Filter by GitHub team slug in the current repo's org. Repeatable:
                   --team backend --team fullstack shows PRs by members of either team.
                   Slugs are case-insensitive. Combines with --author as a union.
                   Memberships are cached for 24h. Also honored via GH_PRS_TEAM.
  --exclude-author <login>
                   Drop PRs by the given author. Repeatable: --exclude-author alice
                   --exclude-author bob hides PRs by either. Supports @me (the
                   authenticated user). Useful with --team to hide one teammate's
                   PRs from the team feed. Also honored via GH_PRS_EXCLUDE_AUTHOR
                   (comma-separated).
  --format <name>  Output format. One of:
                     text  (default) human-readable terminal output with color.
                     json  structured JSON to stdout. No colors, no spinner.
                     toon  Token-Oriented Object Notation (compact, agent-friendly).
                           ~40% fewer tokens than JSON with an explicit tabular schema.
                   Also honored via GH_PRS_FORMAT.
  --debug          Log the actual GraphQL request + response (URL, headers, body, timing)
                   to stderr. Still prints the "REST equivalent" block for orientation.
                   Also honored via DEBUG=1 env var.
  --no-cache       Skip the disk cache for this invocation.
                   Also honored via GH_PRS_NO_CACHE=1.
  --cache-ttl <d>  Cache TTL (Go duration: "60s", "2m", "10m"). Default 5m.
                   Also honored via GH_PRS_CACHE_TTL.
  --stats          Show the footer with request latency, GraphQL query cost,
                   and rate-limit remaining. Hidden by default.
                   Also honored via GH_PRS_STATS=1.
  --help           Show this help.

Cache lives in $XDG_CACHE_HOME/gh-prs/ (or platform equivalent). Data is served
stale for up to 2x the TTL while a background refresh runs.

Exit codes: 0 success · 1 gh/network failure · 2 not in a GitHub repo · 3 no authored open PRs.
`

// Execute parses flags, runs the app, returns the exit code.
// Never panics; maps all errors to exit codes via MapError.
// Call site: main.go -> os.Exit(cli.Execute(os.Args[1:], os.Environ()))
func Execute(argv []string, env []string) int {
	envMap := envSliceToMap(env)
	var cobraDebug, cobraNoCache, cobraStats bool
	var cobraFormat, cobraCacheTTL string
	var cobraAuthors, cobraTeams, cobraExcludeAuthors []string
	runExit := ExitSuccess

	cmd := &cobra.Command{
		Use:           "gh-prs",
		Short:         "Compact overview of the current user's open PRs in the current repo",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			flags := composeFlags(cobraFormat, cobraDebug, cobraNoCache, cobraCacheTTL, cobraStats, cobraAuthors, cobraTeams, cobraExcludeAuthors, envMap)
			if _, ok := render.Lookup(flags.Format); !ok {
				return fmt.Errorf("unknown --format %q (want %s)", flags.Format, strings.Join(render.Names(), "|"))
			}
			runExit = runOnce(flags, envMap, os.Stdout, os.Stderr)
			return nil
		},
	}
	cmd.SetArgs(argv)
	cmd.Flags().StringArrayVar(&cobraAuthors, "author", nil, "Filter by author login (repeatable; default: @me). Also via GH_PRS_AUTHOR.")
	cmd.Flags().StringArrayVar(&cobraTeams, "team", nil, "Filter by team slug in current repo's org (repeatable, case-insensitive). Also via GH_PRS_TEAM.")
	cmd.Flags().StringArrayVar(&cobraExcludeAuthors, "exclude-author", nil, "Exclude PRs by author login (repeatable; supports @me). Also via GH_PRS_EXCLUDE_AUTHOR.")
	cmd.Flags().StringVarP(&cobraFormat, "format", "f", "", "Output format: text|json|toon (default text; also via GH_PRS_FORMAT)")
	cmd.Flags().BoolVar(&cobraDebug, "debug", false, "Log actual GraphQL request/response to stderr (also via DEBUG=1)")
	cmd.Flags().BoolVar(&cobraNoCache, "no-cache", false, "Skip the disk cache (also via GH_PRS_NO_CACHE=1)")
	cmd.Flags().StringVar(&cobraCacheTTL, "cache-ttl", "", "Cache TTL (e.g. 60s, 2m). Default 5m.")
	cmd.Flags().BoolVar(&cobraStats, "stats", false, "Show latency + GraphQL cost + rate-limit footer (also via GH_PRS_STATS=1)")
	cmd.SetOut(os.Stdout)
	cmd.SetErr(os.Stderr)
	cmd.SetHelpFunc(func(_ *cobra.Command, _ []string) {
		_, _ = fmt.Fprint(os.Stdout, USAGE)
	})

	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "gh prs: %s\n%s", err, USAGE)
		return ExitUsage
	}
	return runExit
}

func runOnce(flags Flags, env map[string]string, stdout, stderr io.Writer) int {
	if flags.Debug {
		if owner, name, ok := tryCurrentRepo(); ok {
			PrintRESTEquivalent(owner, name, stderr)
		}
	}
	stdoutIsTTY := term.IsTerminal(int(os.Stdout.Fd()))
	stderrIsTTY := term.IsTerminal(int(os.Stderr.Fd()))

	clientOpts := buildClientOptions(flags, env, stderr, stderrIsTTY)
	client, err := github.New(clientOpts)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "gh prs: %s\n", err)
		return MapError(err, false)
	}

	var swr *github.SWRClient
	if !flags.NoCache {
		ttl, _ := github.ParseCacheTTL(flags.CacheTTL)
		swr = github.NewSWRClient(client, repository.Current, github.DefaultCacheDir(), ttl)
		client = swr
	}

	machine := flags.Machine()
	spinner := NewSpinner(!machine, stderrIsTTY, stderr)
	spinner.Start()
	defer spinner.Stop()

	// Execute already validated via render.Lookup; safe to ignore ok.
	formatter, _ := render.Lookup(flags.Format)

	// Build the filter set.
	//
	// Author/team logic:
	//   1. Resolve --team slugs to logins (network; cached on disk for 24h).
	//   2. Union team logins with --author logins.
	//   3. If the user passed neither --author nor --team, default to @me
	//      so the bare "gh prs" still means "show my PRs".
	//   4. If --team was passed but --author was not, DO NOT inject @me:
	//      the user's intent is "PRs by this team", not "PRs by me or this
	//      team".
	teamLogins, teamErr := resolveTeamsForCurrentRepo(flags, clientOpts, stderr)
	if teamErr != nil {
		spinner.Stop()
		_, _ = fmt.Fprintf(stderr, "gh prs: %s\n", teamErr)
		return MapError(teamErr, false)
	}

	authors := flags.Authors
	if len(authors) == 0 && len(teamLogins) == 0 {
		authors = []string{"@me"}
	}
	combined := unionLogins(authors, teamLogins)
	// Strip excluded logins from the author-order list so the renderer
	// does not reserve an empty group for an author whose PRs we just
	// removed. Comparison is case-insensitive to match filter semantics.
	authorOrder := pruneExcluded(combined, flags.ExcludeAuthors)
	af := filter.NewAuthorFilter(combined)
	queries := []filter.QueryFilter{af}
	lists := []filter.ListFilter{af}
	if len(flags.ExcludeAuthors) > 0 {
		ex := filter.NewExcludeAuthorFilter(flags.ExcludeAuthors)
		queries = append(queries, ex)
		lists = append(lists, ex)
	}
	filters := filter.NewSet(queries, lists)

	exitCode := app.Run(context.Background(), app.Deps{
		Flags:     app.Flags{Machine: machine},
		Filters:   filters,
		Client:    client,
		Formatter: formatter,
		FormatCtx: render.Context{
			Color:       ShouldColor(env, stdoutIsTTY),
			OSC8:        ShouldOSC8(stdoutIsTTY),
			LatencyMs:   0,
			ShowStats:   flags.Stats,
			FilterLabel: filters.Label(),
			AuthorOrder: authorOrder,
			Command:     renderCommand(flags),
			Width:       TerminalWidth(int(os.Stdout.Fd())),
		},
		Stdout: stdout,
		Stderr: stderr,
		Now:    time.Now,
	})

	if swr != nil {
		swr.LingerWait()
	}
	return exitCode
}

// buildClientOptions converts CLI flags + env into github.Options. Debug logs
// are colorized only when stderr is a TTY and color is not suppressed.
func buildClientOptions(flags Flags, env map[string]string, stderr io.Writer, stderrIsTTY bool) github.Options {
	opts := github.Options{}
	if flags.Debug {
		opts.Debug = true
		opts.DebugOut = stderr
		opts.DebugColor = ShouldColor(env, stderrIsTTY)
	}
	return opts
}

func tryCurrentRepo() (owner, name string, ok bool) {
	r, err := repository.Current()
	if err != nil {
		return "", "", false
	}
	return r.Owner, r.Name, true
}

// renderCommand reconstructs the canonical invocation from parsed flags so
// the text formatter can echo it back. We deliberately do NOT use os.Args:
// callers may invoke us through aliases, scripts, or different argv-shapes,
// and the canonical form is more useful in shared output.
//
// Order: filters (author/team/exclude-author) first, then orthogonal flags.
// Keeps the line stable across invocations with the same effective query.
// Flags left at their defaults are omitted so the bare "gh prs" stays bare.
func renderCommand(flags Flags) string {
	parts := []string{"gh prs"}
	for _, a := range flags.Authors {
		parts = append(parts, "--author", a)
	}
	for _, t := range flags.Teams {
		parts = append(parts, "--team", t)
	}
	for _, a := range flags.ExcludeAuthors {
		parts = append(parts, "--exclude-author", a)
	}
	if flags.Format != "" && flags.Format != DefaultFormat {
		parts = append(parts, "--format", flags.Format)
	}
	if flags.Stats {
		parts = append(parts, "--stats")
	}
	if flags.NoCache {
		parts = append(parts, "--no-cache")
	}
	if flags.CacheTTL != "" {
		parts = append(parts, "--cache-ttl", flags.CacheTTL)
	}
	if flags.Debug {
		parts = append(parts, "--debug")
	}
	return strings.Join(parts, " ")
}

func envSliceToMap(env []string) map[string]string {
	out := make(map[string]string, len(env))
	for _, kv := range env {
		if i := strings.IndexByte(kv, '='); i >= 0 {
			out[kv[:i]] = kv[i+1:]
		}
	}
	return out
}
