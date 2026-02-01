package crypto

import (
	"fmt"
	"os"
	"strings"
)

// SavePublicKeyToConfig creates or updates .yewseal.toml with the public key
func SavePublicKeyToConfig(publicKey, inputFile, outputFile string) error {
	const configPath = ".yewseal.toml"

	// Check if config file already exists
	if _, err := os.Stat(configPath); err == nil {
		// Config exists, read and update it
		content, err := os.ReadFile(configPath)
		if err != nil {
			return fmt.Errorf("failed to read existing config: %w", err)
		}

		// Check if public_key already exists
		if strings.Contains(string(content), "public_key") {
			fmt.Println("⏭️  .yewseal.toml already contains public_key")
			return nil
		}

		// Append public_key to [key] section
		lines := strings.Split(string(content), "\n")
		var newLines []string
		inKeySection := false
		publicKeyAdded := false

		for _, line := range lines {
			newLines = append(newLines, line)

			if strings.TrimSpace(line) == "[key]" {
				inKeySection = true
			} else if strings.HasPrefix(strings.TrimSpace(line), "[") {
				inKeySection = false
			}

			// Add public_key after file_path line in [key] section
			if inKeySection && strings.Contains(line, "file_path") && !publicKeyAdded {
				newLines = append(newLines, fmt.Sprintf(`public_key = "%s"`, publicKey))
				publicKeyAdded = true
			}
		}

		if err := os.WriteFile(configPath, []byte(strings.Join(newLines, "\n")), 0644); err != nil {
			return fmt.Errorf("failed to update config: %w", err)
		}

		fmt.Println("✅ Updated .yewseal.toml with public key")
		return nil
	}

	// Create new config file
	configContent := fmt.Sprintf(`# YewSeal 配置文件
#
# 配置优先级：
# 1. 命令行参数（最高优先级）
# 2. 环境变量
# 3. 此配置文件
# 4. 默认值（最低优先级）

[encryption]
# 需要加密的输入文件（TOML 格式）
input_file = "%s"

# 加密后的输出文件（YAML 格式）
output_file = "%s"

[key]
# Age 私钥文件路径
# 默认值: ".age/keys.txt"
# 重要提示：该配置文件不应该存储密钥值本身！
file_path = ".age/keys.txt"

# Age 公钥（用于加密，可以安全提交到版本控制）
public_key = "%s"
`, inputFile, outputFile, publicKey)

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to create config: %w", err)
	}

	fmt.Println("✅ Created .yewseal.toml with public key")
	return nil
}
