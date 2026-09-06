package cli

import (
	"fmt"
	"strings"

	"github.com/YewFence/YewSeal/internal/task"
	"github.com/spf13/cobra"
)

func validateBatchArgs(cmd *cobra.Command, args []string, patterns []string, parallel int) error {
	if err := cobra.MaximumNArgs(1)(cmd, args); err != nil {
		return err
	}
	if parallel < 1 {
		return fmt.Errorf("--parallel must be at least 1")
	}
	if _, err := task.ParsePatternRules(patterns); err != nil {
		return err
	}
	if strings.TrimSpace(firstArg(args)) == "" {
		if flagChangedOrEnvSet(cmd.Flags(), "output", "SOPS_OUTPUT_FILE") {
			return fmt.Errorf("--output is only supported when the path target is a file")
		}
	}
	return nil
}
