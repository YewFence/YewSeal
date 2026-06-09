package main

import (
	"github.com/YewFence/YewSeal/internal/config"
	tools "github.com/YewFence/YewSeal/internal/doctor"

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
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if shouldSkipToolCheck(cmd) {
				return nil
			}
			return tools.CheckTools()
		},
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
		checkCommand(),
		docsCommand(rootCmd),
		syncCommand(cfg),
	)

	return rootCmd
}

func shouldSkipToolCheck(cmd *cobra.Command) bool {
	for current := cmd; current != nil; current = current.Parent() {
		switch current.Name() {
		case "check", "doctor", "sync", "docs", "plan", "completion", cobra.ShellCompRequestCmd, cobra.ShellCompNoDescRequestCmd:
			return true
		}
	}
	return false
}
