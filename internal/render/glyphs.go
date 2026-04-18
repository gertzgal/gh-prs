package render

import (
	"fmt"

	"github.com/gertzgal/gh-prs/internal/model"
)

type tone int

const (
	toneOk tone = iota
	toneBad
	tonePending
	toneNone
)

func glyphFor(t tone, color bool) string {
	switch t {
	case toneOk:
		return fgGreen("✓", color)
	case toneBad:
		return fgRed("✗", color)
	case tonePending:
		return fgYellow("●", color)
	default:
		return fgGray("○", color)
	}
}

func ciTone(s model.CiState) tone {
	switch s {
	case model.CiSuccess:
		return toneOk
	case model.CiFailure, model.CiError:
		return toneBad
	case model.CiPending, model.CiExpected:
		return tonePending
	default:
		return toneNone
	}
}

func reviewTone(d model.ReviewDecision) tone {
	switch d {
	case model.ReviewApproved:
		return toneOk
	case model.ReviewChangesRequested:
		return toneBad
	case model.ReviewRequired:
		return tonePending
	default:
		return toneNone
	}
}

func ciStatus(s model.CiState, color bool) string {
	return glyphFor(ciTone(s), color) + " " + fgGray("ci", color)
}

func reviewStatus(d model.ReviewDecision, color bool) string {
	return glyphFor(reviewTone(d), color) + " " + fgGray("review", color)
}

func additions(p model.PR, color bool) string {
	return fgGreen(fmt.Sprintf("+%d", p.Additions), color) + fgRed(fmt.Sprintf("-%d", p.Deletions), color)
}
