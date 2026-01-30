package main

import (
	"fmt"
	"os"

	"github.com/yourusername/sops-config-tool/internal/crypto"
	"github.com/yourusername/sops-config-tool/internal/tools"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:    "sops-config-tool",
		Usage:   "Encrypt/decrypt configuration files using SOPS, Age, and yq",
		Version: "1.0.0",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "key-file",
				Aliases: []string{"k"},
				Usage:   "Path to Age private key file",
				EnvVars: []string{"AGE_KEY_FILE"},
			},
			&cli.BoolFlag{
				Name:  "verbose",
				Usage: "Enable verbose output",
			},
		},
		Commands: []*cli.Command{
			{
				Name:  "init",
				Usage: "Initialize project with Age keys and SOPS configuration",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "force",
						Usage: "Force overwrite existing configuration",
					},
				},
				Action: func(c *cli.Context) error {
					return crypto.InitProject(c.Bool("force"))
				},
			},
			{
				Name:  "encrypt",
				Usage: "Encrypt wrangler.toml to wrangler.enc.yaml",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "input",
						Value: "wrangler.toml",
						Usage: "Input TOML file",
					},
					&cli.StringFlag{
						Name:  "output",
						Value: "wrangler.enc.yaml",
						Usage: "Output encrypted YAML file",
					},
				},
				Action: func(c *cli.Context) error {
					keyFile := c.String("key-file")
					verbose := c.Bool("verbose")
					return crypto.Encrypt(c.String("input"), c.String("output"), keyFile, verbose)
				},
			},
			{
				Name:  "decrypt",
				Usage: "Decrypt wrangler.enc.yaml to wrangler.toml",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "input",
						Value: "wrangler.enc.yaml",
						Usage: "Input encrypted YAML file",
					},
					&cli.StringFlag{
						Name:  "output",
						Value: "wrangler.toml",
						Usage: "Output TOML file",
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
						Name:  "file",
						Value: "wrangler.enc.yaml",
						Usage: "Encrypted file to edit",
					},
					&cli.StringFlag{
						Name:  "editor",
						Usage: "Editor command to use (e.g., 'code -w', 'vim')",
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
