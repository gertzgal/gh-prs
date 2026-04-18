package cli

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
)

func TestHelpTemplateMatchesUSAGE(t *testing.T) {
	var buf bytes.Buffer
	cmd := &cobra.Command{Use: "gh-prs"}
	cmd.SetOut(&buf)
	cmd.SetHelpFunc(func(_ *cobra.Command, _ []string) {
		buf.WriteString(USAGE)
	})
	if err := cmd.Help(); err != nil {
		t.Fatalf("cmd.Help() returned error: %v", err)
	}
	if got := buf.String(); got != USAGE {
		t.Fatalf("help output does not match USAGE:\n--- got ---\n%q\n--- want ---\n%q", got, USAGE)
	}
}
