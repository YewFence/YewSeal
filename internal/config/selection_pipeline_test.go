package config

import (
	"os"
	"path/filepath"
	"testing"

	"filippo.io/age"
	"github.com/YewFence/YewSeal/internal/task"
	"github.com/stretchr/testify/require"
)

func selectionConfig(t *testing.T) *Config {
	t.Helper()
	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	defaults := []string{"owner"}
	return &Config{
		CurrentDir: t.TempDir(),
		Recipients: RecipientConfig{Defaults: &defaults, Registry: map[string]string{"owner": identity.Recipient().String()}},
	}
}

func selectionFile(t *testing.T, root, name string) string {
	t.Helper()
	path := filepath.Join(root, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
	require.NoError(t, os.WriteFile(path, []byte("unused"), 0600))
	return path
}

func TestSelectionDirectoryOnlyFiltersRegisteredMappings(t *testing.T) {
	cfg := selectionConfig(t)
	registered := selectionFile(t, cfg.CurrentDir, "config/app.yaml")
	selectionFile(t, cfg.CurrentDir, "extra/config/unregistered.yaml")
	cfg.Encryption.Groups = []GroupConfig{{ConfigDir: cfg.CurrentDir, Patterns: []string{"/config/*.yaml"}}}
	result, err := ResolveSelection(cfg, SelectionOptions{Command: task.ModeEncrypt, Target: filepath.Join(cfg.CurrentDir, "config")})
	require.NoError(t, err)
	require.Len(t, result.FilePairs, 1)
	require.Equal(t, registered, result.FilePairs[0].PlaintextPath)
	require.Equal(t, PairSourceScan, result.FilePairs[0].Source)
	require.Equal(t, SelectedByDirectoryTarget, result.FilePairs[0].SelectedBy)
	filtered, err := ResolveSelection(cfg, SelectionOptions{Command: task.ModeEncrypt, Target: filepath.Join(cfg.CurrentDir, "config"), Patterns: []string{"/config/*.yaml"}})
	require.NoError(t, err)
	require.Len(t, filtered.FilePairs, 1)
	require.Equal(t, registered, filtered.FilePairs[0].PlaintextPath)
	_, err = ResolveSelection(cfg, SelectionOptions{Command: task.ModeEncrypt, Target: filepath.Join(cfg.CurrentDir, "extra")})
	require.Error(t, err)
	_, err = ResolveSelection(cfg, SelectionOptions{Command: task.ModeEncrypt, Target: filepath.Join(cfg.CurrentDir, "extra"), Patterns: []string{"*.yaml"}})
	require.Error(t, err, "a CLI pattern cannot register additional files")
}

func TestSelectionExplicitDirectoryUsesCommandSide(t *testing.T) {
	cfg := selectionConfig(t)
	plain := selectionFile(t, cfg.CurrentDir, "app/dev.yaml")
	enc := selectionFile(t, cfg.CurrentDir, "secrets/dev.enc.yaml")
	cfg.Encryption.Files = []FilePair{{PlaintextPath: plain, EncryptedPath: enc, ConfigPath: filepath.Join(cfg.CurrentDir, ".yewseal.toml")}}
	for _, tc := range []struct {
		command string
		dir     string
		want    bool
	}{
		{task.ModeEncrypt, "app", true}, {task.ModeEncrypt, "secrets", false},
		{task.ModeDecrypt, "app", false}, {task.ModeDecrypt, "secrets", true},
		{"plan", "app", true}, {"plan", "secrets", true},
	} {
		t.Run(tc.command+"/"+tc.dir, func(t *testing.T) {
			result, err := ResolveSelection(cfg, SelectionOptions{Command: tc.command, Target: filepath.Join(cfg.CurrentDir, tc.dir)})
			if !tc.want {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, result.FilePairs, 1)
			require.Equal(t, ValueSourceExact, result.FilePairs[0].PlaintextSource.Kind)
		})
	}
}

func TestPlanDiscoversBothSidesWithoutInspectingContent(t *testing.T) {
	cfg := selectionConfig(t)
	for _, name := range []string{"new.yaml", "remote.enc.yaml", "existing.yml", "existing.enc.yaml"} {
		selectionFile(t, cfg.CurrentDir, name)
	}
	cfg.Encryption.Groups = []GroupConfig{{ConfigDir: cfg.CurrentDir}}
	for _, target := range []string{"", cfg.CurrentDir, filepath.Join(cfg.CurrentDir, "remote.enc.yaml")} {
		result, err := ResolveSelection(cfg, SelectionOptions{Command: task.ModePlan, Target: target})
		require.NoError(t, err)
		require.Equal(t, "plan", result.Command)
		if target == filepath.Join(cfg.CurrentDir, "remote.enc.yaml") {
			require.Len(t, result.FilePairs, 1)
			require.Equal(t, filepath.Join(cfg.CurrentDir, "remote.yaml"), result.FilePairs[0].PlaintextPath)
			continue
		}
		require.Len(t, result.FilePairs, 3)
		paths := make([]string, 0, len(result.FilePairs))
		for _, pair := range result.FilePairs {
			paths = append(paths, filepath.Base(pair.PlaintextPath))
		}
		require.ElementsMatch(t, []string{"new.yaml", "remote.yaml", "existing.yml"}, paths)
	}
}

func TestPlanRejectsCompetingMappingsUnlessExplicitlyResolved(t *testing.T) {
	cfg := selectionConfig(t)
	for _, name := range []string{"config.yaml", "config.yml", "config.enc.yaml"} {
		selectionFile(t, cfg.CurrentDir, name)
	}
	cfg.Encryption.Groups = []GroupConfig{{ConfigDir: cfg.CurrentDir}}
	_, err := ResolveSelection(cfg, SelectionOptions{Command: task.ModePlan})
	require.ErrorContains(t, err, "conflicting group file pairs")
	cfg.Encryption.Files = []FilePair{{PlaintextPath: filepath.Join(cfg.CurrentDir, "custom.yaml"), EncryptedPath: filepath.Join(cfg.CurrentDir, "config.enc.yaml"), Format: "yaml"}}
	result, err := ResolveSelection(cfg, SelectionOptions{Command: task.ModePlan})
	require.NoError(t, err)
	require.Len(t, result.FilePairs, 1)
	require.Equal(t, cfg.Encryption.Files[0].PlaintextPath, result.FilePairs[0].PlaintextPath)
}

func TestSelectionChecksUnselectedAuthorizationBeforeFiltering(t *testing.T) {
	cfg := selectionConfig(t)
	missing := []string{"missing"}
	good := filepath.Join(cfg.CurrentDir, "good.yaml")
	cfg.Encryption.Files = []FilePair{
		{PlaintextPath: good, EncryptedPath: filepath.Join(cfg.CurrentDir, "good.enc.yaml")},
		{PlaintextPath: filepath.Join(cfg.CurrentDir, "bad.yaml"), EncryptedPath: filepath.Join(cfg.CurrentDir, "bad.enc.yaml"), Recipients: &missing},
	}
	for _, command := range []string{task.ModeEncrypt, "plan"} {
		_, err := ResolveSelection(cfg, SelectionOptions{Command: command, Target: good})
		require.ErrorContains(t, err, "unknown recipient alias")
	}
	result, err := ResolveSelection(cfg, SelectionOptions{Command: task.ModeDecrypt})
	require.NoError(t, err)
	require.Len(t, result.FilePairs, 2)
	require.Contains(t, result.FilePairs[1].RecipientWarning, "unknown recipient alias")
}

func TestPlanGroupFormatRulesAndAuthorizationConflicts(t *testing.T) {
	cfg := selectionConfig(t)
	selectionFile(t, cfg.CurrentDir, "config/live.conf")
	selectionFile(t, cfg.CurrentDir, "config/remote.conf.enc.env")
	group := GroupConfig{ConfigDir: cfg.CurrentDir, ConfigPath: filepath.Join(cfg.CurrentDir, ".yewseal.toml"), Patterns: []string{"config/*.conf"}, FormatRules: []string{"config/*.conf=env"}}
	cfg.Encryption.Groups = []GroupConfig{group}
	selection, err := ResolveSelection(cfg, SelectionOptions{Command: task.ModePlan})
	require.NoError(t, err)
	require.Len(t, selection.FilePairs, 2)
	for _, pair := range selection.FilePairs {
		require.Equal(t, "env", pair.Format)
		require.Equal(t, "defaults", pair.RecipientInfo.Kind)
		require.Equal(t, group.ConfigPath, pair.ConfigPath)
	}
	other, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	cfg.Recipients.Registry["other"] = other.Recipient().String()
	aliases := []string{"other"}
	group.Recipients = &aliases
	cfg.Encryption.Groups = append(cfg.Encryption.Groups, group)
	_, err = ResolveSelection(cfg, SelectionOptions{Command: task.ModePlan})
	require.ErrorContains(t, err, "conflicting recipient sets")
}

func TestSelectionOutputOverridePreservesRegisteredSnapshot(t *testing.T) {
	cfg := selectionConfig(t)
	plain := filepath.Join(cfg.CurrentDir, "config.yaml")
	enc := filepath.Join(cfg.CurrentDir, "config.enc.yaml")
	cfg.Encryption.Files = []FilePair{{PlaintextPath: plain, EncryptedPath: enc}}
	for _, command := range []string{task.ModeEncrypt, task.ModeDecrypt} {
		for _, target := range []string{plain, enc} {
			result, err := ResolveSelection(cfg, SelectionOptions{Command: command, Target: target, OutputSet: true, Output: "export.json"})
			require.NoError(t, err)
			require.Len(t, result.FilePairs, 1)
			require.Len(t, result.AllConfigPairs, 1)
			require.Equal(t, plain, result.AllConfigPairs[0].PlaintextPath)
			require.Equal(t, enc, result.AllConfigPairs[0].EncryptedPath)
			require.Equal(t, "yaml", result.FilePairs[0].Format)
			require.Equal(t, ValueSourceFilename, result.FilePairs[0].FormatSource.Kind)
			if command == task.ModeEncrypt {
				require.Equal(t, filepath.Join(cfg.CurrentDir, "export.json"), result.FilePairs[0].EncryptedPath)
				require.Equal(t, ValueSourceArgument, result.FilePairs[0].EncryptedSource.Kind)
			} else {
				require.Equal(t, filepath.Join(cfg.CurrentDir, "export.json"), result.FilePairs[0].PlaintextPath)
				require.Equal(t, ValueSourceArgument, result.FilePairs[0].PlaintextSource.Kind)
			}
		}
	}
	_, err := ResolveSelection(cfg, SelectionOptions{Command: task.ModePlan, Target: plain, OutputSet: true, Output: "export.json"})
	require.ErrorContains(t, err, "plan does not support output overrides")
}
