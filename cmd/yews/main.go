package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/crypto"
	tools "github.com/YewFence/YewSeal/internal/doctor"
	"github.com/YewFence/YewSeal/internal/project"
	"github.com/YewFence/YewSeal/internal/sync"

	"github.com/urfave/cli/v2"
)

var Version = "dev"

func validateCLIFormatOverride(format string) (string, error) {
	if strings.TrimSpace(format) == "" {
		return "", nil
	}

	parsed := crypto.ParseFormat(format)
	if parsed == crypto.FormatUnknown {
		return "", fmt.Errorf("unsupported format %q (supported: toml, yaml, json, env, ini)", format)
	}

	return string(parsed), nil
}

func resolveFormatOverride(cliFormat string, filePair config.FilePair) (string, error) {
	validatedFormat, err := validateCLIFormatOverride(cliFormat)
	if err != nil {
		return "", err
	}
	if validatedFormat != "" {
		return validatedFormat, nil
	}
	return filePair.Format, nil
}

func resolveTargetFilePairs(cfg *config.Config, target string) ([]config.FilePair, error) {
	target = strings.TrimSpace(target)
	files := cfg.GetFiles()
	if target == "" {
		return files, nil
	}

	for _, filePair := range files {
		if target == filePair.PlaintextPath || target == filePair.EncryptedPath {
			return []config.FilePair{filePair}, nil
		}
	}

	return nil, fmt.Errorf("target %s is not configured as a plaintext or encrypted file", target)
}

func resolveSingleTargetFilePair(cfg *config.Config, target, commandName string) (config.FilePair, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return config.FilePair{}, fmt.Errorf("%s requires exactly one target", commandName)
	}

	filePairs, err := resolveTargetFilePairs(cfg, target)
	if err != nil {
		return config.FilePair{}, err
	}
	if len(filePairs) != 1 {
		return config.FilePair{}, fmt.Errorf("%s requires exactly one target", commandName)
	}
	return filePairs[0], nil
}

func writeViewedTarget(w io.Writer, cfg *config.Config, target, keyFile, cliFormat string, verbose bool) error {
	filePair, err := resolveSingleTargetFilePair(cfg, target, "view")
	if err != nil {
		return err
	}

	formatOverride, err := resolveFormatOverride(cliFormat, filePair)
	if err != nil {
		return err
	}

	plainData, err := crypto.DecryptToBytes(
		filePair.EncryptedPath,
		filePair.PlaintextPath,
		keyFile,
		formatOverride,
		verbose,
	)
	if err != nil {
		return err
	}

	if _, err := w.Write(plainData); err != nil {
		return err
	}
	return nil
}

func resolveSyncKeyFile(c *cli.Context, cfg *config.Config) string {
	if c.IsSet("key-file") {
		return cfg.GetKeyFile(c.String("key-file"))
	}
	return cfg.GetKeyFile("")
}

func resolveSyncProvider(c *cli.Context, cfg *config.Config) string {
	if c.IsSet("provider") {
		return cfg.GetSyncProvider(c.String("provider"))
	}
	return cfg.GetSyncProvider("")
}

func resolveSyncSecretName(c *cli.Context, cfg *config.Config) string {
	if c.IsSet("name") {
		return cfg.GetSyncSecretName(c.String("name"))
	}
	return cfg.GetSyncSecretName("")
}

func resolveSyncProjectID(c *cli.Context, cfg *config.Config) string {
	if c.IsSet("project-id") {
		return cfg.GetSyncProjectID(c.String("project-id"))
	}
	return cfg.GetSyncProjectID("")
}

func resolveSyncPath(c *cli.Context, cfg *config.Config) string {
	if c.IsSet("path") {
		return cfg.GetSyncPath(c.String("path"))
	}
	return cfg.GetSyncPath("")
}

