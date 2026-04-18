package cli

import (
	"bytes"
	"io"
	"os"
	"testing"
)

// TestExecute_InvalidFormat verifies that passing an unknown --format value
// returns the usage exit code and writes the usage block to stderr. Guards
// the contract documented in USAGE and prevents silent fallthrough to a
// default formatter on typos.
func TestExecute_InvalidFormat(t *testing.T) {
	origStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stderr = w
	t.Cleanup(func() { os.Stderr = origStderr })

	done := make(chan []byte, 1)
	go func() {
		buf, _ := io.ReadAll(r)
		done <- buf
	}()

	code := Execute([]string{"--format", "yaml"}, []string{})

	_ = w.Close()
	stderrOut := <-done

	if code != ExitUsage {
		t.Fatalf("exit code = %d, want %d", code, ExitUsage)
	}
	if !bytes.Contains(stderrOut, []byte(`unknown --format "yaml"`)) {
		t.Fatalf("stderr missing error message; got:\n%s", stderrOut)
	}
	if !bytes.Contains(stderrOut, []byte("Usage: gh prs")) {
		t.Fatalf("stderr missing USAGE block; got:\n%s", stderrOut)
	}
}
