package cli

import (
	"github.com/YewFence/YewSeal/internal/config"

	"github.com/spf13/cobra"
)

// NewRootCommand 构建 yews 的根命令。version 由二进制在构建期注入。
func NewRootCommand(version string) *cobra.Command {
	return newRootCommand(version, config.LoadConfig)
}

func newRootCommand(version string, load configLoader) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "yews",
		Short: "YewSeal - Encrypt/decrypt configuration files using SOPS and Age (supports TOML, YAML, JSON, ENV, INI, and binary files)",
		Long: `YewSeal manages encrypted configuration files with SOPS and Age,
natively supporting TOML, YAML, JSON, ENV, INI, and binary formats.

File mappings, formats, and recipient authorization are declared once in
.yewseal.toml ("yews init" scaffolds it); command arguments only select
among the registered files.

CLI option precedence: flags > command environment variables > defaults.
Project mappings and authorization come only from .yewseal.toml.

Identity resolution order: --key-file (env YEWSEAL_KEY_FILE), then
YEWSEAL_AGE_IDENTITIES, SOPS_AGE_KEY, SOPS_AGE_KEY_FILE, SOPS_AGE_KEY_CMD,
then .age/keys.txt in the current working directory.

Documentation: ` + docsBaseURL + `/`,
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	rootCmd.PersistentFlags().StringP("key-file", "k", "", "Path to the Age private key file (fallback: YEWSEAL_AGE_IDENTITIES, SOPS_AGE_KEY, SOPS_AGE_KEY_FILE, SOPS_AGE_KEY_CMD, then .age/keys.txt)")
	annotateFlagEnvironment(rootCmd.PersistentFlags().Lookup("key-file"), "YEWSEAL_KEY_FILE")
	rootCmd.DisableAutoGenTag = true

	rootCmd.AddCommand(
		initCommand(),
		encryptCommand(load),
		decryptCommand(load),
		planCommand(load),
		editCommand(load),
		viewCommand(load),
		diffCommand(load),
	)

	return rootCmd
}
