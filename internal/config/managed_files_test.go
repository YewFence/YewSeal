package config

import (
	"path/filepath"
	"testing"

	"github.com/YewFence/YewSeal/internal/task"
	"github.com/stretchr/testify/require"
)

func TestResolveSelectionPreparesConfigurationOwnership(t *testing.T) {
	cfg := selectionConfig(t)
	root := cfg.CurrentDir
	child := filepath.Join(root, "service")
	empty := filepath.Join(root, "empty")
	rootConfig := filepath.Join(root, ".yewseal.toml")
	childConfig := filepath.Join(child, ".config", ".yewseal.toml")
	cfg.LoadedFiles = []LoadedFile{
		{Path: childConfig, Dir: child},
		{Path: filepath.Join(empty, ".yewseal.toml"), Dir: empty},
		{Path: rootConfig, Dir: root},
	}
	cfg.Encryption.Files = []FilePair{
		{ConfigPath: rootConfig, PlaintextPath: filepath.Join(child, "parent.yaml"), EncryptedPath: filepath.Join(child, "parent.enc.yaml")},
		{ConfigPath: childConfig, PlaintextPath: filepath.Join(child, "secret.yaml"), EncryptedPath: filepath.Join(child, "secret.enc.yaml")},
	}
	selection, err := ResolveSelection(cfg, SelectionOptions{
		Command: task.ModeEncrypt, Targets: []string{cfg.Encryption.Files[1].PlaintextPath}, PrepareManagedFiles: true,
	})
	require.NoError(t, err)
	require.Len(t, selection.FilePairs, 1)
	require.Len(t, selection.ManagedScopes, 3)
	require.Equal(t, root, selection.ManagedScopes[0].Dir)
	require.Equal(t, cfg.Encryption.Files[0].PlaintextPath, selection.ManagedScopes[0].FilePairs[0].PlaintextPath)
	require.Equal(t, ManagedFileScope{Dir: empty}, selection.ManagedScopes[1])
	require.Equal(t, child, selection.ManagedScopes[2].Dir)
	require.Equal(t, cfg.Encryption.Files[1].PlaintextPath, selection.ManagedScopes[2].FilePairs[0].PlaintextPath)
}

func TestResolveSelectionManagedOwnershipFallsBackToMappingSource(t *testing.T) {
	cfg := selectionConfig(t)
	root := cfg.CurrentDir
	child := filepath.Join(root, "service")
	cfg.Encryption.Files = []FilePair{
		{PlaintextPath: filepath.Join(root, "root.yaml"), EncryptedPath: filepath.Join(root, "root.enc.yaml")},
		{ConfigPath: filepath.Join("service", ".yewseal.toml"), PlaintextPath: filepath.Join(child, "relative.yaml"), EncryptedPath: filepath.Join(child, "relative.enc.yaml")},
		{ConfigPath: filepath.Join(child, ".yewseal.toml"), PlaintextPath: filepath.Join(child, "absolute.yaml"), EncryptedPath: filepath.Join(child, "absolute.enc.yaml")},
	}
	selection, err := ResolveSelection(cfg, SelectionOptions{Command: task.ModePlan, PrepareManagedFiles: true})
	require.NoError(t, err)
	require.Len(t, selection.ManagedScopes, 2)
	require.Equal(t, root, selection.ManagedScopes[0].Dir)
	require.Len(t, selection.ManagedScopes[0].FilePairs, 1)
	require.Equal(t, child, selection.ManagedScopes[1].Dir)
	require.Len(t, selection.ManagedScopes[1].FilePairs, 2)
}

func TestResolveSelectionManagedFilesReuseDiscoveryAndPreserveCommandPaths(t *testing.T) {
	for _, command := range []string{task.ModeEncrypt, task.ModeDecrypt, task.ModeVerify, task.ModeClean} {
		t.Run(command, func(t *testing.T) {
			cfg := selectionConfig(t)
			plain := selectionFile(t, cfg.CurrentDir, "app.yml")
			enc := selectionFile(t, cfg.CurrentDir, "app.enc.yaml")
			cfg.Encryption.Groups = []GroupConfig{{ConfigDir: cfg.CurrentDir, Patterns: []string{"*.yaml", "*.yml"}}}
			selection, err := ResolveSelection(cfg, SelectionOptions{Command: command, PrepareManagedFiles: true})
			require.NoError(t, err)
			require.Equal(t, task.ScanStats{Entries: 3, Directories: 1}, selection.ScanStats)
			require.Len(t, selection.FilePairs, 1)
			wantPlaintext := plain
			if command == task.ModeDecrypt {
				wantPlaintext = filepath.Join(cfg.CurrentDir, "app.yaml")
			}
			require.Equal(t, wantPlaintext, selection.FilePairs[0].PlaintextPath)
			require.Equal(t, enc, selection.FilePairs[0].EncryptedPath)
			require.Len(t, selection.ManagedScopes, 1)
			require.Len(t, selection.ManagedScopes[0].FilePairs, 1)
			require.Equal(t, plain, selection.ManagedScopes[0].FilePairs[0].PlaintextPath)
			require.Equal(t, enc, selection.ManagedScopes[0].FilePairs[0].EncryptedPath)
		})
	}
}

