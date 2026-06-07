package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/YewFence/YewSeal/internal/task"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelectFilePairs_ConfigModeFiltersEncryptByCurrentPlaintextScope(t *testing.T) {
	root := t.TempDir()
	apiDir := filepath.Join(root, "packages", "api")
	cfg := &Config{
		CurrentDir: apiDir,
		UserConfig: true,
		Encryption: EncryptionConfig{
			Files: []FilePair{
				{
					PlaintextPath: filepath.Join(root, "packages", "api", ".env"),
					EncryptedPath: filepath.Join(root, "packages", "api", ".env.enc.yaml"),
					Format:        "env",
				},
				{
					PlaintextPath: filepath.Join(root, "packages", "web", ".env"),
					EncryptedPath: filepath.Join(root, "packages", "web", ".env.enc.yaml"),
					Format:        "env",
				},
			},
		},
	}

	result, err := SelectFilePairs(cfg, SelectionOptions{
		Command:          task.ModeEncrypt,
		AllowEmptyTarget: true,
	})
	require.NoError(t, err)
	require.Len(t, result.FilePairs, 1)
	assert.Equal(t, filepath.Join(apiDir, ".env"), result.FilePairs[0].PlaintextPath)
}

func TestSelectFilePairs_ConfigModeFiltersDecryptByCurrentEncryptedScope(t *testing.T) {
	root := t.TempDir()
	apiDir := filepath.Join(root, "packages", "api")
	cfg := &Config{
		CurrentDir: apiDir,
		UserConfig: true,
		Encryption: EncryptionConfig{
			Files: []FilePair{
				{
					PlaintextPath: filepath.Join(root, "outside", ".env"),
					EncryptedPath: filepath.Join(apiDir, ".env.enc.yaml"),
					Format:        "env",
				},
				{
					PlaintextPath: filepath.Join(root, "outside", "web.env"),
					EncryptedPath: filepath.Join(root, "packages", "web", ".env.enc.yaml"),
					Format:        "env",
				},
			},
		},
	}

	result, err := SelectFilePairs(cfg, SelectionOptions{
		Command:          task.ModeDecrypt,
		AllowEmptyTarget: true,
	})
	require.NoError(t, err)
	require.Len(t, result.FilePairs, 1)
	assert.Equal(t, filepath.Join(apiDir, ".env.enc.yaml"), result.FilePairs[0].EncryptedPath)
}

func TestSelectFilePairs_TargetMatchesEitherSideOfConfiguredPair(t *testing.T) {
	root := t.TempDir()
	cfg := &Config{
		CurrentDir: root,
		UserConfig: true,
		Encryption: EncryptionConfig{
			Files: []FilePair{
				{
					PlaintextPath: filepath.Join(root, ".env"),
					EncryptedPath: filepath.Join(root, ".env.enc.yaml"),
					Format:        "env",
				},
			},
		},
	}

	result, err := SelectFilePairs(cfg, SelectionOptions{
		Command: task.ModeEncrypt,
		Target:  filepath.Join(root, ".env.enc.yaml"),
	})
	require.NoError(t, err)
	require.Len(t, result.FilePairs, 1)
	assert.Equal(t, filepath.Join(root, ".env"), result.FilePairs[0].PlaintextPath)
	assert.Equal(t, filepath.Join(root, ".env.enc.yaml"), result.FilePairs[0].EncryptedPath)
}

func TestSelectFilePairs_PatternFiltersPlaintextAndEncryptedPaths(t *testing.T) {
	root := t.TempDir()
	cfg := &Config{
		CurrentDir: root,
		UserConfig: true,
		Encryption: EncryptionConfig{
			Files: []FilePair{
				{
					PlaintextPath: filepath.Join(root, "packages", "api", ".env"),
					EncryptedPath: filepath.Join(root, "packages", "api", ".env.enc.yaml"),
					Format:        "env",
				},
				{
					PlaintextPath: filepath.Join(root, "packages", "web", "config.yaml"),
					EncryptedPath: filepath.Join(root, "packages", "web", "config.enc.yaml"),
					Format:        "yaml",
				},
			},
		},
	}

	result, err := SelectFilePairs(cfg, SelectionOptions{
		Command:          task.ModeEncrypt,
		AllowEmptyTarget: true,
		Patterns:         []string{"*.enc.yaml", "!packages/web/**"},
	})
	require.NoError(t, err)
	require.Len(t, result.FilePairs, 1)
	assert.Equal(t, filepath.Join(root, "packages", "api", ".env"), result.FilePairs[0].PlaintextPath)
}

func TestPathWithinHandlesMixedSeparators(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "packages", "api", ".env")
	mixedPath := filepath.ToSlash(path)

	inside, err := pathWithin(root, mixedPath)
	require.NoError(t, err)

	assert.True(t, inside)
}

func TestResolvePlanSelection_TargetShowsArgumentAndProtocolSources(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, ".dev.vars")
	require.NoError(t, os.WriteFile(target, []byte("TOKEN=secret\n"), 0644))
	cfg := &Config{CurrentDir: root}

	result, err := ResolvePlanSelection(cfg, SelectionOptions{
		Target: target,
		Format: "env",
	})
	require.NoError(t, err)
	require.Len(t, result.FilePairs, 1)

	pair := result.FilePairs[0]
	assert.Equal(t, "plan", result.Command)
	assert.Equal(t, ValueSourceArgument, pair.PlaintextSource.Kind)
	assert.Equal(t, ValueSourceProtocol, pair.EncryptedSource.Kind)
	assert.Equal(t, ValueSourceArgument, pair.FormatSource.Kind)
	assert.Equal(t, SelectedByPathTarget, pair.SelectedBy)
	assert.Equal(t, filepath.Join(root, ".dev.vars.enc.env"), pair.EncryptedPath)
}

func TestResolvePlanSelection_NoTargetUsesEitherSideCurrentScope(t *testing.T) {
	root := t.TempDir()
	apiDir := filepath.Join(root, "packages", "api")
	cfg := &Config{
		CurrentDir: apiDir,
		UserConfig: true,
		Encryption: EncryptionConfig{
			Files: []FilePair{
				{
					PlaintextPath: filepath.Join(root, "outside", ".env"),
					EncryptedPath: filepath.Join(apiDir, ".env.enc.yaml"),
					Format:        "env",
					ConfigPath:    filepath.Join(root, ".yewseal.toml"),
					Source:        "exact",
				},
			},
		},
	}

	result, err := ResolvePlanSelection(cfg, SelectionOptions{})
	require.NoError(t, err)
	require.Len(t, result.FilePairs, 1)
	assert.Equal(t, filepath.Join(apiDir, ".env.enc.yaml"), result.FilePairs[0].EncryptedPath)
	assert.Equal(t, SelectedByCurrentDirectory, result.FilePairs[0].SelectedBy)
}

func TestCheckWriteConflictsDecryptReportsEncryptedSources(t *testing.T) {
	target := filepath.Join(t.TempDir(), "plain.toml")
	err := checkWriteConflicts(task.ModeDecrypt, []ResolvedFilePair{
		{
			PlaintextPath: target,
			EncryptedPath: "first.enc.toml.yaml",
		},
		{
			PlaintextPath: target,
			EncryptedPath: "second.enc.toml.yaml",
		},
	})
	require.Error(t, err)

	assert.Contains(t, err.Error(), "decrypted from first.enc.toml.yaml and second.enc.toml.yaml")
}
