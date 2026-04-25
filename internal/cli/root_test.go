package cli

import (
	"bytes"
	"testing"
)

func TestBuildClientOptions_DebugWiresStderr(t *testing.T) {
	flags := Flags{Debug: true}
	var stderr bytes.Buffer

	got := buildClientOptions(flags, map[string]string{}, &stderr, true)

	if !got.Debug {
		t.Errorf("Debug: want true, got false")
	}
	if got.DebugOut != &stderr {
		t.Errorf("DebugOut: want &stderr, got %v", got.DebugOut)
	}
	// TTY + no NO_COLOR => color on.
	if !got.DebugColor {
		t.Errorf("DebugColor: want true (TTY + no NO_COLOR), got false")
	}
}

func TestBuildClientOptions_DebugNoColorWhenNonTTY(t *testing.T) {
	flags := Flags{Debug: true}
	var stderr bytes.Buffer

	got := buildClientOptions(flags, map[string]string{}, &stderr, false)

	if got.DebugColor {
		t.Errorf("DebugColor: want false (non-TTY stderr), got true")
	}
}

func TestBuildClientOptions_ZeroValue(t *testing.T) {
	flags := Flags{}
	var stderr bytes.Buffer

	got := buildClientOptions(flags, map[string]string{}, &stderr, false)

	if got.Debug {
		t.Errorf("Debug: want false by default, got true")
	}
	if got.DebugOut != nil {
		t.Errorf("DebugOut: want nil by default, got %v", got.DebugOut)
	}
}
