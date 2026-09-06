package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Empty(t, cfg.GetFiles())
}

func TestGetFilesDoesNotFallBackToDefault(t *testing.T) {
	cfg := &Config{}

	assert.Empty(t, cfg.GetFiles())
}

func TestGetFilesClonesRecipientSlices(t *testing.T) {
	recipients := []string{"owner"}
	cfg := &Config{Encryption: EncryptionConfig{Files: []FilePair{{Recipients: &recipients}}}}
	files := cfg.GetFiles()
	(*files[0].Recipients)[0] = "changed"
	require.Equal(t, "owner", (*cfg.Encryption.Files[0].Recipients)[0])
}

func TestLoadConfig_NoFile(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.Chdir(oldWd))
	}()

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	cfg, err := LoadConfig()
	require.ErrorContains(t, err, "no YewSeal configuration found")
	assert.Nil(t, cfg)
}

func TestLoadConfigEmptyFileIsNotMissing(t *testing.T) {
	t.Chdir(t.TempDir())
	require.NoError(t, os.WriteFile(".yewseal.toml", nil, 0600))
	cfg, err := LoadConfig()
	require.NoError(t, err)
	require.Len(t, cfg.LoadedFiles, 1)
	require.Empty(t, cfg.GetFiles())
}

func TestLoadConfig_WithFile(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.Chdir(oldWd))
	}()

	configContent := `[encryption]

[[encryption.groups]]
patterns = ["*.toml", "!*.example.toml"]
format_rules = [".dev.vars=env"]
unknown_as_binary = true

[[encryption.groups]]
patterns = [".dev.vars"]
format_rules = [".dev.vars=env"]

[[encryption.files]]
plaintext = "custom.toml"
encrypted = "custom.enc.yaml"

[[encryption.files]]
plaintext = ".dev.vars"
encrypted = ".dev.vars.enc.yaml"
format = "env"

[recipients.registry]
owner = "age1r09mha3l82nt25r3kujgkpw4ts60ezntwcj74vnk0t3e9elyu3rswkx08j"

`
	err = os.WriteFile(filepath.Join(tmpDir, ".yewseal.toml"), []byte(configContent), 0644)
	require.NoError(t, err)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	cfg, err := LoadConfig()
	require.NoError(t, err)
	require.Len(t, cfg.GetFiles(), 2)

	assert.Equal(t, filepath.Join(tmpDir, "custom.toml"), cfg.GetFiles()[0].PlaintextPath)
	assert.Equal(t, filepath.Join(tmpDir, "custom.enc.yaml"), cfg.GetFiles()[0].EncryptedPath)
	assert.Equal(t, filepath.Join(tmpDir, ".dev.vars"), cfg.GetFiles()[1].PlaintextPath)
	assert.Equal(t, filepath.Join(tmpDir, ".dev.vars.enc.yaml"), cfg.GetFiles()[1].EncryptedPath)
	assert.Equal(t, "env", cfg.GetFiles()[1].Format)
	groups := cfg.GetGroups()
	require.Len(t, groups, 2)
	assert.Equal(t, []string{"*.toml", "!*.example.toml"}, groups[0].Patterns)
	assert.Equal(t, []string{".dev.vars=env"}, groups[0].FormatRules)
	assert.True(t, groups[0].UnknownAsBinary)
	assert.Equal(t, []string{".dev.vars"}, groups[1].Patterns)
	assert.Equal(t, []string{".dev.vars=env"}, groups[1].FormatRules)
	assert.False(t, groups[1].UnknownAsBinary)
	assert.Equal(t, "age1r09mha3l82nt25r3kujgkpw4ts60ezntwcj74vnk0t3e9elyu3rswkx08j", cfg.Recipients.Registry["owner"])
}

func TestLoadConfig_RejectsLegacyGroupTable(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.Chdir(oldWd))
	}()

	configContent := `[encryption]

[encryption.group]
patterns = ["*.toml"]
`
	err = os.WriteFile(filepath.Join(tmpDir, ".yewseal.toml"), []byte(configContent), 0644)
	require.NoError(t, err)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	_, err = LoadConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "use [[encryption.groups]] instead")
}

func TestLoadConfigRejectsDeprecatedPublicKey(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, os.Chdir(oldWd)) })
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".yewseal.toml"), []byte("[key]\npublic_key = \"age1legacy\"\n"), 0644))
	require.NoError(t, os.Chdir(tmpDir))
	_, err = LoadConfig()
	require.EqualError(t, err, "deprecated key.public_key is not supported; use recipients.registry and recipients.defaults")
}

func TestLoadConfigRejectsKeyFilePath(t *testing.T) {
	for _, value := range []string{`"custom/keys.txt"`, `""`} {
		t.Run(value, func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)
			require.NoError(t, os.WriteFile(filepath.Join(dir, ".yewseal.toml"), []byte("[key]\nfile_path = "+value+"\n"), 0600))
			_, err := LoadConfig()
			require.EqualError(t, err, "key.file_path is no longer supported; use --key-file or SOPS_AGE_KEY_FILE instead")
		})
	}
}

