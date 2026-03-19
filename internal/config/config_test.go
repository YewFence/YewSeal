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
	assert.Equal(t, defaultDecryptedFile, cfg.GetFiles()[0].Dec)
	assert.Equal(t, defaultEncryptedFile, cfg.GetFiles()[0].Enc)
	assert.Equal(t, defaultKeyFile, cfg.Key.FilePath)
	assert.Empty(t, cfg.Key.PublicKey)
}

func TestGetFilesFallsBackToDefault(t *testing.T) {
	cfg := &Config{}

	files := cfg.GetFiles()
	require.Len(t, files, 1)
	assert.Equal(t, defaultDecryptedFile, files[0].Dec)
	assert.Equal(t, defaultEncryptedFile, files[0].Enc)
}

func TestGetPrimaryFilePair(t *testing.T) {
	cfg := &Config{
		Encryption: EncryptionConfig{
			Files: []FilePair{
				{Dec: "app.toml", Enc: "app.enc.toml.yaml"},
				{Dec: "db.toml", Enc: "db.enc.toml.yaml"},
			},
		},
	}

	pair := cfg.GetPrimaryFilePair()
	assert.Equal(t, "app.toml", pair.Dec)
	assert.Equal(t, "app.enc.toml.yaml", pair.Enc)
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
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)

	err := os.Chdir(tmpDir)
	require.NoError(t, err)

	cfg, err := LoadConfig()
	require.NoError(t, err)
	require.Len(t, cfg.GetFiles(), 1)

	assert.Equal(t, defaultDecryptedFile, cfg.GetFiles()[0].Dec)
	assert.Equal(t, defaultEncryptedFile, cfg.GetFiles()[0].Enc)
	assert.Equal(t, defaultKeyFile, cfg.Key.FilePath)
}

func TestLoadConfig_WithFile(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)

	configContent := `[encryption]

[[encryption.files]]
dec = "custom.toml"
enc = "custom.enc.yaml"

[[encryption.files]]
dec = ".dev.vars"
enc = ".dev.vars.enc.yaml"
format = "env"

[key]
file_path = "custom/keys.txt"
public_key = "age1customkey"
`
	err := os.WriteFile(filepath.Join(tmpDir, ".yewseal.toml"), []byte(configContent), 0644)
	require.NoError(t, err)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	cfg, err := LoadConfig()
	require.NoError(t, err)
	require.Len(t, cfg.GetFiles(), 2)

	assert.Equal(t, "custom.toml", cfg.GetFiles()[0].Dec)
	assert.Equal(t, "custom.enc.yaml", cfg.GetFiles()[0].Enc)
	assert.Equal(t, ".dev.vars", cfg.GetFiles()[1].Dec)
	assert.Equal(t, ".dev.vars.enc.yaml", cfg.GetFiles()[1].Enc)
	assert.Equal(t, "env", cfg.GetFiles()[1].Format)
	assert.Equal(t, "custom/keys.txt", cfg.Key.FilePath)
	assert.Equal(t, "age1customkey", cfg.Key.PublicKey)
}

func TestLoadConfig_InvalidToml(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)

	invalidContent := `this is not valid toml [[[`
	err := os.WriteFile(filepath.Join(tmpDir, ".yewseal.toml"), []byte(invalidContent), 0644)
	require.NoError(t, err)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	_, err = LoadConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse config file")
}

func TestLoadConfig_MultipleLocations(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)

	yewsealDir := filepath.Join(tmpDir, ".yewseal")
	err := os.MkdirAll(yewsealDir, 0755)
	require.NoError(t, err)

	highPriorityConfig := `[encryption]

[[encryption.files]]
dec = "high-priority.toml"
enc = "high-priority.enc.toml.yaml"
`
	err = os.WriteFile(filepath.Join(yewsealDir, ".yewseal.toml"), []byte(highPriorityConfig), 0644)
	require.NoError(t, err)

	lowPriorityConfig := `[encryption]

[[encryption.files]]
dec = "low-priority.toml"
enc = "low-priority.enc.toml.yaml"
`
	err = os.WriteFile(filepath.Join(tmpDir, ".yewseal.toml"), []byte(lowPriorityConfig), 0644)
	require.NoError(t, err)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	cfg, err := LoadConfig()
	require.NoError(t, err)

	assert.Equal(t, "high-priority.toml", cfg.GetFiles()[0].Dec)
	assert.Equal(t, "high-priority.enc.toml.yaml", cfg.GetFiles()[0].Enc)
}
