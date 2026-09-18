package cli

import (
	"errors"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/errx"

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

Exit codes: 0 on success, 1 on business failure (file processing or
output delivery), 2 on calling errors (arguments, config, selection, or
identity source). See each command's --help for specifics.

Documentation: ` + docsBaseURL + `/`,
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	rootCmd.PersistentFlags().StringP("key-file", "k", "", "Path to the Age private key file (fallback: YEWSEAL_AGE_IDENTITIES, SOPS_AGE_KEY, SOPS_AGE_KEY_FILE, SOPS_AGE_KEY_CMD, then .age/keys.txt)")
	annotateFlagEnvironment(rootCmd.PersistentFlags().Lookup("key-file"), "YEWSEAL_KEY_FILE")
	rootCmd.DisableAutoGenTag = true
	rootCmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return errx.Usage(err) })

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

// ExitCode 把命令错误映射为进程退出码：UsageError 或未进入 RunE 的
// Cobra 错误（未知命令等）为 2，其余业务与输出错误为 1。
func ExitCode(cmd *cobra.Command, err error) int {
	var usage *errx.UsageError
	if errors.As(err, &usage) {
		return usage.ExitCode()
	}
	if cmd != nil && !cmd.Runnable() {
		return 2
	}
	return 1
}
