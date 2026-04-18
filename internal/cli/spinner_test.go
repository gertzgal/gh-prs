package cli

import (
	"bytes"
	"strings"
	"sync"
	"testing"
	"time"
)

// syncBuffer is a goroutine-safe bytes.Buffer for capturing spinner output.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func TestSpinner_DisabledByFlag(t *testing.T) {
	var buf bytes.Buffer
	s := NewSpinner(false, true, &buf)
	s.Start()
	s.Stop()
	if buf.Len() != 0 {
		t.Fatalf("disabled spinner wrote %q", buf.String())
	}
}

func TestSpinner_DisabledByNonTTY(t *testing.T) {
	var buf bytes.Buffer
	s := NewSpinner(true, false, &buf)
	s.Start()
	s.Stop()
	if buf.Len() != 0 {
		t.Fatalf("non-TTY spinner wrote %q", buf.String())
	}
}

func TestSpinner_StopIsIdempotent(t *testing.T) {
	var buf bytes.Buffer
	s := NewSpinner(false, false, &buf)
	s.Start()
	s.Stop()
	s.Stop() // must not panic or block
}

func TestSpinner_WritesFramesWhenEnabled(t *testing.T) {
	saved := spinnerInterval
	spinnerInterval = 5 * time.Millisecond
	t.Cleanup(func() { spinnerInterval = saved })

	buf := &syncBuffer{}
	s := NewSpinner(true, true, buf)
	s.Start()
	time.Sleep(40 * time.Millisecond)
	s.Stop()

	out := buf.String()
	if !strings.Contains(out, "loading…") {
		t.Fatalf("expected spinner frames in output, got %q", out)
	}
	if !strings.HasSuffix(out, "\r\x1b[2K") {
		t.Fatalf("expected clear-line suffix, got %q", out)
	}
}
