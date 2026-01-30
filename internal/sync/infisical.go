package sync

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/YewFence/YewSeal/internal/tools"
)

// InfisicalProvider 实现 Infisical 的同步逻辑
type InfisicalProvider struct{}

func (p *InfisicalProvider) Name() string {
	return "infisical"
}

// Check 检查 Infisical 是否可用
func (p *InfisicalProvider) Check() error {
	// 检查 CLI 是否安装
	if _, err := exec.LookPath("infisical"); err != nil {
		return fmt.Errorf("infisical CLI not found\nInstall: https://infisical.com/docs/cli/overview")
	}

	// 检查 .infisical.json 是否存在
	if _, err := os.Stat(".infisical.json"); os.IsNotExist(err) {
		return fmt.Errorf(".infisical.json not found\nRun 'infisical init' first to initialize the project")
	}

	return nil
}

// Sync 将密钥同步到 Infisical
func (p *InfisicalProvider) Sync(keyFile, secretName, path string) error {
	// 构建 secret 设置参数: SECRET_NAME=@filepath
	secretArg := fmt.Sprintf("%s=@%s", secretName, keyFile)

	args := []string{"secrets", "set", secretArg}

	// 如果指定了路径，添加 --path 参数
	if path != "" {
		args = append(args, "--path="+path)
	}

	fmt.Printf("🔄 Syncing %s to Infisical as %s...\n", keyFile, secretName)

	stdout, stderr, err := tools.ExecCommand("infisical", args...)
	if err != nil {
		return fmt.Errorf("failed to sync to Infisical: %w\n%s", err, stderr)
	}

	if stdout != "" {
		fmt.Print(stdout)
	}

	fmt.Printf("✅ Successfully synced %s to Infisical\n", secretName)
	return nil
}
