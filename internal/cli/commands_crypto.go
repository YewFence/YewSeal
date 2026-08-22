package cli

import (
	"os"

	yewsapp "github.com/YewFence/YewSeal/internal/app"
	"github.com/YewFence/YewSeal/internal/config"

	"github.com/spf13/cobra"
)

func encryptCommand(cfg *config.Config, keyFile *string) *cobra.Command {
	opts := encryptOptions{
		Output:    envValue("SOPS_OUTPUT_FILE"),
		Format:    envValue("YEWSEAL_FORMAT", "SOPS_FORMAT"),
		PublicKey: envValue("SOPS_AGE_RECIPIENTS"),
		Parallel:  1,
	}

	cmd := &cobra.Command{
		Use:     "encrypt [command options] [path]",
		Aliases: []string{"e"},
		Short:   "Encrypt configuration file (supports .toml, .yaml, .yml, .json, .env, .ini, and binary output)",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := firstArg(args)
			return yewsapp.EncryptFiles(cfg, yewsapp.EncryptRequest{
				KeyFile:               cfg.GetKeyFile(*keyFile),
				PublicKey:             opts.PublicKey,
				Verbose:               opts.Verbose,
				Output:                opts.Output,
				OutputSet:             flagChangedOrEnvSet(cmd.Flags(), "output", "SOPS_OUTPUT_FILE"),
				Format:                opts.Format,
				Target:                target,
				Patterns:              opts.Patterns,
				FormatRules:           opts.FormatRules,
				UnknownAsBinary:       opts.UnknownAsBinary,
				UnknownAsBinarySet:    cmd.Flags().Changed("unknown-as-binary"),
				Parallel:              opts.Parallel,
				UpdateProjectMetadata: true,
			})
		},
	}
	addEncryptFlags(cmd.Flags(), &opts)
	return cmd
}

func decryptCommand(cfg *config.Config, keyFile *string) *cobra.Command {
	opts := decryptOptions{
		Output:   envValue("SOPS_OUTPUT_FILE"),
		Format:   envValue("YEWSEAL_FORMAT", "SOPS_FORMAT"),
		Parallel: 1,
	}

	cmd := &cobra.Command{
		Use:     "decrypt [command options] [path]",
		Aliases: []string{"d"},
		Short:   "Decrypt encrypted file (output format determined by extension)",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := firstArg(args)
			return yewsapp.DecryptFiles(cfg, yewsapp.DecryptRequest{
				KeyFile:               cfg.GetKeyFile(*keyFile),
				Verbose:               opts.Verbose,
				Output:                opts.Output,
				OutputSet:             flagChangedOrEnvSet(cmd.Flags(), "output", "SOPS_OUTPUT_FILE"),
				Format:                opts.Format,
				Target:                target,
				Patterns:              opts.Patterns,
				FormatRules:           opts.FormatRules,
				UnknownAsBinary:       opts.UnknownAsBinary,
				UnknownAsBinarySet:    cmd.Flags().Changed("unknown-as-binary"),
				Parallel:              opts.Parallel,
				Force:                 opts.Force,
				UpdateProjectMetadata: true,
			})
		},
	}
	addDecryptFlags(cmd.Flags(), &opts)
	return cmd
}

func planCommand(cfg *config.Config) *cobra.Command {
	opts := planOptions{
		Output:   envValue("SOPS_OUTPUT_FILE"),
		Format:   envValue("YEWSEAL_FORMAT", "SOPS_FORMAT"),
		Parallel: 1,
	}

	cmd := &cobra.Command{
		Use:   "plan [command options] [path]",
		Short: "Run preflight and print the resolved file selection without writing files",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := firstArg(args)
			return yewsapp.PrintPlan(os.Stdout, cfg, yewsapp.PlanRequest{
				Verbose:            opts.Verbose,
				Output:             opts.Output,
				OutputSet:          flagChangedOrEnvSet(cmd.Flags(), "output", "SOPS_OUTPUT_FILE"),
				Format:             opts.Format,
				Target:             target,
				Patterns:           opts.Patterns,
				FormatRules:        opts.FormatRules,
				UnknownAsBinary:    opts.UnknownAsBinary,
				UnknownAsBinarySet: cmd.Flags().Changed("unknown-as-binary"),
				Parallel:           opts.Parallel,
			}, yewsapp.PreflightPrintOptions{
				JSON:    opts.JSON,
				Verbose: opts.Verbose,
			})
		},
	}
	addPlanFlags(cmd.Flags(), &opts)
	return cmd
}
