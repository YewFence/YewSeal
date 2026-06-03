package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	yewsapp "github.com/YewFence/YewSeal/internal/app"
	"github.com/YewFence/YewSeal/internal/config"
	tools "github.com/YewFence/YewSeal/internal/doctor"
	"github.com/YewFence/YewSeal/internal/project"
	"github.com/YewFence/YewSeal/internal/seal"
	"github.com/YewFence/YewSeal/internal/sync"

	"github.com/urfave/cli/v2"
)

var Version = "dev"

func main() {
	// Load configuration file
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to load config: %v\n", err)
		os.Exit(1)
	}

	app := &cli.App{
		Name:    "yews",
		Usage:   "YewSeal - Encrypt/decrypt configuration files using SOPS and Age (supports TOML, YAML, JSON, ENV, INI)",
		Version: Version,
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
				Usage: "Initialize project with Age keys and YewSeal config entries",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "force",
						Aliases: []string{"f"},
						Usage:   "Force overwrite existing configuration",
					},
					&cli.StringFlag{
						Name:    "input",
						Aliases: []string{"i"},
						Usage:   "Plaintext file for the first config entry (non-interactive mode)",
					},
					&cli.StringFlag{
						Name:    "output",
						Aliases: []string{"o"},
						Usage:   "Encrypted file for the first config entry (non-interactive mode)",
					},
					&cli.StringFlag{
						Name:  "format",
						Usage: "Format override for the first config entry (toml/yaml/json/env/ini/binary)",
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
					return project.InitProject(
						c.Bool("force"),
						c.String("input"),
						c.String("output"),
						c.String("format"),
						c.Bool("create-example"),
						c.Bool("skip-sops-config"),
					)
				},
			},
			{
				Name:      "encrypt",
				Aliases:   []string{"e"},
				Usage:     "Encrypt configuration file (supports .toml, .yaml, .yml, .json, .env, .ini)",
				UsageText: `yews encrypt [command options]`,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "input",
						Aliases: []string{"i"},
						Value:   "wrangler.toml",
						Usage:   "Input file to encrypt (single file mode)",
						EnvVars: []string{"SOPS_INPUT_FILE"},
					},
					&cli.StringFlag{
						Name:    "output",
						Aliases: []string{"o"},
						Value:   "wrangler.enc.toml.yaml",
						Usage:   "Output encrypted file (single file mode only)",
						EnvVars: []string{"SOPS_OUTPUT_FILE"},
					},
					&cli.StringFlag{
						Name:  "format",
						Usage: "Format override for single-file mode (toml/yaml/json/env/ini/binary)",
					},
					&cli.StringFlag{
						Name:  "dir",
						Usage: "Directory to scan for files (enables batch mode)",
					},
					&cli.StringFlag{
						Name:  "pattern",
						Value: "*.toml",
						Usage: "Glob pattern for matching files in directory",
					},
					&cli.StringFlag{
						Name:  "output-dir",
						Usage: "Output directory for encrypted files (batch mode)",
					},
					&cli.StringFlag{
						Name:  "output-suffix",
						Value: ".enc.toml.yaml",
						Usage: "Suffix for output files (batch mode)",
					},
					&cli.IntFlag{
						Name:    "parallel",
						Aliases: []string{"P"},
						Value:   1,
						Usage:   "Number of parallel workers for batch mode",
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
					return yewsapp.EncryptFiles(cfg, yewsapp.EncryptRequest{
						KeyFile:               cfg.GetKeyFile(c.String("key-file")),
						PublicKey:             c.String("public-key"),
						Verbose:               c.Bool("verbose"),
						Input:                 c.String("input"),
						InputSet:              c.IsSet("input"),
						Output:                c.String("output"),
						OutputSet:             c.IsSet("output"),
						Format:                c.String("format"),
						Dir:                   c.String("dir"),
						Pattern:               c.String("pattern"),
						OutputDir:             c.String("output-dir"),
						OutputSuffix:          c.String("output-suffix"),
						Parallel:              c.Int("parallel"),
						UpdateProjectMetadata: true,
					})
				},
			},
			{
				Name:      "decrypt",
				Aliases:   []string{"d"},
				Usage:     "Decrypt encrypted file (output format determined by extension)",
				UsageText: `yews decrypt [command options]`,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "input",
						Aliases: []string{"i"},
						Value:   "wrangler.enc.toml.yaml",
						Usage:   "Input encrypted file (single file mode)",
						EnvVars: []string{"SOPS_INPUT_FILE"},
					},
					&cli.StringFlag{
						Name:    "output",
						Aliases: []string{"o"},
						Value:   "wrangler.toml",
						Usage:   "Output decrypted file (single file mode only)",
						EnvVars: []string{"SOPS_OUTPUT_FILE"},
					},
					&cli.StringFlag{
						Name:  "format",
						Usage: "Format override for single-file mode (toml/yaml/json/env/ini/binary)",
					},
					&cli.StringFlag{
						Name:  "dir",
						Usage: "Directory to scan for encrypted files (enables batch mode)",
					},
					&cli.StringFlag{
						Name:  "pattern",
						Value: "*.enc.toml.yaml",
						Usage: "Glob pattern for matching encrypted files",
					},
					&cli.StringFlag{
						Name:  "output-dir",
						Usage: "Output directory for decrypted files (batch mode)",
					},
					&cli.StringFlag{
						Name:  "output-suffix",
						Value: ".toml",
						Usage: "Suffix for output files (batch mode)",
					},
					&cli.IntFlag{
						Name:    "parallel",
						Aliases: []string{"P"},
						Value:   1,
						Usage:   "Number of parallel workers for batch mode",
					},
					&cli.BoolFlag{
						Name:    "force",
						Aliases: []string{"f"},
						Usage:   "Force overwrite existing plaintext file when it differs from decrypted content",
					},
					&cli.BoolFlag{
						Name:    "verbose",
						Aliases: []string{"v"},
						Usage:   "Enable verbose output",
					},
				},
				Action: func(c *cli.Context) error {
					return yewsapp.DecryptFiles(cfg, yewsapp.DecryptRequest{
						KeyFile:               cfg.GetKeyFile(c.String("key-file")),
						Verbose:               c.Bool("verbose"),
						Input:                 c.String("input"),
						InputSet:              c.IsSet("input"),
						Output:                c.String("output"),
						OutputSet:             c.IsSet("output"),
						Format:                c.String("format"),
						Dir:                   c.String("dir"),
						Pattern:               c.String("pattern"),
						OutputDir:             c.String("output-dir"),
						OutputSuffix:          c.String("output-suffix"),
						Parallel:              c.Int("parallel"),
						Force:                 c.Bool("force"),
						UpdateProjectMetadata: true,
					})
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
					return seal.Edit(seal.EditOptions{
						File:    c.String("file"),
						Editor:  c.String("editor"),
						KeyFile: keyFile,
					})
				},
			},
			{
				Name:      "view",
				Usage:     "Print decrypted plaintext to standard output without writing files",
				UsageText: `yews view [command options] <target>`,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "format",
						Usage: "Format override for the selected target (toml/yaml/json/env/ini/binary)",
					},
					&cli.BoolFlag{
						Name:    "verbose",
						Aliases: []string{"v"},
						Usage:   "Enable verbose output",
					},
				},
				Action: func(c *cli.Context) error {
					if c.Args().Len() != 1 {
						return cli.Exit("view requires exactly one target", 2)
					}

					keyFile := cfg.GetKeyFile(c.String("key-file"))
					verbose := c.Bool("verbose")
					cliFormat, err := yewsapp.ValidateCLIFormatOverride(c.String("format"))
					if err != nil {
						return cli.Exit(err, 2)
					}

					if err := yewsapp.WriteViewedTarget(os.Stdout, cfg, c.Args().First(), keyFile, cliFormat, verbose); err != nil {
						return cli.Exit(err, 2)
					}
					return nil
				},
			},
			{
				Name:      "diff",
				Usage:     "Compare plaintext file with decrypted encrypted file",
				UsageText: `yews diff [target]`,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "format",
						Usage: "Format override for the selected target (toml/yaml/json/env/ini/binary)",
					},
					&cli.StringFlag{
						Name:  "color",
						Value: "auto",
						Usage: "Colorize diff output (auto/always/never)",
					},
					&cli.BoolFlag{
						Name:    "verbose",
						Aliases: []string{"v"},
						Usage:   "Enable verbose output",
					},
				},
				Action: func(c *cli.Context) error {
					if c.Args().Len() > 1 {
						return cli.Exit("diff accepts at most one target", 2)
					}

					keyFile := cfg.GetKeyFile(c.String("key-file"))
					verbose := c.Bool("verbose")
					cliFormat, err := yewsapp.ValidateCLIFormatOverride(c.String("format"))
					if err != nil {
						return cli.Exit(err, 2)
					}

					result, err := yewsapp.DiffPlaintextAgainstEncryptedTargets(os.Stdout, cfg, c.Args().First(), keyFile, cliFormat, verbose, c.String("color"))
					if err != nil {
						return cli.Exit(err, 2)
					}
					if result.Different {
						return cli.Exit("", 1)
					}
					return nil
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
				Name:   "docs",
				Usage:  "Generate CLI documentation in Markdown format",
				Hidden: true,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "output",
						Aliases: []string{"o"},
						Value:   "CLI.md",
						Usage:   "Output file path",
					},
				},
				Action: func(c *cli.Context) error {
					md, err := c.App.ToMarkdown()
					if err != nil {
						return err
					}
					outputFile := c.String("output")
					if err := os.WriteFile(outputFile, []byte(md), 0644); err != nil {
						return err
					}
					fmt.Printf("文档已生成: %s\n", outputFile)
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
						Name:  "project-id",
						Usage: "Infisical project ID",
					},
					&cli.StringFlag{
						Name:  "path",
						Usage: "Path/folder in the provider (e.g., /yewseal)",
					},
					&cli.StringFlag{
						Name:    "env",
						Aliases: []string{"environment"},
						Usage:   "Environment name in the provider",
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
						yewsapp.ResolveSyncProvider(c, cfg),
						yewsapp.ResolveSyncKeyFile(c, cfg),
						yewsapp.ResolveSyncSecretName(c, cfg),
						yewsapp.ResolveSyncProjectID(c, cfg),
						yewsapp.ResolveSyncPath(c, cfg),
						yewsapp.ResolveSyncEnvironment(c, cfg),
					)
				},
				Subcommands: []*cli.Command{
					{
						Name:  "pull",
						Usage: "Pull key from secret management service to local file",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:    "key-file",
								Aliases: []string{"k"},
								Value:   ".age/keys.txt",
								Usage:   "Local path to save the key file",
							},
							&cli.StringFlag{
								Name:    "name",
								Aliases: []string{"n"},
								Value:   "AGE_KEY_FILE",
								Usage:   "Secret name in the provider",
							},
							&cli.StringFlag{
								Name:  "project-id",
								Usage: "Infisical project ID",
							},
							&cli.StringFlag{
								Name:  "path",
								Usage: "Path/folder in the provider (e.g., /yewseal)",
							},
							&cli.StringFlag{
								Name:    "env",
								Aliases: []string{"environment"},
								Usage:   "Environment name in the provider",
							},
							&cli.StringFlag{
								Name:    "provider",
								Aliases: []string{"p"},
								Value:   "infisical",
								Usage:   "Secret management provider (infisical)",
							},
						},
						Action: func(c *cli.Context) error {
							return sync.PullKey(
								yewsapp.ResolveSyncProvider(c, cfg),
								yewsapp.ResolveSyncKeyFile(c, cfg),
								yewsapp.ResolveSyncSecretName(c, cfg),
								yewsapp.ResolveSyncProjectID(c, cfg),
								yewsapp.ResolveSyncPath(c, cfg),
								yewsapp.ResolveSyncEnvironment(c, cfg),
							)
						},
					},
				},
			},
		},
		Before: func(c *cli.Context) error {
			// Skip tool check for commands that don't need them
			cmd := c.Args().First()
			if cmd == "check" || cmd == "doctor" || cmd == "sync" || cmd == "docs" {
				return nil
			}
			return tools.CheckTools()
		},
	}

	if err := app.Run(os.Args); err != nil {
		var exitCoder cli.ExitCoder
		if errors.As(err, &exitCoder) {
			if strings.TrimSpace(err.Error()) != "" {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			}
			os.Exit(exitCoder.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
