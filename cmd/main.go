package main

import (
	"fmt"
	"os"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/crypto"
	"github.com/YewFence/YewSeal/internal/sync"
	"github.com/YewFence/YewSeal/internal/tools"

	"github.com/urfave/cli/v2"
)

func main() {
	// Load configuration file
	cfg, err := config.LoadConfig()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	app := &cli.App{
		Name:    "yews",
		Usage:   "YewSeal - Encrypt/decrypt configuration files using SOPS, Age, and yq",
		Version: "1.0.0",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "key-file",
				Aliases: []string{"k"},
				Usage:   "Path to Age private key file",
				EnvVars: []string{"AGE_KEY_FILE"},
			},
		},
		Commands: []*cli.Command{
			{
				Name:  "init",
				Usage: "Initialize project with Age keys and SOPS configuration",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "force",
						Aliases: []string{"f"},
						Usage:   "Force overwrite existing configuration",
					},
					&cli.StringFlag{
						Name:    "input",
						Aliases: []string{"i"},
						Usage:   "Original configuration file name (non-interactive mode)",
					},
					&cli.StringFlag{
						Name:    "output",
						Aliases: []string{"o"},
						Usage:   "Encrypted output file name (non-interactive mode)",
					},
					&cli.BoolFlag{
						Name:  "create-example",
						Usage: "Create example file (non-interactive mode)",
					},
					&cli.BoolFlag{
						Name:  "skip-sops-config",
						Usage: "Skip creating .sops.yaml file (non-interactive mode)",
					},
				},
				Action: func(c *cli.Context) error {
					return crypto.InitProject(
						c.Bool("force"),
						c.String("input"),
						c.String("output"),
						c.Bool("create-example"),
						c.Bool("skip-sops-config"),
					)
				},
			},
			{
				Name:    "encrypt",
				Aliases: []string{"e"},
				Usage:   "Encrypt wrangler.toml to wrangler.enc.yaml",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "input",
						Aliases: []string{"i"},
						Value:   "wrangler.toml",
						Usage:   "Input TOML file",
						EnvVars: []string{"SOPS_INPUT_FILE"},
					},
					&cli.StringFlag{
						Name:    "output",
						Aliases: []string{"o"},
						Value:   "wrangler.enc.toml.yaml",
						Usage:   "Output encrypted YAML file",
						EnvVars: []string{"SOPS_OUTPUT_FILE"},
					},
					&cli.StringFlag{
						Name:    "public-key",
						Aliases: []string{"p"},
						Usage:   "Age public key for encryption",
						EnvVars: []string{"SOPS_AGE_RECIPIENTS"},
					},
					&cli.BoolFlag{
						Name:    "verbose",
						Aliases: []string{"v"},
						Usage:   "Enable verbose output",
					},
				},
				Action: func(c *cli.Context) error {
					// Priority: CLI args > env vars > config file > defaults
					inputFile := cfg.GetEncryptionInput(c.String("input"))
					outputFile := cfg.GetEncryptionOutput(c.String("output"))
					keyFile := cfg.GetKeyFile(c.String("key-file"))
					publicKey := c.String("public-key")
					verbose := c.Bool("verbose")

					return crypto.Encrypt(inputFile, outputFile, keyFile, publicKey, verbose)
				},
			},
			{
				Name:    "decrypt",
				Aliases: []string{"d"},
				Usage:   "Decrypt wrangler.enc.yaml to wrangler.toml",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "input",
						Aliases: []string{"i"},
						Value:   "wrangler.enc.toml.yaml",
						Usage:   "Input encrypted YAML file",
						EnvVars: []string{"SOPS_INPUT_FILE"},
					},
					&cli.StringFlag{
						Name:    "output",
						Aliases: []string{"o"},
						Value:   "wrangler.toml",
						Usage:   "Output TOML file",
						EnvVars: []string{"SOPS_OUTPUT_FILE"},
					},
					&cli.BoolFlag{
						Name:    "verbose",
						Aliases: []string{"v"},
						Usage:   "Enable verbose output",
					},
				},
				Action: func(c *cli.Context) error {
					// Priority: CLI args > env vars > config file > defaults
					inputFile := cfg.GetDecryptionInput(c.String("input"))
					outputFile := cfg.GetDecryptionOutput(c.String("output"))
					keyFile := cfg.GetKeyFile(c.String("key-file"))
					verbose := c.Bool("verbose")

					return crypto.Decrypt(inputFile, outputFile, keyFile, verbose)
				},
			},
			{
				Name:  "edit",
				Usage: "Edit encrypted configuration file using SOPS",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "file",
						Aliases: []string{"f"},
						Value:   "wrangler.enc.toml.yaml",
						Usage:   "Encrypted file to edit",
					},
					&cli.StringFlag{
						Name:    "editor",
						Aliases: []string{"e"},
						Usage:   "Editor command to use (e.g., 'code -w', 'vim')",
					},
				},
				Action: func(c *cli.Context) error {
					keyFile := c.String("key-file")
					return crypto.Edit(c.String("file"), c.String("editor"), keyFile)
				},
			},
			{
				Name:    "check",
				Aliases: []string{"doctor"},
				Usage:   "Check if required external tools are installed",
				Action: func(c *cli.Context) error {
					if !tools.CheckToolsVerbose() {
						return cli.Exit("", 1)
					}
					return nil
				},
			},
			{
				Name:  "sync",
				Usage: "Sync sensitive files to secret management service",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "key-file",
						Aliases: []string{"k"},
						Value:   ".age/keys.txt",
						Usage:   "Path to the key file to sync",
					},
					&cli.StringFlag{
						Name:    "name",
						Aliases: []string{"n"},
						Value:   "AGE_KEY_FILE",
						Usage:   "Secret name in the provider",
					},
					&cli.StringFlag{
						Name:  "path",
						Usage: "Path/folder in the provider (e.g., /yewseal)",
					},
					&cli.StringFlag{
						Name:    "provider",
						Aliases: []string{"p"},
						Value:   "infisical",
						Usage:   "Secret management provider (infisical)",
					},
				},
				Action: func(c *cli.Context) error {
					return sync.SyncKey(
						c.String("provider"),
						c.String("key-file"),
						c.String("name"),
						c.String("path"),
					)
				},
			},
		},
		Before: func(c *cli.Context) error {
			// Skip tool check for commands that don't need them
			cmd := c.Args().First()
			if cmd == "check" || cmd == "doctor" || cmd == "sync" {
				return nil
			}
			if err := tools.CheckTools(); err != nil {
				return err
			}
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
