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

func TestScopedConfigGroupPairsSkipsConfiguredCustomEncryptedPaths(t *testing.T) {
	root := t.TempDir()
	secretsDir := filepath.Join(root, "secrets")
	require.NoError(t, os.MkdirAll(secretsDir, 0755))
	plaintext := filepath.Join(secretsDir, "app.yaml")
	customEncrypted := filepath.Join(secretsDir, "app.sops.yaml")
	otherPlaintext := filepath.Join(secretsDir, "other.yaml")
	require.NoError(t, os.WriteFile(plaintext, []byte("token: secret\n"), 0644))
	require.NoError(t, os.WriteFile(customEncrypted, []byte("encrypted"), 0644))
	require.NoError(t, os.WriteFile(otherPlaintext, []byte("token: other\n"), 0644))

	cfg := &Config{
		CurrentDir: root,
		Encryption: EncryptionConfig{
			Files:  []FilePair{{PlaintextPath: plaintext, EncryptedPath: customEncrypted, Format: "yaml"}},
			Groups: []GroupConfig{{Patterns: []string{"secrets/*.yaml"}, ConfigDir: root}},
		},
	}

	pairs, err := scopedConfigGroupPairs(cfg, task.ModeEncrypt)
	require.NoError(t, err)
	require.Len(t, pairs, 1)
	assert.Equal(t, otherPlaintext, pairs[0].PlaintextPath)
}

func TestSelectFilePairsDirectorySkipsConfiguredCustomEncryptedPaths(t *testing.T) {
	root := t.TempDir()
	secretsDir := filepath.Join(root, "secrets")
	require.NoError(t, os.MkdirAll(secretsDir, 0755))
	plaintext := filepath.Join(secretsDir, "app.yaml")
	customEncrypted := filepath.Join(secretsDir, "app.sops.yaml")
	otherPlaintext := filepath.Join(secretsDir, "other.yaml")
	require.NoError(t, os.WriteFile(plaintext, []byte("token: secret\n"), 0644))
	require.NoError(t, os.WriteFile(customEncrypted, []byte("encrypted"), 0644))
	require.NoError(t, os.WriteFile(otherPlaintext, []byte("token: other\n"), 0644))

	cfg := &Config{
		CurrentDir: root,
		Encryption: EncryptionConfig{
			Files:  []FilePair{{PlaintextPath: plaintext, EncryptedPath: customEncrypted, Format: "yaml"}},
			Groups: []GroupConfig{{Patterns: []string{"*.yaml"}, ConfigDir: root}},
		},
	}

	result, err := SelectFilePairs(cfg, SelectionOptions{Command: task.ModeEncrypt, Targets: []string{secretsDir}})
	require.NoError(t, err)
	require.Len(t, result.FilePairs, 2)
	assert.ElementsMatch(t, []string{plaintext, otherPlaintext}, []string{result.FilePairs[0].PlaintextPath, result.FilePairs[1].PlaintextPath})
}

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
		Targets: []string{filepath.Join(root, ".env.enc.yaml")},
	})
	require.NoError(t, err)
	require.Len(t, result.FilePairs, 1)
	assert.Equal(t, filepath.Join(root, ".env"), result.FilePairs[0].PlaintextPath)
	assert.Equal(t, filepath.Join(root, ".env.enc.yaml"), result.FilePairs[0].EncryptedPath)
}

