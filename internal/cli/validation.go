package cli

import (
	"fmt"
	"os"
	"strconv"

	"github.com/YewFence/YewSeal/internal/task"
	"github.com/spf13/cobra"
)

func validateBatchArgs(cmd *cobra.Command, args []string, parallel int) error {
	if parallel < 1 {
		return fmt.Errorf("--parallel must be at least 1")
	}
	for _, arg := range args {
		if err := validateTargetArg(arg); err != nil {
			return err
		}
	}
	if flagChangedOrEnvSet(cmd.Flags(), "output", "SOPS_OUTPUT_FILE") && len(args) != 1 {
		return fmt.Errorf("--output is only supported with exactly one file target")
	}
	return nil
}

// validateTargetArg 校验单个位置参数：含 glob 元字符的参数必须是合法模式。
func validateTargetArg(arg string) error {
	if !task.HasPatternMeta(arg) {
		return nil
	}
	if _, err := task.NewPatternMatcher([]string{arg}); err != nil {
		return fmt.Errorf("invalid target pattern %q: %w", arg, err)
	}
	return nil
}

func resolveStrict(cmd *cobra.Command, strict *bool) error {
	if cmd.Flags().Changed("strict") {
		return nil
	}
	*strict = false
	value, set := os.LookupEnv("YEWSEAL_STRICT")
	if !set {
		return nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fmt.Errorf("invalid YEWSEAL_STRICT: expected a boolean (true/false)")
	}
	*strict = parsed
	return nil
}
