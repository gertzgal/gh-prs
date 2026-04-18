package cli

import "testing"

func TestComposeFlags_DebugFromFlag(t *testing.T) {
	got := composeFlags(false, true, false, "", false, map[string]string{})
	if !got.Debug {
		t.Errorf("Debug: want true (flag set), got false")
	}
}

func TestComposeFlags_DebugFromEnv(t *testing.T) {
	got := composeFlags(false, false, false, "", false, map[string]string{"DEBUG": "1"})
	if !got.Debug {
		t.Errorf("Debug: want true (DEBUG=1 env), got false")
	}
}

func TestComposeFlags_DebugEnvEmpty(t *testing.T) {
	got := composeFlags(false, false, false, "", false, map[string]string{"DEBUG": ""})
	if got.Debug {
		t.Errorf("Debug: want false (DEBUG=''), got true")
	}
}

func TestComposeFlags_NoCacheFromFlag(t *testing.T) {
	got := composeFlags(false, false, true, "", false, map[string]string{})
	if !got.NoCache {
		t.Errorf("NoCache: want true (flag set), got false")
	}
}

func TestComposeFlags_NoCacheFromEnv(t *testing.T) {
	got := composeFlags(false, false, false, "", false, map[string]string{"GH_PRS_NO_CACHE": "1"})
	if !got.NoCache {
		t.Errorf("NoCache: want true (env truthy), got false")
	}
}

func TestComposeFlags_NoCacheEnvFalsy(t *testing.T) {
	for _, v := range []string{"", "0", "false", "False"} {
		got := composeFlags(false, false, false, "", false, map[string]string{"GH_PRS_NO_CACHE": v})
		if got.NoCache {
			t.Errorf("NoCache: want false for %q, got true", v)
		}
	}
}

func TestComposeFlags_CacheTTLFlagWins(t *testing.T) {
	got := composeFlags(false, false, false, "2m", false, map[string]string{"GH_PRS_CACHE_TTL": "5m"})
	if got.CacheTTL != "2m" {
		t.Errorf("CacheTTL: flag should override env, want 2m, got %q", got.CacheTTL)
	}
}

func TestComposeFlags_CacheTTLFromEnv(t *testing.T) {
	got := composeFlags(false, false, false, "", false, map[string]string{"GH_PRS_CACHE_TTL": "5m"})
	if got.CacheTTL != "5m" {
		t.Errorf("CacheTTL: want 5m from env, got %q", got.CacheTTL)
	}
}

func TestComposeFlags_JSONPassthrough(t *testing.T) {
	got := composeFlags(true, false, false, "", false, map[string]string{})
	if !got.JSON {
		t.Errorf("JSON: want true, got false")
	}
}

func TestComposeFlags_StatsFromFlag(t *testing.T) {
	got := composeFlags(false, false, false, "", true, map[string]string{})
	if !got.Stats {
		t.Errorf("Stats: want true (flag set), got false")
	}
}

func TestComposeFlags_StatsFromEnv(t *testing.T) {
	got := composeFlags(false, false, false, "", false, map[string]string{"GH_PRS_STATS": "1"})
	if !got.Stats {
		t.Errorf("Stats: want true (env truthy), got false")
	}
}

func TestComposeFlags_StatsEnvFalsy(t *testing.T) {
	for _, v := range []string{"", "0", "false", "False"} {
		got := composeFlags(false, false, false, "", false, map[string]string{"GH_PRS_STATS": v})
		if got.Stats {
			t.Errorf("Stats: want false for %q, got true", v)
		}
	}
}

func TestComposeFlags_StatsDefaultOff(t *testing.T) {
	got := composeFlags(false, false, false, "", false, map[string]string{})
	if got.Stats {
		t.Errorf("Stats: want false by default, got true")
	}
}
