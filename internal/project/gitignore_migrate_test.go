package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/stretchr/testify/require"
)

func TestMigrateGitignoreReplacesLegacyAndExistingManagedRules(t *testing.T) {
	for _, newline := range []string{"\n", "\r\n"} {
		t.Run(newline, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, ".gitignore")
			prefix := "# User prefix" + newline + "build/" + newline
			suffix := "# User suffix" + newline + "!keep.yaml"
			legacy := legacyPlaintextHeader + newline + "obsolete.yaml" + newline + newline + legacyPrivateHeader + newline + privateKeyPath + newline
			oldBlock := renderGitignoreSection([]string{"also-obsolete.yaml"})
			require.NoError(t, os.WriteFile(path, []byte(prefix+legacy+oldBlock+suffix), 0600))
			pairs := []config.FilePair{{PlaintextPath: "current.yaml"}}
			changed, err := MigrateGitignore(dir, pairs, true)
			require.NoError(t, err)
			require.True(t, changed)
			data, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, prefix+renderGitignoreSection([]string{"current.yaml"})+suffix, string(data))
			changed, err = MigrateGitignore(dir, pairs, true)
			require.NoError(t, err)
			require.False(t, changed)
		})
	}
}

func TestMigrateGitignoreRemovesLegacyFromOldWorkingDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".gitignore")
	prefix := "# Custom\ncache/\n"
	legacy := legacyPlaintextHeader + "\nold.yaml\n\n" + legacyPrivateHeader + "\n" + privateKeyPath
	require.NoError(t, os.WriteFile(path, []byte(prefix+legacy), 0600))
	changed, err := MigrateGitignore(dir, nil, false)
	require.NoError(t, err)
	require.True(t, changed)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, prefix, string(data))
}

func TestMigrateGitignoreReportsFilesystemErrors(t *testing.T) {
	t.Run("read", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, ".gitignore")
		require.NoError(t, os.Mkdir(path, 0700))
		changed, err := MigrateGitignore(dir, nil, true)
		require.Error(t, err)
		require.False(t, changed)
		require.DirExists(t, path)
	})
	t.Run("write", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "missing")
		changed, err := MigrateGitignore(dir, nil, true)
		require.ErrorIs(t, err, os.ErrNotExist)
		require.False(t, changed)
		require.NoDirExists(t, dir)
	})
}

func TestMigrateGitignoreRejectsInvalidMappingBeforeWriting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".gitignore")
	before := []byte(legacyPlaintextHeader + "\nold.yaml\n\n" + legacyPrivateHeader + "\n" + privateKeyPath + "\n")
	require.NoError(t, os.WriteFile(path, before, 0600))
	changed, err := MigrateGitignore(dir, []config.FilePair{{PlaintextPath: "invalid\npath.yaml"}}, true)
	require.ErrorContains(t, err, "line break")
	require.False(t, changed)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, before, data)
}

func TestMigrateGitignorePreservesAmbiguousFiles(t *testing.T) {
	for _, content := range []string{
		legacyPlaintextHeader + "\nold.yaml\n",
		legacyPrivateHeader + "\n" + privateKeyPath + "\n",
		legacyPlaintextHeader + "\n" + legacyPrivateHeader + "\n.age/\n",
		legacyPlaintextHeader + "\n# Handwritten comment\n" + legacyPrivateHeader + "\n" + privateKeyPath + "\n",
		legacyPlaintextHeader + "\n*.yaml\n" + legacyPrivateHeader + "\n" + privateKeyPath + "\n",
		legacyPlaintextHeader + "\n!keep.yaml\n" + legacyPrivateHeader + "\n" + privateKeyPath + "\n",
		legacyPlaintextHeader + "\nold.yaml\n" + legacyPlaintextHeader + "\n" + legacyPrivateHeader + "\n" + privateKeyPath + "\n",
		legacyPlaintextHeader + "\n" + legacyPrivateHeader + "\n" + privateKeyPath + "\n" + legacyPrivateHeader + "\n" + privateKeyPath + "\n",
		managedGitignoreBegin + "\n" + legacyPlaintextHeader + "\nold.yaml\n" + legacyPrivateHeader + "\n" + privateKeyPath + "\n",
	} {
		t.Run(content, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, ".gitignore")
			require.NoError(t, os.WriteFile(path, []byte(content), 0600))
			changed, err := MigrateGitignore(dir, []config.FilePair{{PlaintextPath: "current.yaml"}}, true)
			require.Error(t, err)
			require.False(t, changed)
			data, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, content, string(data))
		})
	}
}
