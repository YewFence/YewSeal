package project

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/YewFence/YewSeal/internal/config"
	toml "github.com/pelletier/go-toml/v2"
)

func stringSlicePtr(values []string) *[]string {
	return &values
}

// SaveBootstrapConfig creates or overwrites .yewseal.toml with the owner policy.
func SaveBootstrapConfig(ownerRecipient string, filePairs []config.FilePair) error {
	const configPath = ".yewseal.toml"

	cfg := config.Config{
		Encryption: config.EncryptionConfig{
			Files: filePairs,
		},
		Recipients: config.RecipientConfig{
			Defaults: stringSlicePtr([]string{"owner"}),
			Registry: map[string]string{"owner": ownerRecipient},
		},
	}

	var buffer bytes.Buffer
	buffer.WriteString(`# YewSeal configuration file
#
# Configuration precedence:
# 1. Command-line arguments (highest)
# 2. Environment variables
# 3. This configuration file
# 4. Built-in defaults (lowest)
#
# Every sensitive file is declared in [[encryption.files]]:
# - plaintext is the cleartext file
# - encrypted is the encrypted file

`)

	encoder := toml.NewEncoder(&buffer)
	if err := encoder.Encode(cfg); err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}

	tempFile, err := os.CreateTemp(filepath.Dir(configPath), ".yewseal.toml.*")
	if err != nil {
		return fmt.Errorf("failed to create temp config: %w", err)
	}
	tempPath := tempFile.Name()
	defer func() {
		_ = os.Remove(tempPath)
	}()

	if _, err := tempFile.Write(buffer.Bytes()); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("failed to write temp config: %w", err)
	}
	if err := tempFile.Chmod(0644); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("failed to set temp config permissions: %w", err)
	}
	if err := tempFile.Sync(); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("failed to sync temp config: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp config: %w", err)
	}

	if err := os.Rename(tempPath, configPath); err != nil {
		if removeErr := os.Remove(configPath); removeErr != nil && !os.IsNotExist(removeErr) {
			return fmt.Errorf("failed to replace config: %w", err)
		}
		if retryErr := os.Rename(tempPath, configPath); retryErr != nil {
			return fmt.Errorf("failed to replace config: %w", retryErr)
		}
	}

	return nil
}
