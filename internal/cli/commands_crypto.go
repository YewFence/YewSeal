package cli

import (
	"os"

	yewsapp "github.com/YewFence/YewSeal/internal/app"
	"github.com/YewFence/YewSeal/internal/config"

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
			return validateBatchArgs(cmd, args, opts.Patterns, opts.Parallel)
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
				UpdateProjectMetadata: true,
			})
		}),
	}
	addDecryptFlags(cmd.Flags(), &opts)
	return cmd
}

func planCommand(load configLoader) *cobra.Command {
	opts := planOptions{
		Output:   envValue("SOPS_OUTPUT_FILE"),
		Parallel: 1,
	}

	cmd := &cobra.Command{
		Use:   "plan [command options] [path]",
		Short: "Run preflight and print the resolved file selection without writing files",
		Args: func(cmd *cobra.Command, args []string) error {
			return validateBatchArgs(cmd, args, opts.Patterns, opts.Parallel)
		},
		RunE: withConfig(load, func(cmd *cobra.Command, args []string, cfg *config.Config) error {
			target := firstArg(args)
			return yewsapp.PrintPlan(os.Stdout, cfg, yewsapp.PlanRequest{
				Verbose:   opts.Verbose,
				Output:    opts.Output,
				OutputSet: flagChangedOrEnvSet(cmd.Flags(), "output", "SOPS_OUTPUT_FILE"),
				Target:    target,
				Patterns:  opts.Patterns,
				Parallel:  opts.Parallel,
			}, yewsapp.PreflightPrintOptions{
				JSON:    opts.JSON,
				Verbose: opts.Verbose,
			})
		}),
	}
	addPlanFlags(cmd.Flags(), &opts)
	return cmd
}
