package github

import (
	"os"
	"path/filepath"
	"time"
)

// DefaultCacheTTL is the TTL applied when caching is enabled and no other
// value is specified. Kept short because PR state (CI, review decision,
// merge status) can change minute-to-minute.
const DefaultCacheTTL = 60 * time.Second

// DefaultCacheDir returns the per-user cache directory for gh-prs entries.
// Falls back to a sibling of the binary's temp dir if UserCacheDir is not
// resolvable (very rare on macOS/Linux; e.g. HOME unset in a sandbox).
func DefaultCacheDir() string {
	base, err := os.UserCacheDir()
	if err != nil || base == "" {
		return filepath.Join(os.TempDir(), "gh-prs-cache")
	}
	return filepath.Join(base, "gh-prs")
}

// ParseCacheTTL parses a human duration string (e.g. "60s", "2m"). Empty or
// invalid input returns (DefaultCacheTTL, false) so callers can distinguish
// "defaulted" from "explicitly set".
func ParseCacheTTL(raw string) (time.Duration, bool) {
	if raw == "" {
		return DefaultCacheTTL, false
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return DefaultCacheTTL, false
	}
	return d, true
}
