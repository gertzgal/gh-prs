package render

import (
	"strings"
	"testing"

	"github.com/gertzgal/gh-prs/internal/model"
)

func TestHeader_CommandLine(t *testing.T) {
	repo := &model.Repo{Owner: "acme", Name: "widget", DefaultBranch: "main"}
	ctx := Context{Command: "gh prs --team fullstack"}
	got := renderHeader(repo, ctx, newStyles(false))
	if !strings.Contains(got, "› gh prs --team fullstack") {
		t.Errorf("missing prompt prefix and command in:\n%s", got)
	}
}

func TestHeader_MetadataStrip(t *testing.T) {
	repo := &model.Repo{Owner: "acme", Name: "widget", DefaultBranch: "main"}
	ctx := Context{
		Command:     "gh prs --team fullstack",
		FilterLabel: "team fullstack",
	}
	got := renderHeader(repo, ctx, newStyles(false))
	if !strings.Contains(got, "repo acme/widget") {
		t.Errorf("missing repo metadata in:\n%s", got)
	}
	if !strings.Contains(got, "base main") {
		t.Errorf("missing base branch in:\n%s", got)
	}
	if !strings.Contains(got, "team fullstack") {
		t.Errorf("missing filter label in:\n%s", got)
	}
}

func TestHeader_CommandEmpty_SuppressesPromptLine(t *testing.T) {
	repo := &model.Repo{Owner: "acme", Name: "widget", DefaultBranch: "main"}
	ctx := Context{Command: ""}
	got := renderHeader(repo, ctx, newStyles(false))
	if strings.Contains(got, "›") {
		t.Errorf("command-echo line should be suppressed when Command is empty; got:\n%s", got)
	}
}
