package cli

import (
	"fmt"
	"strings"

	yewsapp "github.com/YewFence/YewSeal/internal/app"
	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/presentation"
	"github.com/YewFence/YewSeal/internal/project"

	"github.com/spf13/cobra"
)

func initCommand() *cobra.Command {
	var force bool
	var input string
	var output string
	var format string
	var createExample bool
	var skipSOPSConfig bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize project with Age keys and YewSeal config entries",
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.NoArgs(cmd, args); err != nil {
				return err
			}
			_, err := yewsapp.ValidateCLIFormatOverride(format)
			return err
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			out := presentation.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), false)
			return project.InitProject(force, input, output, format, createExample, skipSOPSConfig,
				out, out.Prompts(cmd.InOrStdin()))
		},
	}
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Rebuild keys and configuration; existing ciphertext may become undecryptable")
	cmd.Flags().StringVarP(&input, "input", "i", "", "Plaintext file for the first config entry (non-interactive mode)")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Encrypted file for the first config entry (non-interactive mode)")
	cmd.Flags().StringVar(&format, "format", "", "Format override for the first config entry (toml/yaml/json/env/ini/binary)")
	cmd.Flags().BoolVar(&createExample, "create-example", false, "Create example file (non-interactive mode)")
	cmd.Flags().BoolVar(&skipSOPSConfig, "skip-sops-config", false, "Skip creating .sops.yaml file (non-interactive mode)")
	return cmd
}

func editCommand(load configLoader, keyFile *string) *cobra.Command {
	var file string

	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Edit encrypted configuration file using SOPS",
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.NoArgs(cmd, args); err != nil {
				return err
			}
			if strings.TrimSpace(file) == "" {
				return fmt.Errorf("edit requires exactly one configured target")
			}
			return nil
		},
		RunE: withConfig(load, func(cmd *cobra.Command, args []string, cfg *config.Config) error {
			return yewsapp.EditEncryptedFile(yewsapp.EditRequest{
				Presentation: presentation.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), false),
				Config:       cfg,
				File:         file,
				KeyFile:      *keyFile,
			})
		}),
	}
	cmd.Flags().StringVarP(&file, "file", "f", "", "Encrypted file to edit (must be configured in .yewseal.toml)")
	return cmd
}

func viewCommand(load configLoader, keyFile *string) *cobra.Command {
	var verbose bool

	cmd := &cobra.Command{
		Use:   "view [command options] <target>",
		Short: "Print decrypted plaintext to standard output without writing files",
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.ExactArgs(1)(cmd, args); err != nil {
				return err
			}
			if strings.TrimSpace(args[0]) == "" {
				return fmt.Errorf("view requires exactly one target")
			}
			return nil
		},
		RunE: withConfig(load, func(cmd *cobra.Command, args []string, cfg *config.Config) error {
			return yewsapp.WriteViewedTarget(cmd.OutOrStdout(), cmd.ErrOrStderr(), cfg, args[0], *keyFile, verbose)
		}),
	}
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	return cmd
}

func diffCommand(load configLoader, keyFile *string) *cobra.Command {
	var color string
	var verbose bool
	var strict bool

	cmd := &cobra.Command{
		Use:   "diff [path-or-pattern]...",
		Short: "Compare plaintext file with decrypted encrypted file",
		Args: func(cmd *cobra.Command, args []string) error {
			for _, arg := range args {
				if err := validateTargetArg(arg); err != nil {
					return err
				}
			}
			_, err := presentation.ResolveDiffColor(color, cmd.OutOrStdout())
			if err != nil {
				return err
			}
			return resolveStrict(cmd, &strict)
		},
		RunE: withConfig(load, func(cmd *cobra.Command, args []string, cfg *config.Config) error {
			_, err := yewsapp.DiffPlaintextAgainstEncryptedTargets(cmd.OutOrStdout(), cmd.ErrOrStderr(), cfg, args, *keyFile, verbose, color, strict)
			return err
		}),
	}
	cmd.Flags().StringVar(&color, "color", "auto", "Colorize diff output (auto/always/never)")
	cmd.Flags().BoolVar(&strict, "strict", false, "Require comparison of files with both inputs present (default from YEWSEAL_STRICT)")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	return cmd
}
