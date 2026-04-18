package render

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"
)

var updateGolden = flag.Bool("update", false, "overwrite golden files with current output")

var goldenCases = []struct {
	fixture string
	color   bool
	osc8    bool
	suffix  string
}{
	{"graphql-empty", false, false, ""},
	{"graphql-widget-4-stack", false, false, ""},
	{"graphql-widget-4-stack", true, false, ".color"},
	{"graphql-widget-4-stack", true, true, ".osc8"},
	{"graphql-gadget-standalone", false, false, ""},
	{"graphql-gadget-standalone", true, false, ".color"},
}

func TestGoldenText(t *testing.T) {
	for _, c := range goldenCases {
		t.Run(c.fixture+c.suffix, func(t *testing.T) {
			repo := loadRepo(t, c.fixture)
			got := Text{}.Format(repo, Context{Color: c.color, OSC8: c.osc8, LatencyMs: 0, ShowStats: true})
			path := filepath.Join("..", "..", "testdata", "golden", "text", c.fixture+c.suffix+".txt")
			checkGolden(t, path, []byte(got))
		})
	}
}

func TestGoldenJSON(t *testing.T) {
	for _, c := range goldenCases {
		if c.suffix != "" {
			continue
		}
		t.Run(c.fixture, func(t *testing.T) {
			repo := loadRepo(t, c.fixture)
			got := JSON{}.Format(repo, Context{Color: false, OSC8: false, LatencyMs: 0})
			path := filepath.Join("..", "..", "testdata", "golden", "json", c.fixture+".json")
			checkGolden(t, path, []byte(got))
		})
	}
}

func TestGoldenTOON(t *testing.T) {
	for _, c := range goldenCases {
		if c.suffix != "" {
			continue
		}
		t.Run(c.fixture, func(t *testing.T) {
			repo := loadRepo(t, c.fixture)
			got := TOON{}.Format(repo, Context{Color: false, OSC8: false, LatencyMs: 0})
			path := filepath.Join("..", "..", "testdata", "golden", "toon", c.fixture+".toon")
			checkGolden(t, path, []byte(got))
		})
	}
}

func checkGolden(t *testing.T, path string, got []byte) {
	t.Helper()
	if *updateGolden {
		if err := os.WriteFile(path, got, 0644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("golden mismatch at %s\n--- got ---\n%s\n--- want ---\n%s", path, got, want)
	}
}
