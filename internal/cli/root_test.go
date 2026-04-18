package cli

import (
	"bytes"
	"testing"

	"github.com/gertzgal/gh-prs/internal/github"
)

func TestBuildClientOptions_DefaultsToCachingOn(t *testing.T) {
	flags := Flags{}
	env := map[string]string{}
	var stderr bytes.Buffer

	got := buildClientOptions(flags, env, &stderr, false)

	if !got.EnableCache {
		t.Errorf("EnableCache: want true by default, got false")
	}
	if got.CacheTTL != github.DefaultCacheTTL {
		t.Errorf("CacheTTL: want default %v, got %v", github.DefaultCacheTTL, got.CacheTTL)
	}
	if got.CacheDir == "" {
		t.Errorf("CacheDir: want non-empty default path, got empty")
	}
	if got.Debug {
		t.Errorf("Debug: want false by default, got true")
	}
}

func TestBuildClientOptions_NoCache(t *testing.T) {
	flags := Flags{NoCache: true}
	var stderr bytes.Buffer

	got := buildClientOptions(flags, map[string]string{}, &stderr, false)

	if got.EnableCache {
		t.Errorf("EnableCache: want false when NoCache=true, got true")
	}
	if got.CacheDir != "" {
		t.Errorf("CacheDir: want empty when NoCache=true, got %q", got.CacheDir)
	}
}

func TestBuildClientOptions_DebugWiresStderr(t *testing.T) {
	flags := Flags{Debug: true}
	var stderr bytes.Buffer

	got := buildClientOptions(flags, map[string]string{}, &stderr, true)

	if !got.Debug {
		t.Errorf("Debug: want true, got false")
	}
	if got.DebugOut != &stderr {
		t.Errorf("DebugOut: want &stderr, got %v", got.DebugOut)
	}
	// TTY + no NO_COLOR => color on.
	if !got.DebugColor {
		t.Errorf("DebugColor: want true (TTY + no NO_COLOR), got false")
	}
}

func TestBuildClientOptions_DebugNoColorWhenNonTTY(t *testing.T) {
	flags := Flags{Debug: true}
	var stderr bytes.Buffer

	got := buildClientOptions(flags, map[string]string{}, &stderr, false)

	if got.DebugColor {
		t.Errorf("DebugColor: want false (non-TTY stderr), got true")
	}
}

func TestBuildClientOptions_ExplicitCacheTTL(t *testing.T) {
	flags := Flags{CacheTTL: "90s"}
	var stderr bytes.Buffer

	got := buildClientOptions(flags, map[string]string{}, &stderr, false)

	if got.CacheTTL.Seconds() != 90 {
		t.Errorf("CacheTTL: want 90s, got %v", got.CacheTTL)
	}
}

func TestBuildClientOptions_BadCacheTTLFallsBack(t *testing.T) {
	flags := Flags{CacheTTL: "garbage"}
	var stderr bytes.Buffer

	got := buildClientOptions(flags, map[string]string{}, &stderr, false)

	if got.CacheTTL != github.DefaultCacheTTL {
		t.Errorf("CacheTTL: want default %v for bad input, got %v", github.DefaultCacheTTL, got.CacheTTL)
	}
}
