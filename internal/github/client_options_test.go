package github

import (
	"bytes"
	"testing"
	"time"
)

func TestBuildClientOptions_ZeroValue(t *testing.T) {
	co := buildClientOptions(Options{})

	if !co.LogIgnoreEnv {
		t.Errorf("LogIgnoreEnv: want true (we control logging, not GH_DEBUG), got false")
	}
	if co.Log != nil {
		t.Errorf("Log: want nil (no debug), got %v", co.Log)
	}
	if co.LogVerboseHTTP {
		t.Errorf("LogVerboseHTTP: want false, got true")
	}
	if co.EnableCache {
		t.Errorf("EnableCache: want false, got true")
	}
	if co.CacheTTL != 0 {
		t.Errorf("CacheTTL: want 0, got %v", co.CacheTTL)
	}
}

func TestBuildClientOptions_Debug(t *testing.T) {
	var buf bytes.Buffer
	co := buildClientOptions(Options{
		Debug:      true,
		DebugOut:   &buf,
		DebugColor: true,
	})

	if co.Log != &buf {
		t.Errorf("Log: want &buf, got %v", co.Log)
	}
	if !co.LogVerboseHTTP {
		t.Errorf("LogVerboseHTTP: want true when Debug is set, got false")
	}
	if !co.LogColorize {
		t.Errorf("LogColorize: want true (DebugColor=true), got false")
	}
}

func TestBuildClientOptions_DebugButNilWriter(t *testing.T) {
	co := buildClientOptions(Options{Debug: true, DebugOut: nil})

	if co.Log != nil {
		t.Errorf("Log: want nil when DebugOut is nil (no sink), got %v", co.Log)
	}
	if co.LogVerboseHTTP {
		t.Errorf("LogVerboseHTTP: want false when no sink, got true")
	}
}

func TestBuildClientOptions_Cache(t *testing.T) {
	co := buildClientOptions(Options{
		EnableCache: true,
		CacheTTL:    90 * time.Second,
		CacheDir:    "/tmp/gh-prs-test",
	})

	if !co.EnableCache {
		t.Errorf("EnableCache: want true, got false")
	}
	if co.CacheTTL != 90*time.Second {
		t.Errorf("CacheTTL: want 90s, got %v", co.CacheTTL)
	}
	if co.CacheDir != "/tmp/gh-prs-test" {
		t.Errorf("CacheDir: want /tmp/gh-prs-test, got %q", co.CacheDir)
	}
}

func TestBuildClientOptions_CacheDefaultsPassthrough(t *testing.T) {
	// EnableCache without TTL or Dir leaves them zero so go-gh applies its
	// own defaults. We only care that EnableCache flowed through.
	co := buildClientOptions(Options{EnableCache: true})

	if !co.EnableCache {
		t.Errorf("EnableCache: want true, got false")
	}
	if co.CacheTTL != 0 {
		t.Errorf("CacheTTL: want 0 (go-gh default), got %v", co.CacheTTL)
	}
	if co.CacheDir != "" {
		t.Errorf("CacheDir: want empty (go-gh default), got %q", co.CacheDir)
	}
}
