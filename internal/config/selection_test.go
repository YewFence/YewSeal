package config

import (
	"os"
	"path/filepath"
	"testing"

	"filippo.io/age"

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

func TestResolvePlanSelection_RejectsUnconfiguredTarget(t *testing.T) {
	root := t.TempDir()
	cfg := &Config{CurrentDir: root}

	_, err := ResolvePlanSelection(cfg, SelectionOptions{
		Target: filepath.Join(root, ".dev.vars"),
		Format: "env",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestSelectFilePairsPatternCannotCreateTemporaryGroup(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "secret.yaml"), []byte("secret: value\n"), 0644))
	cfg := &Config{CurrentDir: root}
	_, err := SelectFilePairs(cfg, SelectionOptions{Command: task.ModeEncrypt, Patterns: []string{"*.yaml"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no configured file pairs selected")
}

func TestResolvePlanSelectionEncryptedTargetStillRequiresAuthorization(t *testing.T) {
	root := t.TempDir()
	encrypted := filepath.Join(root, "config.enc.yaml")
	require.NoError(t, os.WriteFile(encrypted, []byte("encrypted"), 0600))
	cfg := &Config{CurrentDir: root, Encryption: EncryptionConfig{Files: []FilePair{{PlaintextPath: filepath.Join(root, "config.yaml"), EncryptedPath: encrypted, Format: "yaml"}}}}
	_, err := ResolvePlanSelection(cfg, SelectionOptions{Target: encrypted})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no recipient set")
}

func TestResolvePlanSelection_NoTargetUsesEitherSideCurrentScope(t *testing.T) {
	root := t.TempDir()
	apiDir := filepath.Join(root, "packages", "api")
	cfg := &Config{
		CurrentDir: apiDir,
		UserConfig: true,
		Recipients: RecipientConfig{Defaults: func() *[]string { values := []string{"owner"}; return &values }(), Registry: map[string]string{"owner": "age1r09mha3l82nt25r3kujgkpw4ts60ezntwcj74vnk0t3e9elyu3rswkx08j"}},
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
			EncryptedPath: "first.enc.toml",
		},
		{
			PlaintextPath: target,
			EncryptedPath: "second.enc.toml",
		},
	})
	require.Error(t, err)

	assert.Contains(t, err.Error(), "decrypted from first.enc.toml and second.enc.toml")
}

func TestResolveSelectionGroupAuthorizationProvenance(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "config", "app.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
	require.NoError(t, os.WriteFile(path, []byte("value: true\n"), 0644))
	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	recipients := []string{"ops"}
	cfg := &Config{CurrentDir: root, UserConfig: true, Recipients: RecipientConfig{Registry: map[string]string{"ops": identity.Recipient().String()}}, Encryption: EncryptionConfig{Groups: []GroupConfig{{Patterns: []string{"config/*.yaml"}, Recipients: &recipients}}}}
	result, err := ResolveSelection(cfg, SelectionOptions{Command: task.ModeEncrypt})
	require.NoError(t, err)
	require.Len(t, result.FilePairs, 1)
	pair := result.FilePairs[0]
	require.Equal(t, []string{"ops"}, pair.RecipientAliases)
	require.Equal(t, []string{identity.Recipient().String()}, pair.Recipients)
	require.Equal(t, "group", pair.RecipientInfo.Kind)
	require.Equal(t, "group", pair.RecipientInfo.EffectiveSource.Kind)
}

func TestResolveSelectionGroupInheritsDefaultsProvenance(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("value: true\n"), 0644))
	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	defaults := []string{"owner"}
	cfg := &Config{CurrentDir: root, Recipients: RecipientConfig{Defaults: &defaults, DefaultsConfigPath: filepath.Join(root, ".yewseal.toml"), Registry: map[string]string{"owner": identity.Recipient().String()}}, Encryption: EncryptionConfig{Groups: []GroupConfig{{Patterns: []string{"config.yaml"}, ConfigDir: root}}}}
	result, err := ResolveSelection(cfg, SelectionOptions{Command: task.ModeEncrypt})
	require.NoError(t, err)
	require.Len(t, result.FilePairs, 1)
	assert.Equal(t, "defaults", result.FilePairs[0].RecipientInfo.Kind)
	assert.Equal(t, "defaults", result.FilePairs[0].RecipientInfo.EffectiveSource.Kind)
	assert.Equal(t, filepath.Join(root, ".yewseal.toml"), result.FilePairs[0].RecipientInfo.EffectiveSource.ConfigPath)
}

func TestResolveSelectionRejectsConflictingGroups(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "config", "app.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
	require.NoError(t, os.WriteFile(path, []byte("value: true\n"), 0644))
	first, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	second, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	firstAliases := []string{"first"}
	secondAliases := []string{"second"}
	cfg := &Config{CurrentDir: root, UserConfig: true, Recipients: RecipientConfig{Registry: map[string]string{"first": first.Recipient().String(), "second": second.Recipient().String()}}, Encryption: EncryptionConfig{Groups: []GroupConfig{{Patterns: []string{"config/*.yaml"}, Recipients: &firstAliases}, {Patterns: []string{"config/*.yaml"}, Recipients: &secondAliases}}}}
	_, err = ResolveSelection(cfg, SelectionOptions{Command: task.ModeEncrypt})
	require.Error(t, err)
	require.Contains(t, err.Error(), "conflicting recipient sets")
}

func TestResolveSelectionExplicitFileOverridesConflictingGroups(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "config", "app.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
	require.NoError(t, os.WriteFile(path, []byte("value: true\n"), 0644))
	first, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	second, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	firstAliases := []string{"first"}
	secondAliases := []string{"second"}
	explicitAliases := []string{"first"}
	cfg := &Config{CurrentDir: root, UserConfig: true, Recipients: RecipientConfig{Registry: map[string]string{"first": first.Recipient().String(), "second": second.Recipient().String()}}, Encryption: EncryptionConfig{Files: []FilePair{{PlaintextPath: path, EncryptedPath: filepath.Join(root, "explicit.enc.yaml"), Format: "yaml", Recipients: &explicitAliases}}, Groups: []GroupConfig{{Patterns: []string{"config/*.yaml"}, ConfigDir: root, Recipients: &firstAliases}, {Patterns: []string{"config/*.yaml"}, ConfigDir: root, Recipients: &secondAliases}}}}
	result, err := ResolveSelection(cfg, SelectionOptions{Command: task.ModeEncrypt, Target: path})
	require.NoError(t, err)
	require.Len(t, result.FilePairs, 1)
	assert.Equal(t, filepath.Join(root, "explicit.enc.yaml"), result.FilePairs[0].EncryptedPath)
	assert.Equal(t, []string{"first"}, result.FilePairs[0].RecipientAliases)
}

func TestDedupeFilePairsRejectsConflictingExplicitPairs(t *testing.T) {
	_, err := dedupeFilePairs([]FilePair{
		{PlaintextPath: "secret.yaml", EncryptedPath: "secret.enc.yaml", Source: PairSourceExact},
		{PlaintextPath: "secret.yaml", EncryptedPath: "other.enc.yaml", Source: PairSourceExact},
	})
	require.EqualError(t, err, `conflicting file pairs for plaintext "secret.yaml" or encrypted "other.enc.yaml"`)
}
