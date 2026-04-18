package cli

import (
	"errors"

	"github.com/gertzgal/gh-prs/internal/model"
)

const (
	ExitSuccess = 0
	ExitGhError = 1
	ExitNoRepo  = 2
	ExitNoPRs   = 3
	ExitUsage   = 64
)

// MapError maps any error to an exit code. Pass err=nil with hasPRs=false for
// the empty-results case (ExitNoPRs); hasPRs=true yields ExitSuccess.
func MapError(err error, hasPRs bool) int {
	if err == nil {
		if hasPRs {
			return ExitSuccess
		}
		return ExitNoPRs
	}
	var useErr *model.UsageError
	if errors.As(err, &useErr) {
		return ExitUsage
	}
	if errors.Is(err, model.ErrRepoNotFound) {
		return ExitNoRepo
	}
	var ghErr *model.GhError
	if errors.As(err, &ghErr) {
		return ExitGhError
	}
	return ExitGhError
}
