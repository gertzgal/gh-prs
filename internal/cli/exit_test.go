package cli

import (
	"errors"
	"fmt"
	"testing"

	"github.com/gertzgal/gh-prs/internal/model"
)

func TestMapError(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		hasPRs bool
		want   int
	}{
		{"RepoNotFound => ExitNoRepo", model.ErrRepoNotFound, false, ExitNoRepo},
		{"wrapped RepoNotFound => ExitNoRepo", fmt.Errorf("wrap: %w", model.ErrRepoNotFound), false, ExitNoRepo},
		{"GhError => ExitGhError", &model.GhError{Msg: "foo"}, false, ExitGhError},
		{"generic error => ExitGhError", errors.New("generic"), false, ExitGhError},
		{"UsageError => ExitUsage", &model.UsageError{Msg: "unknown"}, false, ExitUsage},
		{"nil + hasPRs=false => ExitNoPRs", nil, false, ExitNoPRs},
		{"nil + hasPRs=true => ExitSuccess", nil, true, ExitSuccess},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := MapError(tc.err, tc.hasPRs)
			if got != tc.want {
				t.Fatalf("MapError(%v, %v) = %d, want %d", tc.err, tc.hasPRs, got, tc.want)
			}
		})
	}
}

func TestExitCodeConstants(t *testing.T) {
	if ExitSuccess != 0 {
		t.Errorf("ExitSuccess = %d, want 0", ExitSuccess)
	}
	if ExitGhError != 1 {
		t.Errorf("ExitGhError = %d, want 1", ExitGhError)
	}
	if ExitNoRepo != 2 {
		t.Errorf("ExitNoRepo = %d, want 2", ExitNoRepo)
	}
	if ExitNoPRs != 3 {
		t.Errorf("ExitNoPRs = %d, want 3", ExitNoPRs)
	}
	if ExitUsage != 64 {
		t.Errorf("ExitUsage = %d, want 64", ExitUsage)
	}
}
