package sync

import (
	"os"

	"github.com/YewFence/YewSeal/internal/errx"
)

// Provider 定义密钥同步提供者的接口
type Provider interface {
	// Name 返回提供者名称
	Name() string
	// Check 检查提供者是否可用（CLI 安装、配置文件等）
	Check() error
	// Sync 执行同步操作
	Sync(keyFile, secretName, path, environment string) error
	// Pull 从提供者拉取密钥
	Pull(keyFile, secretName, path, environment string) error
}

// GetProvider 根据名称获取对应的 Provider
func GetProvider(name string) (Provider, error) {
	switch name {
	case "infisical":
		return &InfisicalProvider{}, nil
	default:
		return nil, &errx.UnknownProviderError{Name: name, Supported: []string{"infisical"}}
	}
}

// SyncKey 执行密钥同步的通用入口
func SyncKey(providerName, keyFile, secretName, path, environment string) error {
	// 通用检查：密钥文件是否存在
	if _, err := os.Stat(keyFile); os.IsNotExist(err) {
		return &errx.KeyFileNotFoundError{Path: keyFile}
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
	return provider.Sync(keyFile, secretName, path, environment)
}

// PullKey 执行密钥拉取的通用入口
func PullKey(providerName, keyFile, secretName, path, environment string) error {
	// 获取 Provider
	provider, err := GetProvider(providerName)
	if err != nil {
		return err
	}

	// Provider 专属检查
	if err := provider.Check(); err != nil {
		return err
	}

	// 执行拉取
	return provider.Pull(keyFile, secretName, path, environment)
}
