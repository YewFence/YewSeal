package cli

import (
	"fmt"

	"github.com/YewFence/YewSeal/internal/task"
)

func validateBatchArgs(args []string, parallel int, outputSet bool) error {
	if parallel < 1 {
		return fmt.Errorf("--parallel must be at least 1")
	}
	for _, arg := range args {
		if err := validateTargetArg(arg); err != nil {
			return err
		}
	}
	if outputSet && len(args) != 1 {
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
