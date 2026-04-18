package cli

import (
	"testing"

	"github.com/gertzgal/gh-prs/internal/render"
)

func TestComposeFlags_DebugFromFlag(t *testing.T) {
	got := composeFlags("", true, false, "", false, nil, map[string]string{})
	if !got.Debug {
		t.Errorf("Debug: want true (flag set), got false")
	}
}

func TestComposeFlags_DebugFromEnv(t *testing.T) {
	got := composeFlags("", false, false, "", false, nil, map[string]string{"DEBUG": "1"})
	if !got.Debug {
		t.Errorf("Debug: want true (DEBUG=1 env), got false")
	}
}

func TestComposeFlags_DebugEnvEmpty(t *testing.T) {
	got := composeFlags("", false, false, "", false, nil, map[string]string{"DEBUG": ""})
	if got.Debug {
		t.Errorf("Debug: want false (DEBUG=''), got true")
	}
}

func TestComposeFlags_NoCacheFromFlag(t *testing.T) {
	got := composeFlags("", false, true, "", false, nil, map[string]string{})
	if !got.NoCache {
		t.Errorf("NoCache: want true (flag set), got false")
	}
}

func TestComposeFlags_NoCacheFromEnv(t *testing.T) {
	got := composeFlags("", false, false, "", false, nil, map[string]string{"GH_PRS_NO_CACHE": "1"})
	if !got.NoCache {
		t.Errorf("NoCache: want true (env truthy), got false")
	}
}

func TestComposeFlags_NoCacheEnvFalsy(t *testing.T) {
	for _, v := range []string{"", "0", "false", "False"} {
		got := composeFlags("", false, false, "", false, nil, map[string]string{"GH_PRS_NO_CACHE": v})
		if got.NoCache {
			t.Errorf("NoCache: want false for %q, got true", v)
		}
	}
}

func TestComposeFlags_CacheTTLFlagWins(t *testing.T) {
	got := composeFlags("", false, false, "2m", false, nil, map[string]string{"GH_PRS_CACHE_TTL": "5m"})
	if got.CacheTTL != "2m" {
		t.Errorf("CacheTTL: flag should override env, want 2m, got %q", got.CacheTTL)
	}
}

func TestComposeFlags_CacheTTLFromEnv(t *testing.T) {
	got := composeFlags("", false, false, "", false, nil, map[string]string{"GH_PRS_CACHE_TTL": "5m"})
	if got.CacheTTL != "5m" {
		t.Errorf("CacheTTL: want 5m from env, got %q", got.CacheTTL)
	}
}

func TestComposeFlags_FormatDefaultsToText(t *testing.T) {
	got := composeFlags("", false, false, "", false, nil, map[string]string{})
	if got.Format != render.FormatText {
		t.Errorf("Format: want %q default, got %q", render.FormatText, got.Format)
	}
	if got.Machine() {
		t.Errorf("Machine(): want false for text, got true")
	}
}

func TestComposeFlags_FormatFromFlag(t *testing.T) {
	for _, name := range []string{render.FormatText, render.FormatJSON, render.FormatTOON} {
		got := composeFlags(name, false, false, "", false, nil, map[string]string{})
		if got.Format != name {
			t.Errorf("Format: want %q, got %q", name, got.Format)
		}
	}
}

func TestComposeFlags_FormatFromEnv(t *testing.T) {
	got := composeFlags("", false, false, "", false, nil, map[string]string{"GH_PRS_FORMAT": "toon"})
	if got.Format != render.FormatTOON {
		t.Errorf("Format: want %q from env, got %q", render.FormatTOON, got.Format)
	}
}

func TestComposeFlags_FormatFlagWinsOverEnv(t *testing.T) {
	got := composeFlags("json", false, false, "", false, nil, map[string]string{"GH_PRS_FORMAT": "toon"})
	if got.Format != render.FormatJSON {
		t.Errorf("Format: flag should beat env, want %q, got %q", render.FormatJSON, got.Format)
	}
}

func TestComposeFlags_FormatCaseInsensitive(t *testing.T) {
	got := composeFlags("  JSON  ", false, false, "", false, nil, map[string]string{})
	if got.Format != render.FormatJSON {
		t.Errorf("Format: want normalised %q, got %q", render.FormatJSON, got.Format)
	}
}

