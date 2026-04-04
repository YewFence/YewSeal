package main

import (
	"fmt"
	"os"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/crypto"
	"github.com/YewFence/YewSeal/internal/sync"
	"github.com/YewFence/YewSeal/internal/tools"
	"github.com/YewFence/YewSeal/internal/utils"

	"github.com/urfave/cli/v2"
)

var Version = "dev"

func main() {
	// Load configuration file
	cfg, err := config.LoadConfig()
	if err != nil {
		cfg = config.DefaultConfig()
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
				Usage:   "Encrypt configuration file (supports .toml, .yaml, .yml, .json, .env, .ini)",
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
					keyFile := cfg.GetKeyFile(c.String("key-file"))
					publicKey := c.String("public-key")
					verbose := c.Bool("verbose")
					hasSingleFileOverride := c.IsSet("input") || c.IsSet("output")

					// 目录扫描批量模式
					if c.String("dir") != "" {
						opts := crypto.BatchOptions{
							InputDir:     c.String("dir"),
							Pattern:      c.String("pattern"),
							OutputDir:    c.String("output-dir"),
							OutputSuffix: c.String("output-suffix"),
							KeyFile:      keyFile,
							PublicKey:    publicKey,
							Parallel:     c.Int("parallel"),
							Verbose:      verbose,
						}
						_, err := crypto.BatchEncrypt(opts)
						return err
					}

					if hasSingleFileOverride {
						filePair := cfg.GetPrimaryFilePair()
						if c.IsSet("input") {
							filePair.Dec = c.String("input")
						}
						if c.IsSet("output") {
							filePair.Enc = c.String("output")
						}
						return crypto.Encrypt(filePair.Dec, filePair.Enc, keyFile, publicKey, "", verbose)
					}

					filePairs := cfg.GetFiles()
					if err := utils.UpdateGitignore(filePairs); err != nil {
						return err
					}

					resolvedPublicKey, err := crypto.GetPublicKey(publicKey, keyFile, verbose)
					if err != nil {
						return err
					}
					if err := crypto.SyncSopsYaml(filePairs, resolvedPublicKey); err != nil {
						return err
					}

					opts := crypto.BatchOptions{
						FilePairs: filePairs,
						KeyFile:   keyFile,
						PublicKey: resolvedPublicKey,
						Parallel:  c.Int("parallel"),
						Verbose:   verbose,
					}
					_, err = crypto.BatchEncrypt(opts)
					return err
				},
			},
			{
				Name:    "decrypt",
				Aliases: []string{"d"},
				Usage:   "Decrypt encrypted file (output format determined by extension)",
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
						Name:    "verbose",
						Aliases: []string{"v"},
						Usage:   "Enable verbose output",
					},
				},
				Action: func(c *cli.Context) error {
					keyFile := cfg.GetKeyFile(c.String("key-file"))
					verbose := c.Bool("verbose")
					hasSingleFileOverride := c.IsSet("input") || c.IsSet("output")

					// 目录扫描批量模式
					if c.String("dir") != "" {
						opts := crypto.BatchOptions{
							InputDir:     c.String("dir"),
							Pattern:      c.String("pattern"),
							OutputDir:    c.String("output-dir"),
							OutputSuffix: c.String("output-suffix"),
							KeyFile:      keyFile,
							Parallel:     c.Int("parallel"),
							Verbose:      verbose,
						}
						_, err := crypto.BatchDecrypt(opts)
						return err
					}

					if hasSingleFileOverride {
						filePair := cfg.GetPrimaryFilePair()
						if c.IsSet("input") {
							filePair.Enc = c.String("input")
						}
						if c.IsSet("output") {
							filePair.Dec = c.String("output")
						}
						return crypto.Decrypt(filePair.Enc, filePair.Dec, keyFile, "", verbose)
					}

					filePairs := cfg.GetFiles()
					if err := utils.UpdateGitignore(filePairs); err != nil {
						return err
					}

					opts := crypto.BatchOptions{
						FilePairs: filePairs,
						KeyFile:   keyFile,
						Parallel:  c.Int("parallel"),
						Verbose:   verbose,
					}
					_, err := crypto.BatchDecrypt(opts)
					return err
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
							return sync.PullKey(
								c.String("provider"),
								c.String("key-file"),
								c.String("name"),
								c.String("path"),
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
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
