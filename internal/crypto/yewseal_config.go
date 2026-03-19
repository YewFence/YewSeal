package crypto

import (
	"bytes"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/YewFence/YewSeal/internal/config"
)

// SavePublicKeyToConfig creates or overwrites .yewseal.toml with canonical config content.
func SavePublicKeyToConfig(publicKey string, filePairs []config.FilePair) error {
	const configPath = ".yewseal.toml"

	cfg := config.Config{
		Encryption: config.EncryptionConfig{
			Files: filePairs,
		},
		Key: config.KeyConfig{
			FilePath:  ".age/keys.txt",
			PublicKey: publicKey,
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
# - dec 表示明文文件
# - enc 表示加密文件

`)

	if err := toml.NewEncoder(&buffer).Encode(cfg); err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}

	if err := os.WriteFile(configPath, buffer.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	fmt.Println("✅ Wrote .yewseal.toml")
	return nil
}
