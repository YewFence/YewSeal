package main

import (
	"fmt"
	"os"

	"github.com/yourusername/YewSeal/internal/crypto"
	"github.com/yourusername/YewSeal/internal/tools"

	"github.com/urfave/cli/v2"
)

func main() {
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
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"v"},
				Usage:   "Enable verbose output",
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
				},
				Action: func(c *cli.Context) error {
					return crypto.InitProject(c.Bool("force"))
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
						Value:   "wrangler.enc.yaml",
						Usage:   "Output encrypted YAML file",
						EnvVars: []string{"SOPS_OUTPUT_FILE"},
					},
				},
				Action: func(c *cli.Context) error {
					keyFile := c.String("key-file")
					verbose := c.Bool("verbose")
					return crypto.Encrypt(c.String("input"), c.String("output"), keyFile, verbose)
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
						Value:   "wrangler.enc.yaml",
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
				},
				Action: func(c *cli.Context) error {
					keyFile := c.String("key-file")
					verbose := c.Bool("verbose")
					return crypto.Decrypt(c.String("input"), c.String("output"), keyFile, verbose)
				},
			},
			{
				Name:  "edit",
				Usage: "Edit encrypted configuration file using SOPS",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "file",
						Aliases: []string{"f"},
						Value:   "wrangler.enc.yaml",
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
		},
		Before: func(c *cli.Context) error {
			// Check if required tools are installed
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
