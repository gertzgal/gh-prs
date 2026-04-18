package cli

import "strings"

// Output format names recognized by the CLI. Kept in sync with render.Name*.
const (
	FormatText = "text"
	FormatJSON = "json"
	FormatTOON = "toon"
)

// DefaultFormat is the human-readable terminal output.
const DefaultFormat = FormatText

type Flags struct {
	// Format is one of text|json|toon. Empty string resolves to DefaultFormat.
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
func (f Flags) Machine() bool { return f.Format != FormatText }

// validFormat reports whether s is one of the supported format names.
func validFormat(s string) bool {
	return s == FormatText || s == FormatJSON || s == FormatTOON
}

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
