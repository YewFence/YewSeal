package main

import (
	"errors"
	"os"

	yewsapp "github.com/YewFence/YewSeal/internal/app"
	"github.com/YewFence/YewSeal/internal/config"
	tools "github.com/YewFence/YewSeal/internal/doctor"
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
		RunE: func(cmd *cobra.Command, args []string) error {
			return project.InitProject(force, input, output, format, createExample, skipSOPSConfig)
		},
	}
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force overwrite existing configuration")
	cmd.Flags().StringVarP(&input, "input", "i", "", "Plaintext file for the first config entry (non-interactive mode)")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Encrypted file for the first config entry (non-interactive mode)")
	cmd.Flags().StringVar(&format, "format", "", "Format override for the first config entry (toml/yaml/json/env/ini/binary)")
	cmd.Flags().BoolVar(&createExample, "create-example", false, "Create example file (non-interactive mode)")
	cmd.Flags().BoolVar(&skipSOPSConfig, "skip-sops-config", false, "Skip creating .sops.yaml file (non-interactive mode)")
	return cmd
}

func editCommand(cfg *config.Config, keyFile *string) *cobra.Command {
	var file string
	var editor string

	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Edit encrypted configuration file using SOPS",
		RunE: func(cmd *cobra.Command, args []string) error {
			return yewsapp.EditEncryptedFile(yewsapp.EditRequest{
				Config:  cfg,
				File:    file,
				Editor:  editor,
				KeyFile: *keyFile,
			})
		},
	}
	cmd.Flags().StringVarP(&file, "file", "f", "wrangler.enc.toml.yaml", "Encrypted file to edit")
	cmd.Flags().StringVarP(&editor, "editor", "e", "", "Editor command to use (e.g., 'code -w', 'vim')")
	return cmd
}

func viewCommand(cfg *config.Config, keyFile *string) *cobra.Command {
	var format string
	var verbose bool

	cmd := &cobra.Command{
		Use:   "view [command options] <target>",
		Short: "Print decrypted plaintext to standard output without writing files",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cliFormat, err := yewsapp.ValidateCLIFormatOverride(format)
			if err != nil {
				return err
			}

			if err := yewsapp.WriteViewedTarget(os.Stdout, cfg, args[0], cfg.GetKeyFile(*keyFile), cliFormat, verbose); err != nil {
				return err
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&format, "format", "", "Format override for the selected target (toml/yaml/json/env/ini/binary)")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	return cmd
}

func diffCommand(cfg *config.Config, keyFile *string) *cobra.Command {
	var format string
	var color string
	var verbose bool

	cmd := &cobra.Command{
		Use:   "diff [target]",
		Short: "Compare plaintext file with decrypted encrypted file",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cliFormat, err := yewsapp.ValidateCLIFormatOverride(format)
			if err != nil {
				return err
			}

			result, err := yewsapp.DiffPlaintextAgainstEncryptedTargets(os.Stdout, cfg, firstArg(args), cfg.GetKeyFile(*keyFile), cliFormat, verbose, color)
			if err != nil {
				return err
			}
			if result.Different {
				return errors.New("")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&format, "format", "", "Format override for the selected target (toml/yaml/json/env/ini/binary)")
	cmd.Flags().StringVar(&color, "color", "auto", "Colorize diff output (auto/always/never)")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	return cmd
}

func checkCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "check",
		Aliases: []string{"doctor"},
		Short:   "Check if required external tools are installed",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !tools.CheckToolsVerbose() {
				return errors.New("")
			}
			return nil
		},
	}
}
