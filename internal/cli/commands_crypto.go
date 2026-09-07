package cli

import (
	yewsapp "github.com/YewFence/YewSeal/internal/app"
	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/task"

	"github.com/spf13/cobra"
)

func encryptCommand(load configLoader) *cobra.Command {
	opts := encryptOptions{
		Output:   envValue("SOPS_OUTPUT_FILE"),
		Parallel: 1,
	}

	cmd := &cobra.Command{
		Use:     "encrypt [command options] [path]",
		Aliases: []string{"e"},
		Short:   "Encrypt configuration file (supports .toml, .yaml, .yml, .json, .env, .ini, and binary output)",
		Args: func(cmd *cobra.Command, args []string) error {
			return validateBatchArgs(cmd, args, opts.Patterns, opts.Parallel)
		},
		RunE: withConfig(load, func(cmd *cobra.Command, args []string, cfg *config.Config) error {
			target := firstArg(args)
			return yewsapp.EncryptFiles(cfg, yewsapp.EncryptRequest{
				Verbose:               opts.Verbose,
				Output:                opts.Output,
				OutputSet:             flagChangedOrEnvSet(cmd.Flags(), "output", "SOPS_OUTPUT_FILE"),
				Target:                target,
				Patterns:              opts.Patterns,
				Parallel:              opts.Parallel,
				UpdateProjectMetadata: true,
			})
		}),
	}
	addEncryptFlags(cmd.Flags(), &opts)
	return cmd
}

func decryptCommand(load configLoader, keyFile *string) *cobra.Command {
	opts := decryptOptions{
		Output:   envValue("SOPS_OUTPUT_FILE"),
		Parallel: 1,
	}

	cmd := &cobra.Command{
		Use:     "decrypt [command options] [path]",
		Aliases: []string{"d"},
		Short:   "Decrypt encrypted file (output format determined by extension)",
		Args: func(cmd *cobra.Command, args []string) error {
			if err := validateBatchArgs(cmd, args, opts.Patterns, opts.Parallel); err != nil {
				return err
			}
			return resolveStrict(cmd, &opts.Strict)
		},
		RunE: withConfig(load, func(cmd *cobra.Command, args []string, cfg *config.Config) error {
			target := firstArg(args)
			return yewsapp.DecryptFiles(cfg, yewsapp.DecryptRequest{
				KeyFile:               *keyFile,
				Verbose:               opts.Verbose,
				Output:                opts.Output,
				OutputSet:             flagChangedOrEnvSet(cmd.Flags(), "output", "SOPS_OUTPUT_FILE"),
				Target:                target,
				Patterns:              opts.Patterns,
				Parallel:              opts.Parallel,
				Force:                 opts.Force,
				Strict:                opts.Strict,
				UpdateProjectMetadata: true,
			})
		}),
	}
	addDecryptFlags(cmd.Flags(), &opts)
	return cmd
}

func planCommand(load configLoader) *cobra.Command {
	opts := planOptions{}

	cmd := &cobra.Command{
		Use:   "plan [command options] [path]",
		Short: "Check configured file mappings, formats, and recipient authorization without writing files",
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.MaximumNArgs(1)(cmd, args); err != nil {
				return err
			}
			_, err := task.ParsePatternRules(opts.Patterns)
			return err
		},
		RunE: withConfig(load, func(cmd *cobra.Command, args []string, cfg *config.Config) error {
			target := firstArg(args)
			return yewsapp.PrintPlan(cmd.OutOrStdout(), cfg, yewsapp.PlanRequest{
				Target:   target,
				Patterns: opts.Patterns,
			}, yewsapp.PlanPrintOptions{
				JSON:    opts.JSON,
				Verbose: opts.Verbose,
			})
		}),
	}
	addPlanFlags(cmd.Flags(), &opts)
	return cmd
}
