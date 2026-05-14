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
	fixture     string
	color       bool
	osc8        bool
	suffix      string
	authorOrder []string
	command     string
	width       int
}{
	{"graphql-empty", false, false, "", nil, "", 0},
	{"graphql-widget-4-stack", false, false, "", nil, "gh prs", 0},
	{"graphql-widget-4-stack", true, false, ".color", nil, "gh prs", 0},
	{"graphql-widget-4-stack", true, true, ".osc8", nil, "gh prs", 0},
	{"graphql-gadget-standalone", false, false, "", nil, "", 0},
	{"graphql-gadget-standalone", true, false, ".color", nil, "", 0},
	// multi-author fixture: text golden with author sections; JSON/TOON use suffix="" so
	// they run normally through the JSON/TOON loop (no AuthorOrder needed for those formats).
	{"graphql-multi-author", false, false, "", nil, "", 0},
	{"graphql-multi-author", false, false, ".multi-author", []string{"alice", "bob"}, "gh prs --author alice --author bob", 0},
	// Width-driven legend variants. Width 70 suppresses the legend entirely;
	// 90 forces a multi-line wrap inside the box; 130 still wraps because the
	// body content is wider than a single line at that width, but the box
	// fills the full terminal width.
	{"graphql-widget-4-stack", false, false, ".w70", nil, "gh prs", 70},
	{"graphql-widget-4-stack", false, false, ".w90", nil, "gh prs", 90},
	{"graphql-widget-4-stack", false, false, ".w130", nil, "gh prs", 130},
}

func TestGoldenText(t *testing.T) {
	for _, c := range goldenCases {
		t.Run(c.fixture+c.suffix, func(t *testing.T) {
			repo := loadRepo(t, c.fixture)
			got := mustFormat(t, Text{}, repo, Context{
				Color:       c.color,
				OSC8:        c.osc8,
				LatencyMs:   0,
				ShowStats:   true,
				AuthorOrder: c.authorOrder,
				Command:     c.command,
				Width:       c.width,
			})
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
			got := mustFormat(t, JSON{}, repo, Context{Color: false, OSC8: false, LatencyMs: 0})
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
			got := mustFormat(t, TOON{}, repo, Context{Color: false, OSC8: false, LatencyMs: 0})
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