func TestSelectFilePairs_PatternTargetMatchesCommandPrimarySide(t *testing.T) {
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

	// encrypt 匹配明文侧：packages/api/* 只选中 api 的映射
	result, err := SelectFilePairs(cfg, SelectionOptions{
		Command: task.ModeEncrypt,
		Targets: []string{"packages/api/*"},
	})
	require.NoError(t, err)
	require.Len(t, result.FilePairs, 1)
	assert.Equal(t, filepath.Join(root, "packages", "api", ".env"), result.FilePairs[0].PlaintextPath)

	// decrypt 匹配密文侧：*.enc.yaml 选中两个映射
	result, err = SelectFilePairs(cfg, SelectionOptions{
		Command: task.ModeDecrypt,
		Targets: []string{"*.enc.yaml"},
	})
	require.NoError(t, err)
	assert.Len(t, result.FilePairs, 2)

	// view 与 decrypt 一样按密文侧匹配模式目标。
	result, err = SelectFilePairs(cfg, SelectionOptions{
		Command: task.ModeView,
		Targets: []string{"*.enc.yaml"},
	})
	require.NoError(t, err)
	assert.Len(t, result.FilePairs, 2)

	// encrypt 匹配明文侧：*.enc.yaml 不命中任何明文路径，报错
	_, err = SelectFilePairs(cfg, SelectionOptions{
		Command: task.ModeEncrypt,
		Targets: []string{"*.enc.yaml"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "matches no configured file pairs")
}

func TestSelectFilePairs_MultipleTargetsUnionAndDedupe(t *testing.T) {
	root := t.TempDir()
	cfg := &Config{
		CurrentDir: root,
		UserConfig: true,
		Encryption: EncryptionConfig{
			Files: []FilePair{
				{
					PlaintextPath: filepath.Join(root, "a.yaml"),
					EncryptedPath: filepath.Join(root, "a.enc.yaml"),
					Format:        "yaml",
				},
				{
					PlaintextPath: filepath.Join(root, "b.yaml"),
					EncryptedPath: filepath.Join(root, "b.enc.yaml"),
					Format:        "yaml",
				},
			},
		},
	}

	// 精确路径与模式取并集，重叠的映射只选一次
	result, err := SelectFilePairs(cfg, SelectionOptions{
		Command: task.ModeEncrypt,
		Targets: []string{filepath.Join(root, "a.yaml"), "*.yaml"},
	})
	require.NoError(t, err)
	assert.Len(t, result.FilePairs, 2)
}

func TestSelectFilePairs_DirectoryTargetSetsCurrentDirScope(t *testing.T) {
	root := t.TempDir()
	subDir := filepath.Join(root, "configs")
	require.NoError(t, os.MkdirAll(subDir, 0755))
	cfg := &Config{
		CurrentDir: root,
		UserConfig: true,
		Encryption: EncryptionConfig{
			Files: []FilePair{
				{
					PlaintextPath: filepath.Join(subDir, "a.yaml"),
					EncryptedPath: filepath.Join(subDir, "a.enc.yaml"),
					Format:        "yaml",
				},
			},
		},
	}

	result, err := SelectFilePairs(cfg, SelectionOptions{Command: task.ModeEncrypt, Targets: []string{subDir}})
	require.NoError(t, err)
	require.Len(t, result.FilePairs, 1)
	assert.Equal(t, subDir, result.CurrentDirScope)
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

	_, err := ResolveSelection(cfg, SelectionOptions{
		Command: task.ModePlan,
		Targets: []string{filepath.Join(root, ".dev.vars")},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestSelectFilePairsPatternCannotCreateTemporaryGroup(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "secret.yaml"), []byte("secret: value\n"), 0644))
	cfg := &Config{CurrentDir: root}
	_, err := SelectFilePairs(cfg, SelectionOptions{Command: task.ModeEncrypt, Targets: []string{"*.yaml"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "matches no configured file pairs")
}

func TestResolvePlanSelectionEncryptedTargetStillRequiresAuthorization(t *testing.T) {
	root := t.TempDir()
	encrypted := filepath.Join(root, "config.enc.yaml")
	require.NoError(t, os.WriteFile(encrypted, []byte("encrypted"), 0600))
	cfg := &Config{CurrentDir: root, Encryption: EncryptionConfig{Files: []FilePair{{PlaintextPath: filepath.Join(root, "config.yaml"), EncryptedPath: encrypted, Format: "yaml"}}}}
	_, err := ResolveSelection(cfg, SelectionOptions{Command: task.ModePlan, Targets: []string{encrypted}})
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

	result, err := ResolveSelection(cfg, SelectionOptions{Command: task.ModePlan})
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
	result, err := ResolveSelection(cfg, SelectionOptions{Command: task.ModeEncrypt, Targets: []string{path}})
	require.NoError(t, err)
	require.Len(t, result.FilePairs, 1)
	assert.Equal(t, filepath.Join(root, "explicit.enc.yaml"), result.FilePairs[0].EncryptedPath)
	assert.Equal(t, []string{"first"}, result.FilePairs[0].RecipientAliases)
}

func TestDedupeFilePairsRejectsConflictingExplicitPairs(t *testing.T) {
	_, err := dedupeFilePairs([]FilePair{
		{PlaintextPath: "secret.yaml", EncryptedPath: "secret.enc.yaml", Source: PairSourceExact},
		{PlaintextPath: "secret.yaml", EncryptedPath: "other.enc.yaml", Source: PairSourceExact},
	}, task.ModeEncrypt)
	require.EqualError(t, err, `conflicting file pairs for plaintext "secret.yaml" or encrypted "other.enc.yaml"`)
}

func TestDiffRejectsConflictingGroupMappings(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"config.yaml", "config.yml", "config.enc.yaml"} {
		require.NoError(t, os.WriteFile(filepath.Join(root, name), []byte("unused"), 0600))
	}
	cfg := &Config{CurrentDir: root, Encryption: EncryptionConfig{Groups: []GroupConfig{
		{Patterns: []string{"*.yaml"}, ConfigDir: root},
		{Patterns: []string{"*.yml"}, ConfigDir: root},
	}}}
	_, err := SelectFilePairs(cfg, SelectionOptions{Command: task.ModeDiff})
	require.ErrorContains(t, err, "conflicting group file pairs for comparison")
	cfg.Encryption.Files = []FilePair{{PlaintextPath: filepath.Join(root, "custom.yaml"), EncryptedPath: filepath.Join(root, "config.enc.yaml"), Format: "yaml"}}
	for _, target := range []string{"", root} {
		selection, err := SelectFilePairs(cfg, SelectionOptions{Command: task.ModeDiff, Targets: []string{target}})
		require.NoError(t, err)
		require.Len(t, selection.FilePairs, 1)
		require.Equal(t, filepath.Join(root, "custom.yaml"), selection.FilePairs[0].PlaintextPath)
	}
}
