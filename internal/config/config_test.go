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

	require.Len(t, cfg.GetFiles(), 1)
	assert.Equal(t, defaultDecryptedFile, cfg.GetFiles()[0].PlaintextPath)
	assert.Equal(t, defaultEncryptedFile, cfg.GetFiles()[0].EncryptedPath)
	assert.Equal(t, defaultKeyFile, cfg.Key.FilePath)
	assert.Empty(t, cfg.Key.PublicKey)
}

func TestGetFilesFallsBackToDefault(t *testing.T) {
	cfg := &Config{}

	files := cfg.GetFiles()
	require.Len(t, files, 1)
	assert.Equal(t, defaultDecryptedFile, files[0].PlaintextPath)
	assert.Equal(t, defaultEncryptedFile, files[0].EncryptedPath)
}

func TestGetPrimaryFilePair(t *testing.T) {
	cfg := &Config{
		Encryption: EncryptionConfig{
			Files: []FilePair{
				{PlaintextPath: "app.toml", EncryptedPath: "app.enc.toml"},
				{PlaintextPath: "db.toml", EncryptedPath: "db.enc.toml"},
			},
		},
	}

	pair := cfg.GetPrimaryFilePair()
	assert.Equal(t, "app.toml", pair.PlaintextPath)
	assert.Equal(t, "app.enc.toml", pair.EncryptedPath)
}

func TestGetKeyFile(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		provided string
		expected string
	}{
		{
			name:     "provided value takes priority",
			config:   &Config{Key: KeyConfig{FilePath: "config/keys.txt"}},
			provided: "custom/keys.txt",
			expected: "custom/keys.txt",
		},
		{
			name:     "empty provided uses config",
			config:   &Config{Key: KeyConfig{FilePath: "config/keys.txt"}},
			provided: "",
			expected: "config/keys.txt",
		},
		{
			name:     "empty config uses default",
			config:   &Config{Key: KeyConfig{FilePath: ""}},
			provided: "",
			expected: defaultKeyFile,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetKeyFile(tt.provided)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetPublicKey(t *testing.T) {
	cfg := &Config{Key: KeyConfig{PublicKey: "age1testkey123"}}
	assert.Equal(t, "age1testkey123", cfg.GetPublicKey())

	emptyCfg := &Config{}
	assert.Empty(t, emptyCfg.GetPublicKey())
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
	require.NoError(t, err)
	require.Len(t, cfg.GetFiles(), 1)

	assert.Equal(t, defaultDecryptedFile, cfg.GetFiles()[0].PlaintextPath)
	assert.Equal(t, defaultEncryptedFile, cfg.GetFiles()[0].EncryptedPath)
	assert.Equal(t, defaultKeyFile, cfg.Key.FilePath)
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

[key]
file_path = "custom/keys.txt"
public_key = "age1customkey"

[sync]
provider = "infisical"
project_id = "project-123"
secret_name = "CUSTOM_AGE_KEY"
path = "/apps/yewseal"
environment = "prod"
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
	assert.Equal(t, filepath.Join(tmpDir, "custom/keys.txt"), cfg.Key.FilePath)
	assert.Equal(t, "age1customkey", cfg.Key.PublicKey)
	assert.Equal(t, "infisical", cfg.Sync.Provider)
	assert.Equal(t, "project-123", cfg.Sync.ProjectID)
	assert.Equal(t, "CUSTOM_AGE_KEY", cfg.Sync.SecretName)
	assert.Equal(t, "/apps/yewseal", cfg.Sync.Path)
	assert.Equal(t, "prod", cfg.Sync.Environment)
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

func TestGetSyncConfig(t *testing.T) {
	cfg := &Config{
		Sync: SyncConfig{
			Provider:    "infisical",
			ProjectID:   "project-config",
			SecretName:  "CONFIG_AGE_KEY",
			Path:        "/config-path",
			Environment: "staging",
		},
	}

	assert.Equal(t, "vault", cfg.GetSyncProvider("vault"))
	assert.Equal(t, "infisical", cfg.GetSyncProvider(""))
	assert.Equal(t, "project-cli", cfg.GetSyncProjectID("project-cli"))
	assert.Equal(t, "project-config", cfg.GetSyncProjectID(""))
	assert.Equal(t, "cli-secret", cfg.GetSyncSecretName("cli-secret"))
	assert.Equal(t, "CONFIG_AGE_KEY", cfg.GetSyncSecretName(""))
	assert.Equal(t, "/cli-path", cfg.GetSyncPath("/cli-path"))
	assert.Equal(t, "/config-path", cfg.GetSyncPath(""))
	assert.Equal(t, "prod", cfg.GetSyncEnvironment("prod"))
	assert.Equal(t, "staging", cfg.GetSyncEnvironment(""))

	emptyCfg := &Config{}
	assert.Equal(t, "infisical", emptyCfg.GetSyncProvider(""))
	assert.Empty(t, emptyCfg.GetSyncProjectID(""))
	assert.Equal(t, "AGE_KEY_FILE", emptyCfg.GetSyncSecretName(""))
	assert.Empty(t, emptyCfg.GetSyncPath(""))
	assert.Empty(t, emptyCfg.GetSyncEnvironment(""))
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

[key]
file_path = ".age/root.txt"
public_key = "age1root"
`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".yewseal.toml"), []byte(rootConfig), 0644))

	childConfig := `[encryption]

[[encryption.files]]
plaintext = ".env"
encrypted = ".env.local.enc.yaml"
format = "env"

[key]
public_key = "age1child"
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
	assert.Equal(t, filepath.Join(tmpDir, ".age", "root.txt"), cfg.Key.FilePath)
	assert.Equal(t, "age1child", cfg.Key.PublicKey)
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
	require.NoError(t, err)
	require.Len(t, cfg.GetFiles(), 1)
	assert.Equal(t, defaultDecryptedFile, cfg.GetFiles()[0].PlaintextPath)
	assert.False(t, cfg.UserConfig)
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
