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

	assert.Equal(t, "wrangler.toml", cfg.Encryption.InputFile)
	assert.Equal(t, "wrangler.enc.yaml", cfg.Encryption.OutputFile)
	assert.Equal(t, ".age/keys.txt", cfg.Key.FilePath)
	assert.Empty(t, cfg.Key.PublicKey)
}

func TestGetEncryptionInput(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		provided  string
		expected  string
	}{
		{
			name:     "provided non-default value takes priority",
			config:   &Config{Encryption: EncryptionConfig{InputFile: "config.toml"}},
			provided: "custom.toml",
			expected: "custom.toml",
		},
		{
			name:     "provided default value uses config",
			config:   &Config{Encryption: EncryptionConfig{InputFile: "config.toml"}},
			provided: "wrangler.toml",
			expected: "config.toml",
		},
		{
			name:     "empty provided uses config",
			config:   &Config{Encryption: EncryptionConfig{InputFile: "config.toml"}},
			provided: "",
			expected: "config.toml",
		},
		{
			name:     "empty config uses default",
			config:   &Config{Encryption: EncryptionConfig{InputFile: ""}},
			provided: "",
			expected: "wrangler.toml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetEncryptionInput(tt.provided)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetEncryptionOutput(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		provided string
		expected string
	}{
		{
			name:     "provided non-default value takes priority",
			config:   &Config{Encryption: EncryptionConfig{OutputFile: "config.enc.yaml"}},
			provided: "custom.enc.yaml",
			expected: "custom.enc.yaml",
		},
		{
			name:     "provided default value uses config",
			config:   &Config{Encryption: EncryptionConfig{OutputFile: "config.enc.yaml"}},
			provided: "wrangler.enc.yaml",
			expected: "config.enc.yaml",
		},
		{
			name:     "empty provided uses config",
			config:   &Config{Encryption: EncryptionConfig{OutputFile: "config.enc.yaml"}},
			provided: "",
			expected: "config.enc.yaml",
		},
		{
			name:     "empty config uses default",
			config:   &Config{Encryption: EncryptionConfig{OutputFile: ""}},
			provided: "",
			expected: "wrangler.enc.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetEncryptionOutput(tt.provided)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetDecryptionInput(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		provided string
		expected string
	}{
		{
			name:     "provided non-default value takes priority",
			config:   &Config{Encryption: EncryptionConfig{OutputFile: "config.enc.yaml"}},
			provided: "custom.enc.yaml",
			expected: "custom.enc.yaml",
		},
		{
			name:     "provided default value uses config output",
			config:   &Config{Encryption: EncryptionConfig{OutputFile: "config.enc.yaml"}},
			provided: "wrangler.enc.yaml",
			expected: "config.enc.yaml",
		},
		{
			name:     "empty provided uses config output",
			config:   &Config{Encryption: EncryptionConfig{OutputFile: "config.enc.yaml"}},
			provided: "",
			expected: "config.enc.yaml",
		},
		{
			name:     "empty config uses default",
			config:   &Config{Encryption: EncryptionConfig{OutputFile: ""}},
			provided: "",
			expected: "wrangler.enc.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetDecryptionInput(tt.provided)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetDecryptionOutput(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		provided string
		expected string
	}{
		{
			name:     "provided non-default value takes priority",
			config:   &Config{Encryption: EncryptionConfig{InputFile: "config.toml"}},
			provided: "custom.toml",
			expected: "custom.toml",
		},
		{
			name:     "provided default value uses config input",
			config:   &Config{Encryption: EncryptionConfig{InputFile: "config.toml"}},
			provided: "wrangler.toml",
			expected: "config.toml",
		},
		{
			name:     "empty provided uses config input",
			config:   &Config{Encryption: EncryptionConfig{InputFile: "config.toml"}},
			provided: "",
			expected: "config.toml",
		},
		{
			name:     "empty config uses default",
			config:   &Config{Encryption: EncryptionConfig{InputFile: ""}},
			provided: "",
			expected: "wrangler.toml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetDecryptionOutput(tt.provided)
			assert.Equal(t, tt.expected, result)
		})
	}
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
			expected: ".age/key.txt",
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
	// Use temp directory with no config file
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)

	err := os.Chdir(tmpDir)
	require.NoError(t, err)

	cfg, err := LoadConfig()
	require.NoError(t, err)

	// Should return default config
	assert.Equal(t, "wrangler.toml", cfg.Encryption.InputFile)
	assert.Equal(t, "wrangler.enc.yaml", cfg.Encryption.OutputFile)
	assert.Equal(t, ".age/keys.txt", cfg.Key.FilePath)
}

func TestLoadConfig_WithFile(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)

	// Create config file
	configContent := `[encryption]
input_file = "custom.toml"
output_file = "custom.enc.yaml"

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

	assert.Equal(t, "custom.toml", cfg.Encryption.InputFile)
	assert.Equal(t, "custom.enc.yaml", cfg.Encryption.OutputFile)
	assert.Equal(t, "custom/keys.txt", cfg.Key.FilePath)
	assert.Equal(t, "age1customkey", cfg.Key.PublicKey)
}

func TestLoadConfig_InvalidToml(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)

	// Create invalid config file
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

	// Create .yewseal directory with higher priority config
	yewsealDir := filepath.Join(tmpDir, ".yewseal")
	err := os.MkdirAll(yewsealDir, 0755)
	require.NoError(t, err)

	// Config in .yewseal/ directory (higher priority)
	highPriorityConfig := `[encryption]
input_file = "high-priority.toml"
`
	err = os.WriteFile(filepath.Join(yewsealDir, ".yewseal.toml"), []byte(highPriorityConfig), 0644)
	require.NoError(t, err)

	// Config in root directory (lower priority)
	lowPriorityConfig := `[encryption]
input_file = "low-priority.toml"
`
	err = os.WriteFile(filepath.Join(tmpDir, ".yewseal.toml"), []byte(lowPriorityConfig), 0644)
	require.NoError(t, err)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	cfg, err := LoadConfig()
	require.NoError(t, err)

	// Should use higher priority config from .yewseal/ directory
	assert.Equal(t, "high-priority.toml", cfg.Encryption.InputFile)
}
