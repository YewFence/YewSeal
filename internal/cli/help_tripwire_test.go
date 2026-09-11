package cli

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// Tripwire: the CLI is self-explanatory. Every business command must carry a
// Long synopsis, runnable examples, and a documentation deep-link, so --help
// and the generated reference pages remain the single source of command
// truth (docs/commands/ is gone by design).
func TestCommandsAreSelfDocumented(t *testing.T) {
	root := newRootCommand("test", nil)

	if strings.TrimSpace(root.Long) == "" {
		t.Fatal("root command must have a Long description")
	}
	if !strings.Contains(root.Long, docsBaseURL) {
		t.Fatal("root command Long must link the documentation site")
	}

	var walk func(cmd *cobra.Command)
	walk = func(cmd *cobra.Command) {
		for _, sub := range cmd.Commands() {
			// help/completion are cobra-generated and exempt.
			if sub.Name() == "help" || sub.Name() == "completion" {
				continue
			}
			if strings.TrimSpace(sub.Long) == "" {
				t.Errorf("%s: Long description must not be empty", sub.CommandPath())
			}
			if strings.TrimSpace(sub.Example) == "" {
				t.Errorf("%s: Example must not be empty", sub.CommandPath())
			}
			if !strings.Contains(sub.Long, docsBaseURL) {
				t.Errorf("%s: Long must deep-link the documentation site", sub.CommandPath())
			}
			walk(sub)
		}
	}
	walk(root)
}