func TestResolveSelectionManagedFilesAreDemandDriven(t *testing.T) {
	cfg := selectionConfig(t)
	enc := selectionFile(t, cfg.CurrentDir, "app.enc.yaml")
	selectionFile(t, cfg.CurrentDir, "broken.unknown")
	cfg.Encryption.Groups = []GroupConfig{{ConfigDir: cfg.CurrentDir, Patterns: []string{"*"}}}
	selection, err := ResolveSelection(cfg, SelectionOptions{Command: task.ModeDecrypt, Targets: []string{enc}})
	require.NoError(t, err)
	require.Len(t, selection.FilePairs, 1)
	require.Empty(t, selection.ManagedScopes)
	require.Equal(t, task.ScanStats{Entries: 3, Directories: 1}, selection.ScanStats)
	_, err = ResolveSelection(cfg, SelectionOptions{Command: task.ModeDecrypt, Targets: []string{enc}, PrepareManagedFiles: true})
	require.ErrorContains(t, err, "could not detect format")
}

func TestResolveSelectionManagedFilesRetainHistoricalAuthorization(t *testing.T) {
	cfg := selectionConfig(t)
	enc := selectionFile(t, cfg.CurrentDir, "app.enc.yaml")
	selectionFile(t, cfg.CurrentDir, "other.yaml")
	missing := []string{"removed"}
	cfg.Encryption.Groups = []GroupConfig{{ConfigDir: cfg.CurrentDir, Patterns: []string{"*.yaml"}, Recipients: &missing}}
	selection, err := ResolveSelection(cfg, SelectionOptions{Command: task.ModeDecrypt, Targets: []string{enc}, PrepareManagedFiles: true})
	require.NoError(t, err)
	require.Len(t, selection.FilePairs, 1)
	require.Contains(t, selection.FilePairs[0].RecipientWarning, "unknown recipient alias")
	require.Len(t, selection.ManagedScopes, 1)
	require.Len(t, selection.ManagedScopes[0].FilePairs, 2)
	for _, pair := range selection.ManagedScopes[0].FilePairs {
		require.Contains(t, pair.RecipientWarning, "unknown recipient alias")
	}
	_, err = ResolveSelection(cfg, SelectionOptions{Command: task.ModeEncrypt, PrepareManagedFiles: true})
	require.ErrorContains(t, err, "unknown recipient alias")
}

func TestResolveSelectionManagedFilesDiscoverAgainOnNextCall(t *testing.T) {
	cfg := selectionConfig(t)
	selectionFile(t, cfg.CurrentDir, "first.yaml")
	cfg.Encryption.Groups = []GroupConfig{{ConfigDir: cfg.CurrentDir, Patterns: []string{"*.yaml"}}}
	opts := SelectionOptions{Command: task.ModeEncrypt, PrepareManagedFiles: true}
	first, err := ResolveSelection(cfg, opts)
	require.NoError(t, err)
	require.Len(t, first.FilePairs, 1)
	selectionFile(t, cfg.CurrentDir, "second.yaml")
	second, err := ResolveSelection(cfg, opts)
	require.NoError(t, err)
	require.Len(t, second.FilePairs, 2)
	require.Len(t, second.ManagedScopes[0].FilePairs, 2)
	require.Equal(t, task.ScanStats{Entries: 3, Directories: 1}, second.ScanStats)
	require.Len(t, first.ManagedScopes[0].FilePairs, 1)
}

func TestResolveSelectionManagedFilesKeepRegisteredOutput(t *testing.T) {
	cfg := selectionConfig(t)
	plain := selectionFile(t, cfg.CurrentDir, "app.yaml")
	enc := filepath.Join(cfg.CurrentDir, "app.enc.yaml")
	cfg.Encryption.Files = []FilePair{{PlaintextPath: plain, EncryptedPath: enc}}
	selection, err := ResolveSelection(cfg, SelectionOptions{
		Command: task.ModeEncrypt, Targets: []string{plain}, Output: "export.json", OutputSet: true, PrepareManagedFiles: true,
	})
	require.NoError(t, err)
	require.Equal(t, filepath.Join(cfg.CurrentDir, "export.json"), selection.FilePairs[0].EncryptedPath)
	require.Equal(t, enc, selection.AllConfigPairs[0].EncryptedPath)
	require.Equal(t, enc, selection.ManagedScopes[0].FilePairs[0].EncryptedPath)
}
