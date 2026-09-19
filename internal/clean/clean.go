package clean

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/YewFence/YewSeal/internal/agekey"
	"github.com/YewFence/YewSeal/internal/seal"
)

type Outcome string

const (
	Removed       Outcome = "removed"
	Retained      Outcome = "retained"
	AlreadyAbsent Outcome = "already-absent"
)

type Options struct {
	EncryptedPath    string
	Format           string
	IdentityBundle   agekey.IdentityBundle
	Force            bool
	SkipDifferent    bool
	ConfirmDifferent func() (bool, error)
}

// Inspect 是轻量预检：解析 plaintext 逻辑路径的完整 symlink 链，
// 只判断最终普通文件目标是否存在，不读取内容。
func Inspect(logicalPath string) (bool, error) {
	_, exists, err := resolveTarget(logicalPath)
	return exists, err
}

// CheckCiphertext 判断密文是否存在且可访问，在加载 identity 前使用。
func CheckCiphertext(path string) error {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return fmt.Errorf("encrypted file is missing")
	}
	if err != nil {
		return fmt.Errorf("failed to inspect encrypted file: %w", err)
	}
	return nil
}

// Process 运行单文件状态机：读取 snapshot、解密、字节比较、
// 按 difference 策略决策，然后复核并删除最终普通文件目标。
func Process(logicalPath string, opts Options) (Outcome, error) {
	targetPath, exists, err := resolveTarget(logicalPath)
	if err != nil {
		return "", err
	}
	if !exists {
		return AlreadyAbsent, nil
	}
	snapshot, err := os.ReadFile(targetPath)
	if err != nil {
		return "", fmt.Errorf("failed to read plaintext file: %w", err)
	}

	decrypted, err := seal.DecryptToBytes(seal.DecryptBytesOptions{
		InputFile:      opts.EncryptedPath,
		OutputFile:     logicalPath,
		IdentityBundle: opts.IdentityBundle,
		FormatOverride: opts.Format,
	})
	if err != nil {
		return "", err
	}

	if !bytes.Equal(snapshot, decrypted) && !opts.Force {
		if opts.SkipDifferent {
			return Retained, nil
		}
		confirmed, err := opts.ConfirmDifferent()
		if err != nil {
			return "", err
		}
		if !confirmed {
			return Retained, nil
		}
	}

	if err := removeVerified(logicalPath, targetPath, snapshot); err != nil {
		return "", err
	}
	return Removed, nil
}

// resolveTarget 解析逻辑路径的完整 symlink 链。断链和缺失目标计为
// 不存在；其他解析错误或非普通文件最终目标返回错误。
func resolveTarget(logicalPath string) (string, bool, error) {
	resolved, err := filepath.EvalSymlinks(logicalPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("failed to resolve plaintext path: %w", err)
	}
	targetPath, err := filepath.Abs(resolved)
	if err != nil {
		return "", false, fmt.Errorf("failed to resolve plaintext target: %w", err)
	}
	info, err := os.Stat(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("failed to inspect plaintext target: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", false, fmt.Errorf("plaintext target is not a regular file")
	}
	return filepath.Clean(targetPath), true, nil
}

// removeVerified 在删除前重新解析同一逻辑路径，确认链接链仍指向
// 先前验证的普通文件且内容与决策时的 snapshot 完全一致，再执行
// 单文件 Remove。logical path 中的 symlink 保持不变。
func removeVerified(logicalPath, targetPath string, snapshot []byte) error {
	currentPath, exists, err := resolveTarget(logicalPath)
	if err != nil {
		return fmt.Errorf("plaintext changed before removal: %w", err)
	}
	if !exists || currentPath != targetPath {
		return fmt.Errorf("plaintext changed before removal: symlink target changed")
	}
	current, err := os.ReadFile(currentPath)
	if err != nil {
		return fmt.Errorf("plaintext changed before removal: failed to read target: %w", err)
	}
	if !bytes.Equal(current, snapshot) {
		return fmt.Errorf("plaintext changed before removal: content changed")
	}
	if err := os.Remove(currentPath); err != nil {
		return fmt.Errorf("failed to remove plaintext file: %w", err)
	}
	return nil
}
