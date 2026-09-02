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

// SavePublicKeyToConfig creates or overwrites .yewseal.toml with canonical config content.
func SavePublicKeyToConfig(publicKey string, filePairs []config.FilePair) error {
	const configPath = ".yewseal.toml"

	cfg := config.Config{
		Encryption: config.EncryptionConfig{
			Files: filePairs,
		},
		Key: config.KeyConfig{
			FilePath: ".age/keys.txt",
		},
		Recipients: config.RecipientConfig{
			Defaults: stringSlicePtr([]string{"owner"}),
			Registry: map[string]string{"owner": publicKey},
		},
	}

	var buffer bytes.Buffer
	buffer.WriteString(`# YewSeal 配置文件
#
# 配置优先级：
# 1. 命令行参数（最高优先级）
# 2. 环境变量
# 3. 此配置文件
# 4. 默认值（最低优先级）
#
# 所有敏感文件都统一写在 [[encryption.files]] 里：
# - plaintext 表示明文文件
# - encrypted 表示加密文件

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

	fmt.Println("✅ Wrote .yewseal.toml")
	return nil
}
