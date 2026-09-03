package cli

import (
	"github.com/YewFence/YewSeal/internal/config"

	"github.com/spf13/cobra"
)

// NewRootCommand 构建 yews 的根命令。version 由二进制在构建期注入。
func NewRootCommand(cfg *config.Config, version string) *cobra.Command {
	var keyFile string

	rootCmd := &cobra.Command{
		Use:           "yews",
		Short:         "YewSeal - Encrypt/decrypt configuration files using SOPS and Age (supports TOML, YAML, JSON, ENV, INI)",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	rootCmd.PersistentFlags().StringVarP(&keyFile, "key-file", "k", envValue("AGE_KEY_FILE"), "Path to Age private key file")
	rootCmd.DisableAutoGenTag = true

	rootCmd.AddCommand(
		initCommand(),
		encryptCommand(cfg),
		decryptCommand(cfg, &keyFile),
		planCommand(cfg),
		editCommand(cfg, &keyFile),
		viewCommand(cfg, &keyFile),
		diffCommand(cfg, &keyFile),
		syncCommand(cfg),
	)

	return rootCmd
}
