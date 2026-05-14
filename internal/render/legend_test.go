package render

import (
	"strings"
	"testing"
)

func TestLegend_SuppressedUnderEighty(t *testing.T) {
	got := renderLegend(60, newStyles(false))
	if got != "" {
		t.Errorf("legend should be suppressed at width=60; got:\n%s", got)
	}
}

func TestLegend_VisibleAtNinety(t *testing.T) {
	got := renderLegend(90, newStyles(false))
	if !strings.Contains(got, "stack") {
		t.Errorf("expected legend at width=90 to contain 'stack'; got:\n%s", got)
	}
	if !strings.Contains(got, "[review]") {
		t.Errorf("expected legend at width=90 to contain '[review]' chip; got:\n%s", got)
	}
	if !strings.Contains(got, "[changes]") {
		t.Errorf("expected legend at width=90 to contain '[changes]' chip; got:\n%s", got)
	}
}

func TestLegend_VisibleAtOneTen(t *testing.T) {
	got := renderLegend(110, newStyles(false))
	if !strings.Contains(got, "stack") {
		t.Errorf("expected legend at width=110 to contain 'stack'; got:\n%s", got)
	}
}

func TestLegend_ZeroWidthTreatedAsWide(t *testing.T) {
	got := renderLegend(0, newStyles(false))
	if !strings.Contains(got, "stack") {
		t.Errorf("expected legend at width=0 (wide) to contain 'stack'; got:\n%s", got)
	}
}
