package github

import (
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultCacheDir_EndsWithGhPrs(t *testing.T) {
	got := DefaultCacheDir()
	if got == "" {
		t.Fatal("DefaultCacheDir: want non-empty, got empty")
	}
	if filepath.Base(got) != "gh-prs" {
		t.Errorf("DefaultCacheDir: want basename gh-prs (isolated from gh's own cache), got %q", got)
	}
	if !filepath.IsAbs(got) {
		t.Errorf("DefaultCacheDir: want absolute path, got %q", got)
	}
}

func TestParseCacheTTL(t *testing.T) {
	tests := []struct {
		name         string
		in           string
		wantDur      time.Duration
		wantExplicit bool
	}{
		{"empty → default, not explicit", "", DefaultCacheTTL, false},
		{"valid 60s → 60s, explicit", "60s", 60 * time.Second, true},
		{"valid 2m → 2m, explicit", "2m", 2 * time.Minute, true},
		{"garbage → default, not explicit", "not-a-duration", DefaultCacheTTL, false},
		{"zero → default, not explicit", "0s", DefaultCacheTTL, false},
		{"negative → default, not explicit", "-5s", DefaultCacheTTL, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotDur, gotExplicit := ParseCacheTTL(tc.in)
			if gotDur != tc.wantDur {
				t.Errorf("duration: want %v, got %v", tc.wantDur, gotDur)
			}
			if gotExplicit != tc.wantExplicit {
				t.Errorf("explicit: want %v, got %v", tc.wantExplicit, gotExplicit)
			}
		})
	}
}

func TestDefaultCacheTTL_Short(t *testing.T) {
	// Sanity: 60s keeps the tool responsive to CI/review-state changes. If
	// someone bumps this above a few minutes they should say so explicitly.
	if DefaultCacheTTL > 5*time.Minute {
		t.Errorf("DefaultCacheTTL: %v is too long for dynamic PR state", DefaultCacheTTL)
	}
	if DefaultCacheTTL < time.Second {
		t.Errorf("DefaultCacheTTL: %v is too short to be useful", DefaultCacheTTL)
	}
}
