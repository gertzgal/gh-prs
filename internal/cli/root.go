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
	"github.com/gertzgal/gh-prs/internal/github"
	"github.com/gertzgal/gh-prs/internal/render"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// USAGE reproduces the TS src/cli-args.ts USAGE constant byte-for-byte.
const USAGE = `Usage: gh prs [--json] [--debug] [--help]

  --json    Emit the fetched repo + PRs as JSON to stdout. No colors, no spinner.
  --debug   Print the equivalent gh api REST calls to stderr before querying.
            Also honored via DEBUG=1 env var.
  --help    Show this help.

Exit codes: 0 success · 1 gh/network failure · 2 not in a GitHub repo · 3 no authored open PRs.
`

// Execute parses flags, runs the app, returns the exit code.
// Never panics; maps all errors to exit codes via MapError.
// Call site: cmd/gh-prs/main.go -> os.Exit(cli.Execute(os.Args[1:], os.Environ()))
func Execute(argv []string, env []string) int {
	envMap := envSliceToMap(env)
	var cobraJSON, cobraDebug bool
	runExit := ExitSuccess

	cmd := &cobra.Command{
		Use:           "gh-prs",
		Short:         "Compact overview of the current user's open PRs in the current repo",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			flags := composeFlags(cobraJSON, cobraDebug, envMap)
			runExit = runOnce(flags, envMap, os.Stdout, os.Stderr)
			return nil
		},
	}
	cmd.SetArgs(argv)
	cmd.Flags().BoolVar(&cobraJSON, "json", false, "Emit JSON to stdout (no colors, no spinner)")
	cmd.Flags().BoolVar(&cobraDebug, "debug", false, "Print REST equivalents to stderr (also via DEBUG=1 env)")
	cmd.SetOut(os.Stdout)
	cmd.SetErr(os.Stderr)
	cmd.SetHelpFunc(func(_ *cobra.Command, _ []string) {
		fmt.Fprint(os.Stdout, USAGE)
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

	client, err := github.New()
	if err != nil {
		fmt.Fprintf(stderr, "gh prs: %s\n", err)
		return MapError(err, false)
	}

	spinner := NewSpinner(!flags.JSON, stderrIsTTY, stderr)
	spinner.Start()
	defer spinner.Stop()

	formatter := render.Formatters()[render.NameText]
	if flags.JSON {
		formatter = render.Formatters()[render.NameJSON]
	}

	return app.Run(context.Background(), app.Deps{
		Flags:     app.Flags{JSON: flags.JSON},
		Client:    client,
		Formatter: formatter,
		FormatCtx: render.Context{
			Color:     ShouldColor(env, stdoutIsTTY),
			OSC8:      ShouldOSC8(stdoutIsTTY),
			LatencyMs: 0,
		},
		Stdout: stdout,
		Stderr: stderr,
		Now:    time.Now,
	})
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
