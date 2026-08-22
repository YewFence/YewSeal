package main

import (
	"github.com/YewFence/YewSeal/internal/config"

	"github.com/spf13/cobra"
)

func rootCommand(cfg *config.Config) *cobra.Command {
	var keyFile string

	rootCmd := &cobra.Command{
		Use:           "yews",
		Short:         "YewSeal - Encrypt/decrypt configuration files using SOPS and Age (supports TOML, YAML, JSON, ENV, INI)",
		Version:       Version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	rootCmd.PersistentFlags().StringVarP(&keyFile, "key-file", "k", envValue("AGE_KEY_FILE"), "Path to Age private key file")
	rootCmd.DisableAutoGenTag = true

	rootCmd.AddCommand(
		initCommand(),
		encryptCommand(cfg, &keyFile),
		decryptCommand(cfg, &keyFile),
		planCommand(cfg),
		editCommand(cfg, &keyFile),
		viewCommand(cfg, &keyFile),
		diffCommand(cfg, &keyFile),
		docsCommand(rootCmd),
		syncCommand(cfg),
	)

	return rootCmd
}
