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

const USAGE = `Usage: gh prs [--author <login>] [--format <text|json|toon>] [--debug] [--no-cache] [--cache-ttl <dur>] [--stats] [--help]

  --author <login> Filter by PR author login. Repeatable: --author alice --author bob
                   shows PRs by alice OR bob. Defaults to @me (the authenticated user).
                   Also honored via GH_PRS_AUTHOR (comma-separated: "alice,bob").
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
  --cache-ttl <d>  Cache TTL (Go duration: "60s", "2m", "10m"). Default 60s.
                   Also honored via GH_PRS_CACHE_TTL.
  --stats          Show the footer with request latency, GraphQL query cost,
                   and rate-limit remaining. Hidden by default.
                   Also honored via GH_PRS_STATS=1.
  --help           Show this help.

Cache lives in $XDG_CACHE_HOME/gh-prs/ (or platform equivalent) and is keyed by
the full GraphQL request body, so a different repo or viewer will never collide.

Exit codes: 0 success · 1 gh/network failure · 2 not in a GitHub repo · 3 no authored open PRs.
`

// Execute parses flags, runs the app, returns the exit code.
// Never panics; maps all errors to exit codes via MapError.
// Call site: main.go -> os.Exit(cli.Execute(os.Args[1:], os.Environ()))
func Execute(argv []string, env []string) int {
	envMap := envSliceToMap(env)
	var cobraDebug, cobraNoCache, cobraStats bool
	var cobraFormat, cobraCacheTTL string
	var cobraAuthors []string
	runExit := ExitSuccess

	cmd := &cobra.Command{
		Use:           "gh-prs",
		Short:         "Compact overview of the current user's open PRs in the current repo",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			flags := composeFlags(cobraFormat, cobraDebug, cobraNoCache, cobraCacheTTL, cobraStats, cobraAuthors, envMap)
			if _, ok := render.Lookup(flags.Format); !ok {
				return fmt.Errorf("unknown --format %q (want %s)", flags.Format, strings.Join(render.Names(), "|"))
			}
			runExit = runOnce(flags, envMap, os.Stdout, os.Stderr)
			return nil
		},
	}
	cmd.SetArgs(argv)
	cmd.Flags().StringArrayVar(&cobraAuthors, "author", nil, "Filter by author login (repeatable; default: @me). Also via GH_PRS_AUTHOR.")
	cmd.Flags().StringVarP(&cobraFormat, "format", "f", "", "Output format: text|json|toon (default text; also via GH_PRS_FORMAT)")
	cmd.Flags().BoolVar(&cobraDebug, "debug", false, "Log actual GraphQL request/response to stderr (also via DEBUG=1)")
	cmd.Flags().BoolVar(&cobraNoCache, "no-cache", false, "Skip the disk cache (also via GH_PRS_NO_CACHE=1)")
	cmd.Flags().StringVar(&cobraCacheTTL, "cache-ttl", "", "Cache TTL (e.g. 60s, 2m). Default 60s.")
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

	machine := flags.Machine()
	spinner := NewSpinner(!machine, stderrIsTTY, stderr)
	spinner.Start()
	defer spinner.Stop()

	// Execute already validated via render.Lookup; safe to ignore ok.
	formatter, _ := render.Lookup(flags.Format)

	// Build the filter set. Flags.Authors is empty when --author was not
	// passed (and GH_PRS_AUTHOR is unset); we default to @me so the
	// behaviour matches the original "show my PRs" default.
	authors := flags.Authors
	if len(authors) == 0 {
		authors = []string{"@me"}
	}
	filters := filter.NewSet(
		[]filter.QueryFilter{filter.NewAuthorFilter(authors)},
		nil,
	)

	return app.Run(context.Background(), app.Deps{
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
			AuthorOrder: authors,
		},
		Stdout: stdout,
		Stderr: stderr,
		Now:    time.Now,
	})
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
	if !flags.NoCache {
		ttl, _ := github.ParseCacheTTL(flags.CacheTTL)
		opts.EnableCache = true
		opts.CacheTTL = ttl
		opts.CacheDir = github.DefaultCacheDir()
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

func envSliceToMap(env []string) map[string]string {
	out := make(map[string]string, len(env))
	for _, kv := range env {
		if i := strings.IndexByte(kv, '='); i >= 0 {
			out[kv[:i]] = kv[i+1:]
		}
	}
	return out
}
