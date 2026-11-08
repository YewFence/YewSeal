package cli

import (
	"strings"

	"github.com/YewFence/YewSeal/internal/config"

	"github.com/spf13/cobra"
)

// registeredTargetCompletion completes only the registered paths that target
// selection accepts for the command, instead of every file on disk. maxArgs
// caps the completed positional slots (0 = unlimited). Missing or invalid
// config falls back to the shell's default file completion; completion must
// never turn into an error in the user's shell.
func registeredTargetCompletion(load configLoader, command string, maxArgs int) func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return func(_ *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if maxArgs > 0 && len(args) >= maxArgs {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		cfg, err := load()
		if err != nil {
			return nil, cobra.ShellCompDirectiveDefault
		}
		paths, err := config.CompletionPaths(cfg, command)
		if err != nil {
			return nil, cobra.ShellCompDirectiveDefault
		}
		matches := make([]string, 0, len(paths))
		for _, path := range paths {
			if strings.HasPrefix(path, toComplete) {
				matches = append(matches, path)
			}
		}
		return matches, cobra.ShellCompDirectiveNoFileComp
	}
}