func TestComposeFlags_MachineTrueForJSONandTOON(t *testing.T) {
	for _, name := range []string{render.FormatJSON, render.FormatTOON} {
		got := composeFlags(name, false, false, "", false, nil, map[string]string{})
		if !got.Machine() {
			t.Errorf("Machine(): want true for %q, got false", name)
		}
	}
}

func TestComposeFlags_StatsFromFlag(t *testing.T) {
	got := composeFlags("", false, false, "", true, nil, map[string]string{})
	if !got.Stats {
		t.Errorf("Stats: want true (flag set), got false")
	}
}

func TestComposeFlags_StatsFromEnv(t *testing.T) {
	got := composeFlags("", false, false, "", false, nil, map[string]string{"GH_PRS_STATS": "1"})
	if !got.Stats {
		t.Errorf("Stats: want true (env truthy), got false")
	}
}

func TestComposeFlags_StatsEnvFalsy(t *testing.T) {
	for _, v := range []string{"", "0", "false", "False"} {
		got := composeFlags("", false, false, "", false, nil, map[string]string{"GH_PRS_STATS": v})
		if got.Stats {
			t.Errorf("Stats: want false for %q, got true", v)
		}
	}
}

func TestComposeFlags_StatsDefaultOff(t *testing.T) {
	got := composeFlags("", false, false, "", false, nil, map[string]string{})
	if got.Stats {
		t.Errorf("Stats: want false by default, got true")
	}
}

// ---------------------------------------------------------------------------
// --author / GH_PRS_AUTHOR
// ---------------------------------------------------------------------------

func TestComposeFlags_AuthorDefault_NilWhenNotSet(t *testing.T) {
	got := composeFlags("", false, false, "", false, nil, map[string]string{})
	if len(got.Authors) != 0 {
		t.Errorf("Authors: want empty (no flag, no env), got %v", got.Authors)
	}
}

func TestComposeFlags_AuthorFromFlag_Single(t *testing.T) {
	got := composeFlags("", false, false, "", false, []string{"alice"}, map[string]string{})
	if len(got.Authors) != 1 || got.Authors[0] != "alice" {
		t.Errorf("Authors: want [alice], got %v", got.Authors)
	}
}

func TestComposeFlags_AuthorFromFlag_Multiple(t *testing.T) {
	got := composeFlags("", false, false, "", false, []string{"alice", "bob"}, map[string]string{})
	if len(got.Authors) != 2 || got.Authors[0] != "alice" || got.Authors[1] != "bob" {
		t.Errorf("Authors: want [alice bob], got %v", got.Authors)
	}
}

func TestComposeFlags_AuthorFromEnv_Single(t *testing.T) {
	got := composeFlags("", false, false, "", false, nil, map[string]string{"GH_PRS_AUTHOR": "alice"})
	if len(got.Authors) != 1 || got.Authors[0] != "alice" {
		t.Errorf("Authors: want [alice] from env, got %v", got.Authors)
	}
}

func TestComposeFlags_AuthorFromEnv_CommaSeparated(t *testing.T) {
	got := composeFlags("", false, false, "", false, nil, map[string]string{"GH_PRS_AUTHOR": "alice,bob"})
	if len(got.Authors) != 2 || got.Authors[0] != "alice" || got.Authors[1] != "bob" {
		t.Errorf("Authors: want [alice bob] from env, got %v", got.Authors)
	}
}

func TestComposeFlags_AuthorFromEnv_TrimsSpaces(t *testing.T) {
	got := composeFlags("", false, false, "", false, nil, map[string]string{"GH_PRS_AUTHOR": " alice , bob "})
	if len(got.Authors) != 2 || got.Authors[0] != "alice" || got.Authors[1] != "bob" {
		t.Errorf("Authors: want trimmed [alice bob], got %v", got.Authors)
	}
}

func TestComposeFlags_AuthorFlagWinsOverEnv(t *testing.T) {
	// CLI flag should shadow the env var entirely.
	got := composeFlags("", false, false, "", false, []string{"carol"}, map[string]string{"GH_PRS_AUTHOR": "alice,bob"})
	if len(got.Authors) != 1 || got.Authors[0] != "carol" {
		t.Errorf("Authors: flag should win over env, want [carol], got %v", got.Authors)
	}
}

func TestComposeFlags_AuthorEnvEmpty_ProducesNil(t *testing.T) {
	got := composeFlags("", false, false, "", false, nil, map[string]string{"GH_PRS_AUTHOR": ""})
	if len(got.Authors) != 0 {
		t.Errorf("Authors: empty env should produce no authors, got %v", got.Authors)
	}
}