func resolveSyncEnvironment(c *cli.Context, cfg *config.Config) string {
	if c.IsSet("env") {
		return cfg.GetSyncEnvironment(c.String("env"))
	}
	return cfg.GetSyncEnvironment("")
}

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
						Usage: "Format override for the first config entry (toml/yaml/json/env/ini)",
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
						Usage: "Format override for single-file mode (toml/yaml/json/env/ini)",
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
					cliFormat, err := validateCLIFormatOverride(c.String("format"))
					if err != nil {
						return err
					}

					// 目录扫描批量模式
					if c.String("dir") != "" {
						if cliFormat != "" {
							return fmt.Errorf("--format is only supported in single-file mode")
						}
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
							filePair.PlaintextPath = c.String("input")
						}
						if c.IsSet("output") {
							filePair.EncryptedPath = c.String("output")
						}
						formatOverride, err := resolveFormatOverride(cliFormat, filePair)
						if err != nil {
							return err
						}
						return crypto.Encrypt(filePair.PlaintextPath, filePair.EncryptedPath, keyFile, publicKey, formatOverride, verbose)
					}

					if cliFormat != "" {
						return fmt.Errorf("--format is only supported in single-file mode")
					}

					filePairs := cfg.GetFiles()
					if err := project.UpdateGitignore(filePairs); err != nil {
						return err
					}

					resolvedPublicKey, err := crypto.GetPublicKey(publicKey, keyFile, verbose)
					if err != nil {
						return err
					}
					if err := project.SyncSopsYaml(filePairs, resolvedPublicKey); err != nil {
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
						Usage: "Format override for single-file mode (toml/yaml/json/env/ini)",
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
					keyFile := cfg.GetKeyFile(c.String("key-file"))
					verbose := c.Bool("verbose")
					hasSingleFileOverride := c.IsSet("input") || c.IsSet("output")
					cliFormat, err := validateCLIFormatOverride(c.String("format"))
					if err != nil {
						return err
					}

					// 目录扫描批量模式
					if c.String("dir") != "" {
						if cliFormat != "" {
							return fmt.Errorf("--format is only supported in single-file mode")
						}
						opts := crypto.BatchOptions{
							InputDir:     c.String("dir"),
							Pattern:      c.String("pattern"),
							OutputDir:    c.String("output-dir"),
							OutputSuffix: c.String("output-suffix"),
							KeyFile:      keyFile,
							Parallel:     c.Int("parallel"),
							Verbose:      verbose,
							Force:        c.Bool("force"),
						}
						_, err := crypto.BatchDecrypt(opts)
						return err
					}

					if hasSingleFileOverride {
						filePair := cfg.GetPrimaryFilePair()
						if c.IsSet("input") {
							filePair.EncryptedPath = c.String("input")
						}
						if c.IsSet("output") {
							filePair.PlaintextPath = c.String("output")
						}
						formatOverride, err := resolveFormatOverride(cliFormat, filePair)
						if err != nil {
							return err
						}
						return crypto.DecryptWithOptions(
							filePair.EncryptedPath,
							filePair.PlaintextPath,
							keyFile,
							formatOverride,
							verbose,
							crypto.DecryptOptions{Force: c.Bool("force")},
						)
					}

					if cliFormat != "" {
						return fmt.Errorf("--format is only supported in single-file mode")
					}

					filePairs := cfg.GetFiles()
					if err := project.UpdateGitignore(filePairs); err != nil {
						return err
					}

					opts := crypto.BatchOptions{
						FilePairs: filePairs,
						KeyFile:   keyFile,
						Parallel:  c.Int("parallel"),
						Verbose:   verbose,
						Force:     c.Bool("force"),
					}
					_, err = crypto.BatchDecrypt(opts)
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
				Name:      "view",
				Usage:     "Print decrypted plaintext to standard output without writing files",
				UsageText: `yews view [command options] <target>`,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "format",
						Usage: "Format override for the selected target (toml/yaml/json/env/ini)",
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
					cliFormat, err := validateCLIFormatOverride(c.String("format"))
					if err != nil {
						return cli.Exit(err, 2)
					}

					if err := writeViewedTarget(os.Stdout, cfg, c.Args().First(), keyFile, cliFormat, verbose); err != nil {
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
						Usage: "Format override for the selected target (toml/yaml/json/env/ini)",
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
					cliFormat, err := validateCLIFormatOverride(c.String("format"))
					if err != nil {
						return cli.Exit(err, 2)
					}

					filePairs, err := resolveTargetFilePairs(cfg, c.Args().First())
					if err != nil {
						return cli.Exit(err, 2)
					}

					different := false
					for _, filePair := range filePairs {
						formatOverride, err := resolveFormatOverride(cliFormat, filePair)
						if err != nil {
							return cli.Exit(err, 2)
						}

						result, err := crypto.DiffPlaintextAgainstEncrypted(
							filePair.PlaintextPath,
							filePair.EncryptedPath,
							keyFile,
							formatOverride,
							verbose,
						)
						if err != nil {
							return cli.Exit(err, 2)
						}
						if result.Different {
							different = true
							fmt.Print(result.Diff)
						}
					}

					if different {
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
						resolveSyncProvider(c, cfg),
						resolveSyncKeyFile(c, cfg),
						resolveSyncSecretName(c, cfg),
						resolveSyncProjectID(c, cfg),
						resolveSyncPath(c, cfg),
						resolveSyncEnvironment(c, cfg),
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
								resolveSyncProvider(c, cfg),
								resolveSyncKeyFile(c, cfg),
								resolveSyncSecretName(c, cfg),
								resolveSyncProjectID(c, cfg),
								resolveSyncPath(c, cfg),
								resolveSyncEnvironment(c, cfg),
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
