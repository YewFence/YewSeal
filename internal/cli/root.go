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
	var keyFile string

	rootCmd := &cobra.Command{
		Use:           "yews",
		Short:         "YewSeal - Encrypt/decrypt configuration files using SOPS and Age (supports TOML, YAML, JSON, ENV, INI, and binary files)",
		Long: `YewSeal manages encrypted configuration files with SOPS and Age,
natively supporting TOML, YAML, JSON, ENV, INI, and binary formats.

File mappings, formats, and recipient authorization are declared once in
.yewseal.toml ("yews init" scaffolds it); command arguments only select
among the registered files.

Configuration precedence: CLI flags > environment variables > config file
> defaults.

Identity resolution order: --key-file (env AGE_KEY_FILE), then
YEWSEAL_AGE_IDENTITIES, SOPS_AGE_KEY, SOPS_AGE_KEY_FILE, SOPS_AGE_KEY_CMD,
then .age/keys.txt in the current working directory.

Documentation: ` + docsBaseURL + `/`,
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	rootCmd.PersistentFlags().StringVarP(&keyFile, "key-file", "k", envValue("AGE_KEY_FILE"),
		"Path to the Age private key file (env AGE_KEY_FILE; fallback: YEWSEAL_AGE_IDENTITIES, SOPS_AGE_KEY*, then .age/keys.txt)")
	rootCmd.DisableAutoGenTag = true

	rootCmd.AddCommand(
		initCommand(),
		encryptCommand(load),
		decryptCommand(load, &keyFile),
		planCommand(load),
		editCommand(load, &keyFile),
		viewCommand(load, &keyFile),
		diffCommand(load, &keyFile),
	)

	return rootCmd
}
