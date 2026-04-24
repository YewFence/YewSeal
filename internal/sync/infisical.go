package sync

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/YewFence/YewSeal/internal/errx"
	tools "github.com/YewFence/YewSeal/internal/execx"
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
		return &errx.MissingDependencyError{Name: "infisical CLI", InstallHint: "https://infisical.com/docs/cli/overview"}
	}

	// 检查 .infisical.json 是否存在
	if _, err := os.Stat(".infisical.json"); os.IsNotExist(err) {
		return &errx.MissingProjectConfigError{Path: ".infisical.json", Hint: "Run 'infisical init' first to initialize the project"}
	}

	return nil
}

func infisicalSyncArgs(keyFile, secretName, path, environment string) []string {
	// 构建 secret 设置参数: SECRET_NAME=@filepath
	secretArg := fmt.Sprintf("%s=@%s", secretName, keyFile)

	args := []string{"secrets", "set", secretArg}

	// 如果指定了路径，添加 --path 参数
	if path != "" {
		args = append(args, "--path="+path)
	}

	if environment != "" {
		args = append(args, "--env="+environment)
	}

	return args
}

func infisicalPullArgs(secretName, path, environment string) []string {
	args := []string{"secrets", "get", secretName, "--plain"}

	// 如果指定了路径，添加 --path 参数
	if path != "" {
		args = append(args, "--path="+path)
	}

	if environment != "" {
		args = append(args, "--env="+environment)
	}

	return args
}

// Sync 将密钥同步到 Infisical
func (p *InfisicalProvider) Sync(keyFile, secretName, path, environment string) error {
	args := infisicalSyncArgs(keyFile, secretName, path, environment)

	fmt.Printf("🔄 Syncing %s to Infisical as %s...\n", keyFile, secretName)

	stdout, stderr, err := tools.ExecCommand("infisical", args...)
	if err != nil {
		return &errx.ExternalCommandError{Op: "failed to sync to Infisical", Cmd: "infisical", Args: args, Stderr: stderr, Err: err}
	}

	if stdout != "" {
		fmt.Print(stdout)
	}

	fmt.Printf("✅ Successfully synced %s to Infisical\n", secretName)
	return nil
}

// Pull 从 Infisical 拉取密钥
func (p *InfisicalProvider) Pull(keyFile, secretName, path, environment string) error {
	args := infisicalPullArgs(secretName, path, environment)

	fmt.Printf("🔄 Pulling %s from Infisical to %s...\n", secretName, keyFile)

	stdout, stderr, err := tools.ExecCommand("infisical", args...)
	if err != nil {
		return &errx.ExternalCommandError{Op: "failed to pull from Infisical", Cmd: "infisical", Args: args, Stderr: stderr, Err: err}
	}

	// 确保目录存在
	dir := filepath.Dir(keyFile)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// 写入文件 (权限 0600)
	if err := os.WriteFile(keyFile, []byte(stdout), 0600); err != nil {
		return fmt.Errorf("failed to write key file %s: %w", keyFile, err)
	}

	fmt.Printf("✅ Successfully pulled %s from Infisical\n", secretName)
	return nil
}
