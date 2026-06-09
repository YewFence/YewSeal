package main

import (
	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/sync"

	"github.com/spf13/cobra"
)

func syncCommand(cfg *config.Config) *cobra.Command {
	opts := syncOptions{
		KeyFile:  ".age/keys.txt",
		Name:     "AGE_KEY_FILE",
		Provider: "infisical",
	}

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync sensitive files to secret management service",
		RunE: func(cmd *cobra.Command, args []string) error {
			return sync.SyncKey(
				resolveSyncProvider(cmd, cfg, opts),
				resolveSyncKeyFile(cmd, cfg, opts),
				resolveSyncSecretName(cmd, cfg, opts),
				resolveSyncProjectID(cmd, cfg, opts),
				resolveSyncPath(cmd, cfg, opts),
				resolveSyncEnvironment(cmd, cfg, opts),
			)
		},
	}
	addSyncFlags(cmd.Flags(), &opts)
	cmd.AddCommand(syncPullCommand(cfg))
	return cmd
}

func syncPullCommand(cfg *config.Config) *cobra.Command {
	opts := syncOptions{
		KeyFile:  ".age/keys.txt",
		Name:     "AGE_KEY_FILE",
		Provider: "infisical",
	}

	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Pull key from secret management service to local file",
		RunE: func(cmd *cobra.Command, args []string) error {
			return sync.PullKey(
				resolveSyncProvider(cmd, cfg, opts),
				resolveSyncKeyFile(cmd, cfg, opts),
				resolveSyncSecretName(cmd, cfg, opts),
				resolveSyncProjectID(cmd, cfg, opts),
				resolveSyncPath(cmd, cfg, opts),
				resolveSyncEnvironment(cmd, cfg, opts),
			)
		},
	}
	addSyncPullFlags(cmd.Flags(), &opts)
	return cmd
}

func resolveSyncKeyFile(cmd *cobra.Command, cfg *config.Config, opts syncOptions) string {
	if cmd.Flags().Changed("key-file") {
		return cfg.GetKeyFile(opts.KeyFile)
	}
	return cfg.GetKeyFile("")
}

func resolveSyncProvider(cmd *cobra.Command, cfg *config.Config, opts syncOptions) string {
	if cmd.Flags().Changed("provider") {
		return cfg.GetSyncProvider(opts.Provider)
	}
	return cfg.GetSyncProvider("")
}

func resolveSyncSecretName(cmd *cobra.Command, cfg *config.Config, opts syncOptions) string {
	if cmd.Flags().Changed("name") {
		return cfg.GetSyncSecretName(opts.Name)
	}
	return cfg.GetSyncSecretName("")
}

func resolveSyncProjectID(cmd *cobra.Command, cfg *config.Config, opts syncOptions) string {
	if cmd.Flags().Changed("project-id") {
		return cfg.GetSyncProjectID(opts.ProjectID)
	}
	return cfg.GetSyncProjectID("")
}

func resolveSyncPath(cmd *cobra.Command, cfg *config.Config, opts syncOptions) string {
	if cmd.Flags().Changed("path") {
		return cfg.GetSyncPath(opts.Path)
	}
	return cfg.GetSyncPath("")
}

func resolveSyncEnvironment(cmd *cobra.Command, cfg *config.Config, opts syncOptions) string {
	if cmd.Flags().Changed("env") {
		return cfg.GetSyncEnvironment(opts.Env)
	}
	return cfg.GetSyncEnvironment("")
}
