package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	Sync       SyncConfig       `toml:"sync"`
}

// EncryptionConfig defines encrypted file mappings.
type EncryptionConfig struct {
	Files []FilePair  `toml:"files"`
	Group GroupConfig `toml:"group"`
}

// FilePair defines one plaintext/encrypted file mapping.
type FilePair struct {
	// PlaintextPath is the plaintext file path used by encrypt input and decrypt output.
	PlaintextPath string `toml:"plaintext"`
	// EncryptedPath is the encrypted file path used by encrypt output and decrypt input.
	EncryptedPath string `toml:"encrypted"`
	// Format overrides the file format detection (toml/yaml/json/env/ini/binary).
	// Useful for files with non-standard extensions like .dev.vars.
	Format string `toml:"format,omitempty"`
}

type GroupConfig struct {
	Patterns        []string `toml:"patterns"`
	FormatRules     []string `toml:"format_rules"`
	UnknownAsBinary bool     `toml:"unknown_as_binary"`
}

// KeyConfig defines key file location.
type KeyConfig struct {
	// FilePath is the path to Age private key file.
	// Do NOT store the actual key value here to avoid leaking secrets.
	FilePath string `toml:"file_path"`
	// PublicKey is the Age public key for encryption (safe to commit).
	PublicKey string `toml:"public_key"`
}

// SyncConfig defines Age key synchronization settings.
type SyncConfig struct {
	// Provider is the secret management provider name.
	Provider string `toml:"provider,omitempty"`
	// ProjectID is the provider project identifier used by sync commands.
	ProjectID string `toml:"project_id,omitempty"`
	// SecretName is the remote secret name for the Age key file.
	SecretName string `toml:"secret_name,omitempty"`
	// Path is the remote path/folder in the provider.
	Path string `toml:"path,omitempty"`
	// Environment is the remote environment name in the provider.
	Environment string `toml:"environment,omitempty"`
}

// DefaultFilePair returns the default plaintext/encrypted mapping.
func DefaultFilePair() FilePair {
	return FilePair{
		PlaintextPath: defaultDecryptedFile,
		EncryptedPath: defaultEncryptedFile,
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
		return nil, fmt.Errorf("failed to get working directory: %w", err)
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
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to stat config file %s: %w", path, err)
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
	for i, filePair := range config.Encryption.Files {
		if strings.TrimSpace(filePair.PlaintextPath) == "" {
			return nil, fmt.Errorf("invalid encryption.files[%d]: plaintext is required", i)
		}
		if strings.TrimSpace(filePair.EncryptedPath) == "" {
			return nil, fmt.Errorf("invalid encryption.files[%d]: encrypted is required", i)
		}
	}
	for i, rule := range config.Encryption.Group.FormatRules {
		if strings.TrimSpace(rule) == "" {
			return nil, fmt.Errorf("invalid encryption.group.format_rules[%d]: rule is empty", i)
		}
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

// GetSyncProvider returns the key sync provider.
// Priority: provided value > config file > default.
func (c *Config) GetSyncProvider(provided string) string {
	if provided != "" {
		return provided
	}
	if c.Sync.Provider != "" {
		return c.Sync.Provider
	}
	return "infisical"
}

// GetSyncSecretName returns the key sync secret name.
// Priority: provided value > config file > default.
func (c *Config) GetSyncSecretName(provided string) string {
	if provided != "" {
		return provided
	}
	if c.Sync.SecretName != "" {
		return c.Sync.SecretName
	}
	return "AGE_KEY_FILE"
}

// GetSyncProjectID returns the key sync project identifier.
// Priority: provided value > config file > default.
func (c *Config) GetSyncProjectID(provided string) string {
	if provided != "" {
		return provided
	}
	return c.Sync.ProjectID
}

// GetSyncPath returns the key sync remote path.
// Priority: provided value > config file > default.
func (c *Config) GetSyncPath(provided string) string {
	if provided != "" {
		return provided
	}
	return c.Sync.Path
}

// GetSyncEnvironment returns the key sync remote environment.
// Priority: provided value > config file > default.
func (c *Config) GetSyncEnvironment(provided string) string {
	if provided != "" {
		return provided
	}
	return c.Sync.Environment
}

// GetFiles returns configured file mappings.
func (c *Config) GetFiles() []FilePair {
	if len(c.Encryption.Files) == 0 {
		return []FilePair{DefaultFilePair()}
	}
	files := make([]FilePair, len(c.Encryption.Files))
	copy(files, c.Encryption.Files)
	return files
}

func (c *Config) GetGroup() GroupConfig {
	return GroupConfig{
		Patterns:        append([]string(nil), c.Encryption.Group.Patterns...),
		FormatRules:     append([]string(nil), c.Encryption.Group.FormatRules...),
		UnknownAsBinary: c.Encryption.Group.UnknownAsBinary,
	}
}

// GetPrimaryFilePair returns the first configured file mapping.
func (c *Config) GetPrimaryFilePair() FilePair {
	files := c.GetFiles()
	return files[0]
}
