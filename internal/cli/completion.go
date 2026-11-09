package cli

import (
	"path/filepath"
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
			if styled := inPrefixStyle(config.CurrentDir(cfg), path, toComplete); styled != "" {
				matches = append(matches, styled)
			}
		}
		if len(matches) == 0 {
			return nil, cobra.ShellCompDirectiveDefault
		}
		return matches, cobra.ShellCompDirectiveNoFileComp
	}
}

// inPrefixStyle reports whether the registered path matches the toComplete
// prefix and returns the suggestion in the user's own prefix style — plain
// relative, "./"-prefixed, or absolute — so "./config" and absolute prefixes
// complete like plain ones. Empty means no match.
func inPrefixStyle(cwd, path, toComplete string) string {
	switch {
	case toComplete == "":
		return path
	case filepath.IsAbs(toComplete):
		abs := path
		if !filepath.IsAbs(abs) {
			abs = filepath.Join(cwd, abs)
		}
		if strings.HasPrefix(abs, toComplete) {
			return abs
		}
		return ""
	case strings.HasPrefix(toComplete, "./"):
		if filepath.IsAbs(path) {
			return ""
		}
		if styled := "./" + path; strings.HasPrefix(styled, toComplete) {
			return styled
		}
		return ""
	default:
		if strings.HasPrefix(path, toComplete) {
			return path
		}
		return ""
	}
}
