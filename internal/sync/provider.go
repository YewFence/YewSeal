package sync

import (
	"fmt"
	"os"
)

// Provider 定义密钥同步提供者的接口
type Provider interface {
	// Name 返回提供者名称
	Name() string
	// Check 检查提供者是否可用（CLI 安装、配置文件等）
	Check() error
	// Sync 执行同步操作
	Sync(keyFile, secretName, path string) error
}

// GetProvider 根据名称获取对应的 Provider
func GetProvider(name string) (Provider, error) {
	switch name {
	case "infisical":
		return &InfisicalProvider{}, nil
	default:
		return nil, fmt.Errorf("unknown provider: %s (supported: infisical)", name)
	}
}

// SyncKey 执行密钥同步的通用入口
func SyncKey(providerName, keyFile, secretName, path string) error {
	// 通用检查：密钥文件是否存在
	if _, err := os.Stat(keyFile); os.IsNotExist(err) {
		return fmt.Errorf("key file not found: %s\nRun 'yews init' first to generate keys", keyFile)
	}

	// 获取 Provider
	provider, err := GetProvider(providerName)
	if err != nil {
		return err
	}

	// Provider 专属检查
	if err := provider.Check(); err != nil {
		return err
	}

	// 执行同步
	return provider.Sync(keyFile, secretName, path)
}
