package render

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/gertzgal/gh-prs/internal/github"
	"github.com/gertzgal/gh-prs/internal/model"
)

func loadRepo(t *testing.T, name string) *model.Repo {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "fixtures", name+".json"))
	if err != nil {
		t.Fatal(err)
	}
	repo, err := github.DecodeForTests(raw)
	if err != nil {
		t.Fatal(err)
	}
	return repo
}

func TestJSONFormatter_Valid(t *testing.T) {
	repo := loadRepo(t, "graphql-widget-4-stack")
	out := JSON{}.Format(repo, Context{Color: false, OSC8: false, LatencyMs: 123})
	var parsed any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
}

func TestJSONFormatter_LatencyMerged(t *testing.T) {
	repo := loadRepo(t, "graphql-widget-4-stack")
	out := JSON{}.Format(repo, Context{Color: false, OSC8: false, LatencyMs: 500})
	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatal(err)
	}
	got, ok := parsed["latencyMs"]
	if !ok {
		t.Fatalf("latencyMs key missing, got keys: %v", parsed)
	}
	if got.(float64) != 500 {
		t.Errorf("latencyMs = %v, want 500", got)
	}
}

func TestJSONFormatter_KeyShape(t *testing.T) {
	repo := loadRepo(t, "graphql-widget-4-stack")
	out := JSON{}.Format(repo, Context{Color: false, OSC8: false, LatencyMs: 0})
	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatal(err)
	}
	keys := make([]string, 0, len(parsed))
	for k := range parsed {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	want := []string{"defaultBranch", "latencyMs", "name", "owner", "prs", "rateLimit", "viewerLogin"}
	if len(keys) != len(want) {
		t.Fatalf("keys = %v, want %v", keys, want)
	}
	for i, k := range want {
		if keys[i] != k {
			t.Errorf("keys[%d] = %q, want %q (full got %v)", i, keys[i], k, keys)
		}
	}
}

func TestJSONFormatter_ColorDoesNotAffect(t *testing.T) {
	repo := loadRepo(t, "graphql-widget-4-stack")
	withColor := JSON{}.Format(repo, Context{Color: true, OSC8: true, LatencyMs: 42})
	withoutColor := JSON{}.Format(repo, Context{Color: false, OSC8: false, LatencyMs: 42})
	if withColor != withoutColor {
		t.Errorf("color flag affected output:\nwith=%q\nwithout=%q", withColor, withoutColor)
	}
}
