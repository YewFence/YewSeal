package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	yewsapp "github.com/YewFence/YewSeal/internal/app"
	"github.com/YewFence/YewSeal/internal/config"
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
			return project.InitProject(force, input, output, format, createExample, skipSOPSConfig)
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
				Config:  cfg,
				File:    file,
				KeyFile: *keyFile,
			})
		}),
	}
	cmd.Flags().StringVarP(&file, "file", "f", "", "Encrypted file to edit (must be configured in .yewseal.toml)")
	return cmd
}

func viewCommand(load configLoader, keyFile *string) *cobra.Command {
	var format string
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
			_, err := yewsapp.ValidateCLIFormatOverride(format)
			return err
		},
		RunE: withConfig(load, func(cmd *cobra.Command, args []string, cfg *config.Config) error {
			return yewsapp.WriteViewedTarget(os.Stdout, cfg, args[0], *keyFile, format, verbose)
		}),
	}
	cmd.Flags().StringVar(&format, "format", "", "Format override for the selected target (toml/yaml/json/env/ini/binary)")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	return cmd
}

func diffCommand(load configLoader, keyFile *string) *cobra.Command {
	var format string
	var color string
	var verbose bool

	cmd := &cobra.Command{
		Use:   "diff [target]",
		Short: "Compare plaintext file with decrypted encrypted file",
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.MaximumNArgs(1)(cmd, args); err != nil {
				return err
			}
			cliFormat, err := yewsapp.ValidateCLIFormatOverride(format)
			if err != nil {
				return err
			}
			if strings.TrimSpace(firstArg(args)) == "" && cliFormat != "" {
				return fmt.Errorf("--format is only supported in single-file mode")
			}
			_, err = yewsapp.ResolveDiffColor(color, os.Stdout)
			return err
		},
		RunE: withConfig(load, func(cmd *cobra.Command, args []string, cfg *config.Config) error {
			result, err := yewsapp.DiffPlaintextAgainstEncryptedTargets(os.Stdout, cfg, firstArg(args), *keyFile, format, verbose, color)
			if err != nil {
				return err
			}
			if result.Different {
				return errors.New("")
			}
			return nil
		}),
	}
	cmd.Flags().StringVar(&format, "format", "", "Format override for the selected target (toml/yaml/json/env/ini/binary)")
	cmd.Flags().StringVar(&color, "color", "auto", "Colorize diff output (auto/always/never)")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	return cmd
}
