package cli

import (
	"strings"

	"github.com/gertzgal/gh-prs/internal/render"
)

// DefaultFormat is the human-readable terminal output selected when neither
// --format nor GH_PRS_FORMAT is provided.
const DefaultFormat = render.FormatText

type Flags struct {
	// Format is the canonical name of the chosen output format. Never empty:
	// composeFlags resolves missing input to DefaultFormat. Validity is
	// checked separately via render.Lookup in Execute.
	Format   string
	Debug    bool
	Help     bool
	NoCache  bool
	CacheTTL string
	Stats    bool
}

// Machine reports whether the selected format is for machine consumption
// (json/toon). Used to suppress TTY-oriented UX like the spinner and the
// "No open PRs" human message.
func (f Flags) Machine() bool { return f.Format != render.FormatText }

// composeFlags merges cobra-parsed flags with env. Env overrides:
//   - DEBUG=<non-empty>          enables --debug
//   - GH_PRS_NO_CACHE=<truthy>   enables --no-cache
//   - GH_PRS_CACHE_TTL=<dur>     sets cache TTL if --cache-ttl not passed
//   - GH_PRS_STATS=<truthy>      enables --stats
//   - GH_PRS_FORMAT=<name>       sets format if --format not passed
func composeFlags(cobraFormat string, cobraDebug, cobraNoCache bool, cobraCacheTTL string, cobraStats bool, env map[string]string) Flags {
	debug := cobraDebug
	if !debug {
		if v, ok := env["DEBUG"]; ok && v != "" {
			debug = true
		}
	}

	noCache := cobraNoCache
	if !noCache {
		if v, ok := env["GH_PRS_NO_CACHE"]; ok && truthyFlag(v, ok) {
			noCache = true
		}
	}

	ttl := cobraCacheTTL
	if ttl == "" {
		ttl = env["GH_PRS_CACHE_TTL"]
	}

	stats := cobraStats
	if !stats {
		if v, ok := env["GH_PRS_STATS"]; ok && truthyFlag(v, ok) {
			stats = true
		}
	}

	format := strings.ToLower(strings.TrimSpace(cobraFormat))
	if format == "" {
		format = strings.ToLower(strings.TrimSpace(env["GH_PRS_FORMAT"]))
	}
	if format == "" {
		format = DefaultFormat
	}

	return Flags{
		Format:   format,
		Debug:    debug,
		NoCache:  noCache,
		CacheTTL: ttl,
		Stats:    stats,
	}
}
