package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const (
	defaultDecryptedFile = "wrangler.toml"
	defaultEncryptedFile = "wrangler.enc.toml.yaml"
	defaultKeyFile       = ".age/keys.txt"
)

// Config represents the YewSeal configuration.
type Config struct {
	Encryption EncryptionConfig `toml:"encryption"`
	Key        KeyConfig        `toml:"key"`
}

// EncryptionConfig defines encrypted file mappings.
type EncryptionConfig struct {
	Files []FilePair `toml:"files"`
}

// FilePair defines one plaintext/encrypted file mapping.
type FilePair struct {
	// Dec is the plaintext file path used by encrypt input and decrypt output.
	Dec string `toml:"dec"`
	// Enc is the encrypted file path used by encrypt output and decrypt input.
	Enc string `toml:"enc"`
	// Format overrides the file format detection (toml/yaml/json/env/ini).
	// Useful for files with non-standard extensions like .dev.vars.
	Format string `toml:"format,omitempty"`
}

// KeyConfig defines key file location.
type KeyConfig struct {
	// FilePath is the path to Age private key file.
	// Do NOT store the actual key value here to avoid leaking secrets.
	FilePath string `toml:"file_path"`
	// PublicKey is the Age public key for encryption (safe to commit).
	PublicKey string `toml:"public_key"`
}

// DefaultFilePair returns the default plaintext/encrypted mapping.
func DefaultFilePair() FilePair {
	return FilePair{
		Dec: defaultDecryptedFile,
		Enc: defaultEncryptedFile,
	}
}

// DefaultConfig returns a config with default values.
func DefaultConfig() *Config {
	return &Config{
		Encryption: EncryptionConfig{
			Files: []FilePair{DefaultFilePair()},
		},
		Key: KeyConfig{
			FilePath: defaultKeyFile,
		},
	}
}

// LoadConfig loads configuration from .yewseal.toml.
// Searches in the following locations (in order):
// 1. .yewseal/.yewseal.toml
// 2. .config/.yewseal.toml
// 3. .yewseal.toml
// Returns default config if file doesn't exist.
func LoadConfig() (*Config, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return DefaultConfig(), nil
	}

	configPaths := []string{
		filepath.Join(cwd, ".yewseal", ".yewseal.toml"),
		filepath.Join(cwd, ".config", ".yewseal.toml"),
		filepath.Join(cwd, ".yewseal.toml"),
	}

	var configPath string
	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			configPath = path
			break
		}
	}

	if configPath == "" {
		return DefaultConfig(), nil
	}

	config := DefaultConfig()
	if _, err := toml.DecodeFile(configPath, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", configPath, err)
	}

	if len(config.Encryption.Files) == 0 {
		config.Encryption.Files = []FilePair{DefaultFilePair()}
	}

	return config, nil
}

// GetKeyFile returns the key file path.
// Priority: provided value > config file > default.
func (c *Config) GetKeyFile(provided string) string {
	if provided != "" {
		return provided
	}
	if c.Key.FilePath != "" {
		return c.Key.FilePath
	}
	return defaultKeyFile
}

// GetPublicKey returns the Age public key.
func (c *Config) GetPublicKey() string {
	return c.Key.PublicKey
}

// GetFiles returns configured file mappings.
func (c *Config) GetFiles() []FilePair {
	if len(c.Encryption.Files) == 0 {
		return []FilePair{DefaultFilePair()}
	}
	return c.Encryption.Files
}

// GetPrimaryFilePair returns the first configured file mapping.
func (c *Config) GetPrimaryFilePair() FilePair {
	files := c.GetFiles()
	return files[0]
}
