package clean

import (
	"bytes"
	"fmt"
	"io/fs"
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
	RemoveDifferent  bool
	SkipDifferent    bool
	ConfirmDifferent func() (bool, error)
}

// Inspect 是轻量预检：解析 plaintext 逻辑路径的完整 symlink 链，
// 只判断最终普通文件目标是否存在，不读取内容。
func Inspect(logicalPath string) (bool, error) {
	_, exists, err := resolveTarget(logicalPath)
	return exists, err
}

// Process 运行单文件清理状态机；Force 只校验删除目标，其他模式
// 还会解密、比较并按 difference 策略决策。
func Process(logicalPath string, opts Options) (Outcome, error) {
	target, exists, err := resolveTarget(logicalPath)
	if err != nil {
		return "", err
	}
	if !exists {
		return AlreadyAbsent, nil
	}
	if opts.Force {
		if err := removeUnverified(logicalPath, target); err != nil {
			return "", err
		}
		return Removed, nil
	}

	snapshot, err := os.ReadFile(target.path)
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

	verifyCiphertext := false
	if !bytes.Equal(snapshot, decrypted) {
		if opts.SkipDifferent {
			return Retained, nil
		}
		if !opts.RemoveDifferent {
			confirmed, err := opts.ConfirmDifferent()
			if err != nil {
				return "", err
			}
			if !confirmed {
				return Retained, nil
			}
		}
		verifyCiphertext = true
	}

	if verifyCiphertext {
		currentDecrypted, err := seal.DecryptToBytes(seal.DecryptBytesOptions{
			InputFile:      opts.EncryptedPath,
			OutputFile:     logicalPath,
			IdentityBundle: opts.IdentityBundle,
			FormatOverride: opts.Format,
		})
		if err != nil {
			return "", fmt.Errorf("ciphertext changed before removal: %w", err)
		}
		if !bytes.Equal(currentDecrypted, decrypted) {
			return "", fmt.Errorf("ciphertext changed before removal: decrypted content changed")
		}
	}

	if err := removeVerified(logicalPath, target.path, snapshot); err != nil {
		return "", err
	}
	return Removed, nil
}

type resolvedTarget struct {
	path string
	info fs.FileInfo
}

// resolveTarget 解析逻辑路径的完整 symlink 链。断链和缺失目标计为
// 不存在；其他解析错误或非普通文件最终目标返回错误。
func resolveTarget(logicalPath string) (resolvedTarget, bool, error) {
	resolved, err := filepath.EvalSymlinks(logicalPath)
	if err != nil {
		if os.IsNotExist(err) {
			return resolvedTarget{}, false, nil
		}
		return resolvedTarget{}, false, fmt.Errorf("failed to resolve plaintext path: %w", err)
	}
	targetPath, err := filepath.Abs(resolved)
	if err != nil {
		return resolvedTarget{}, false, fmt.Errorf("failed to resolve plaintext target: %w", err)
	}
	info, err := os.Stat(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return resolvedTarget{}, false, nil
		}
		return resolvedTarget{}, false, fmt.Errorf("failed to inspect plaintext target: %w", err)
	}
	if !info.Mode().IsRegular() {
		return resolvedTarget{}, false, fmt.Errorf("plaintext target is not a regular file")
	}
	return resolvedTarget{path: filepath.Clean(targetPath), info: info}, true, nil
}

// removeVerified 在删除前重新解析同一逻辑路径，确认链接链仍指向
// 先前验证的普通文件且内容与决策时的 snapshot 完全一致，再执行
// 单文件 Remove。logical path 中的 symlink 保持不变。
func removeVerified(logicalPath, targetPath string, snapshot []byte) error {
	current, exists, err := resolveTarget(logicalPath)
	if err != nil {
		return fmt.Errorf("plaintext changed before removal: %w", err)
	}
	if !exists || current.path != targetPath {
		return fmt.Errorf("plaintext changed before removal: symlink target changed")
	}
	currentData, err := os.ReadFile(current.path)
	if err != nil {
		return fmt.Errorf("plaintext changed before removal: failed to read target: %w", err)
	}
	if !bytes.Equal(currentData, snapshot) {
		return fmt.Errorf("plaintext changed before removal: content changed")
	}
	if err := os.Remove(current.path); err != nil {
		return fmt.Errorf("failed to remove plaintext file: %w", err)
	}
	return nil
}

func removeUnverified(logicalPath string, initial resolvedTarget) error {
	current, exists, err := resolveTarget(logicalPath)
	if err != nil {
		return fmt.Errorf("plaintext changed before removal: %w", err)
	}
	if !exists || current.path != initial.path || !os.SameFile(initial.info, current.info) {
		return fmt.Errorf("plaintext changed before removal: file target changed")
	}
	if err := os.Remove(current.path); err != nil {
		return fmt.Errorf("failed to remove plaintext file: %w", err)
	}
	return nil
}
