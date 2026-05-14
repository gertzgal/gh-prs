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

func glyphFor(t tone, s styles) string {
	switch t {
	case toneOk:
		return s.Green.Render("✓")
	case toneBad:
		return s.Red.Render("✗")
	case tonePending:
		return s.Yellow.Render("●")
	default:
		return s.Gray.Render("○")
	}
}

func ciTone(st model.CiState) tone {
	switch st {
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

func ciStatus(st model.CiState, s styles) string {
	return glyphFor(ciTone(st), s) + " " + s.Gray.Render("ci")
}

func reviewStatus(d model.ReviewDecision, s styles) string {
	return glyphFor(reviewTone(d), s) + " " + s.Gray.Render("review")
}

func additions(p model.PR, s styles) string {
	return s.Green.Render(fmt.Sprintf("+%d", p.Additions)) +
		s.Red.Render(fmt.Sprintf("-%d", p.Deletions))
}
