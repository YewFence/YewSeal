package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config represents the YewSeal configuration
type Config struct {
	// Encryption settings
	Encryption EncryptionConfig `toml:"encryption"`
	// Key settings
	Key KeyConfig `toml:"key"`
}

// EncryptionConfig defines encryption file settings
type EncryptionConfig struct {
	// InputFile is the file to be encrypted (default: wrangler.toml)
	InputFile string `toml:"input_file"`
	// OutputFile is the encrypted output file (default: wrangler.enc.yaml)
	OutputFile string `toml:"output_file"`
	// Files is a list of file pairs for batch encryption/decryption
	Files []FilePair `toml:"files"`
}

// FilePair defines a pair of input and output files
type FilePair struct {
	// Input is the source file (plain text for encryption, encrypted for decryption)
	Input string `toml:"input"`
	// Output is the destination file (encrypted for encryption, plain text for decryption)
	Output string `toml:"output"`
}

// KeyConfig defines key file location
type KeyConfig struct {
	// FilePath is the path to Age private key file
	// Do NOT store the actual key value here to avoid leaking secrets
	FilePath string `toml:"file_path"`
	// PublicKey is the Age public key for encryption (safe to commit)
	PublicKey string `toml:"public_key"`
}

// DefaultConfig returns a config with default values
func DefaultConfig() *Config {
	return &Config{
		Encryption: EncryptionConfig{
			InputFile:  "wrangler.toml",
			OutputFile: "wrangler.enc.yaml",
		},
		Key: KeyConfig{
			FilePath: ".age/keys.txt",
		},
	}
}

// LoadConfig loads configuration from .yewseal.toml
// Searches in the following locations (in order):
// 1. .yewseal/.config/.yewseal.toml
// 2. .yewseal/.yewseal.toml
// 3. .yewseal.toml
// Returns default config if file doesn't exist
func LoadConfig() (*Config, error) {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return DefaultConfig(), nil
	}

	// Try to find config file in multiple locations
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

	// If config file doesn't exist in any location, return default config
	if configPath == "" {
		return DefaultConfig(), nil
	}

	// Read and parse config file
	config := DefaultConfig()
	if _, err := toml.DecodeFile(configPath, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", configPath, err)
	}

	return config, nil
}

// GetEncryptionInput returns the input file for encryption
// Priority: provided value > config file > default
func (c *Config) GetEncryptionInput(provided string) string {
	if provided != "" && provided != "wrangler.toml" {
		return provided
	}
	if c.Encryption.InputFile != "" {
		return c.Encryption.InputFile
	}
	return "wrangler.toml"
}

// GetEncryptionOutput returns the output file for encryption
// Priority: provided value > config file > default
func (c *Config) GetEncryptionOutput(provided string) string {
	if provided != "" && provided != "wrangler.enc.yaml" {
		return provided
	}
	if c.Encryption.OutputFile != "" {
		return c.Encryption.OutputFile
	}
	return "wrangler.enc.yaml"
}

// GetDecryptionInput returns the input file for decryption
// Priority: provided value > config file > default (encrypted file)
func (c *Config) GetDecryptionInput(provided string) string {
	if provided != "" && provided != "wrangler.enc.yaml" {
		return provided
	}
	// For decryption, input is the encrypted output
	if c.Encryption.OutputFile != "" {
		return c.Encryption.OutputFile
	}
	return "wrangler.enc.yaml"
}

// GetDecryptionOutput returns the output file for decryption
// Priority: provided value > config file > default (original file)
func (c *Config) GetDecryptionOutput(provided string) string {
	if provided != "" && provided != "wrangler.toml" {
		return provided
	}
	// For decryption, output is the original input
	if c.Encryption.InputFile != "" {
		return c.Encryption.InputFile
	}
	return "wrangler.toml"
}

// GetKeyFile returns the key file path
// Priority: provided value > config file > default
func (c *Config) GetKeyFile(provided string) string {
	if provided != "" {
		return provided
	}
	if c.Key.FilePath != "" {
		return c.Key.FilePath
	}
	return ".age/key.txt"
}

// GetPublicKey returns the Age public key
func (c *Config) GetPublicKey() string {
	return c.Key.PublicKey
}

// GetFiles returns the list of file pairs for batch operations
// Returns nil if no files are configured
func (c *Config) GetFiles() []FilePair {
	return c.Encryption.Files
}

// HasBatchFiles returns true if batch files are configured
func (c *Config) HasBatchFiles() bool {
	return len(c.Encryption.Files) > 0
}
