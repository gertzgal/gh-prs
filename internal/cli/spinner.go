package cli

import (
	"fmt"
	"io"
	"time"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// spinnerInterval controls the frame cadence. Exposed as a package var so
// tests can shrink it — 80ms matches the TS spinner.
var spinnerInterval = 80 * time.Millisecond

// Spinner writes a braille frame to stderr every spinnerInterval when started
// against a TTY. Stop is idempotent and always clears the current line.
type Spinner struct {
	enabled bool
	out     io.Writer
	stop    chan struct{}
	done    chan struct{}
}

// NewSpinner constructs a spinner. enabled is the caller's intent (false for
// --json); stderrIsTTY short-circuits to a no-op when false.
func NewSpinner(enabled bool, stderrIsTTY bool, stderr io.Writer) *Spinner {
	return &Spinner{
		enabled: enabled && stderrIsTTY,
		out:     stderr,
		stop:    make(chan struct{}),
		done:    make(chan struct{}),
	}
}

func (s *Spinner) Start() {
	if !s.enabled {
		close(s.done)
		return
	}
	go func() {
		ticker := time.NewTicker(spinnerInterval)
		defer ticker.Stop()
		defer close(s.done)
		i := 0
		for {
			select {
			case <-s.stop:
				return
			case <-ticker.C:
				_, _ = fmt.Fprintf(s.out, "\r%s loading…", spinnerFrames[i])
				i = (i + 1) % len(spinnerFrames)
			}
		}
	}()
}

func (s *Spinner) Stop() {
	if !s.enabled {
		return
	}
	select {
	case <-s.stop:
	default:
		close(s.stop)
	}
	<-s.done
	_, _ = fmt.Fprint(s.out, "\r\x1b[2K")
}
