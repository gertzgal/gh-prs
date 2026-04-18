package cli

// Flags mirrors the TS Args type. Cobra handles --help/-h; Help is kept here
// for symmetry with the TS surface.
type Flags struct {
	JSON    bool
	Debug   bool
	Help    bool
	NoCache bool
	// CacheTTL is the raw string value (e.g. "60s", "2m"). Parsed by github.ParseCacheTTL.
	CacheTTL string
	Stats    bool
}

// composeFlags merges cobra-parsed flags with env. Env overrides:
//   - DEBUG=<non-empty>       enables --debug
//   - GH_PRS_NO_CACHE=<truthy> enables --no-cache
//   - GH_PRS_CACHE_TTL=<dur>  sets cache TTL if --cache-ttl not passed
//   - GH_PRS_STATS=<truthy>    enables --stats
func composeFlags(cobraJSON, cobraDebug, cobraNoCache bool, cobraCacheTTL string, cobraStats bool, env map[string]string) Flags {
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

	return Flags{JSON: cobraJSON, Debug: debug, NoCache: noCache, CacheTTL: ttl, Stats: stats}
}