func TestLoadConfig_InvalidToml(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.Chdir(oldWd))
	}()

	invalidContent := `this is not valid toml [[[`
	err = os.WriteFile(filepath.Join(tmpDir, ".yewseal.toml"), []byte(invalidContent), 0644)
	require.NoError(t, err)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	_, err = LoadConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse config file")
}

func TestLoadConfig_MultipleLocations(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.Chdir(oldWd))
	}()

	yewsealDir := filepath.Join(tmpDir, ".yewseal")
	err = os.MkdirAll(yewsealDir, 0755)
	require.NoError(t, err)

	highPriorityConfig := `[encryption]

[[encryption.files]]
plaintext = "high-priority.toml"
encrypted = "high-priority.enc.toml"
`
	err = os.WriteFile(filepath.Join(yewsealDir, ".yewseal.toml"), []byte(highPriorityConfig), 0644)
	require.NoError(t, err)

	lowPriorityConfig := `[encryption]

[[encryption.files]]
plaintext = "low-priority.toml"
encrypted = "low-priority.enc.toml"
`
	err = os.WriteFile(filepath.Join(tmpDir, ".yewseal.toml"), []byte(lowPriorityConfig), 0644)
	require.NoError(t, err)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	cfg, err := LoadConfig()
	require.NoError(t, err)

	assert.Equal(t, filepath.Join(tmpDir, "high-priority.toml"), cfg.GetFiles()[0].PlaintextPath)
	assert.Equal(t, filepath.Join(tmpDir, "high-priority.enc.toml"), cfg.GetFiles()[0].EncryptedPath)
}

func TestLoadConfig_DiscoversAndMergesToGitRoot(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.Chdir(oldWd))
	}()

	require.NoError(t, os.Mkdir(filepath.Join(tmpDir, ".git"), 0755))
	apiDir := filepath.Join(tmpDir, "packages", "api")
	require.NoError(t, os.MkdirAll(apiDir, 0755))

	rootConfig := `[encryption]

[[encryption.files]]
plaintext = "packages/api/.env"
encrypted = "packages/api/.env.enc.yaml"
format = "env"

[[encryption.files]]
plaintext = "shared.toml"
encrypted = "shared.enc.toml"

`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".yewseal.toml"), []byte(rootConfig), 0644))

	childConfig := `[encryption]

[[encryption.files]]
plaintext = ".env"
encrypted = ".env.local.enc.yaml"
format = "env"

`
	require.NoError(t, os.WriteFile(filepath.Join(apiDir, ".yewseal.toml"), []byte(childConfig), 0644))
	require.NoError(t, os.Chdir(apiDir))

	cfg, err := LoadConfig()
	require.NoError(t, err)

	require.Len(t, cfg.LoadedFiles, 2)
	files := cfg.GetFiles()
	require.Len(t, files, 2)
	assert.Equal(t, filepath.Join(tmpDir, "shared.toml"), files[0].PlaintextPath)
	assert.Equal(t, filepath.Join(apiDir, ".env"), files[1].PlaintextPath)
	assert.Equal(t, filepath.Join(apiDir, ".env.local.enc.yaml"), files[1].EncryptedPath)
}

func TestLoadConfig_StopsAtNearestGitRoot(t *testing.T) {
	tmpDir, err := os.MkdirTemp("/tmp", "yewseal-git-boundary-*")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.RemoveAll(tmpDir))
	}()
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.Chdir(oldWd))
	}()

	childDir := filepath.Join(tmpDir, "child")
	require.NoError(t, os.MkdirAll(childDir, 0755))
	require.NoError(t, os.Mkdir(filepath.Join(childDir, ".git"), 0755))
	parentConfig := `[encryption]

[[encryption.files]]
plaintext = "parent.toml"
encrypted = "parent.enc.toml"
`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".yewseal.toml"), []byte(parentConfig), 0644))
	require.NoError(t, os.Chdir(childDir))

	cfg, err := LoadConfig()
	require.ErrorContains(t, err, "no YewSeal configuration found")
	assert.Nil(t, cfg)
}

func TestLoadConfig_NonGitUsesCurrentDirectoryConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("/tmp", "yewseal-current-config-*")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.RemoveAll(tmpDir))
	}()
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.Chdir(oldWd))
	}()

	configContent := `[encryption]

[[encryption.files]]
plaintext = "local.toml"
encrypted = "local.enc.toml"
`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".yewseal.toml"), []byte(configContent), 0644))
	require.NoError(t, os.Chdir(tmpDir))

	cfg, err := LoadConfig()
	require.NoError(t, err)
	require.True(t, cfg.UserConfig)
	require.Len(t, cfg.GetFiles(), 1)
	assert.Equal(t, filepath.Join(tmpDir, "local.toml"), cfg.GetFiles()[0].PlaintextPath)
}
