package errx

import (
	"fmt"
	"strings"
)

// ExitCoder 由携带进程退出码的错误实现，main 依据它决定 os.Exit 的值。
type ExitCoder interface {
	error
	ExitCode() int
}

// UsageError 标记调用类错误：参数校验、配置加载、目标选择、身份源解析等
// 在文件处理开始前即可确定的失败。退出码 2；未标记的业务错误退出码 1。
type UsageError struct {
	Err error
}

func (e *UsageError) Error() string { return e.Err.Error() }
func (e *UsageError) Unwrap() error { return e.Err }
func (e *UsageError) ExitCode() int { return 2 }

// Usage 包装调用类错误；nil 原样返回以便在返回语句中直接使用。
func Usage(err error) error {
	if err == nil {
		return nil
	}
	return &UsageError{Err: err}
}

// NotFoundError represents a missing file/resource.
// Keep Error() messages stable because they are user-facing in the CLI.
type NotFoundError struct {
	What string
	Path string
}

func (e *NotFoundError) Error() string {
	what := strings.TrimSpace(e.What)
	if what == "" {
		what = "file"
	}
	return fmt.Sprintf("%s %s does not exist", what, e.Path)
}

// UnsupportedFormatError indicates an unsupported file format.
type UnsupportedFormatError struct {
	Path      string
	Supported []string
}

func (e *UnsupportedFormatError) Error() string {
	if len(e.Supported) == 0 {
		return fmt.Sprintf("unsupported file format for %s", e.Path)
	}
	return fmt.Sprintf("unsupported file format for %s (supported: %s)", e.Path, strings.Join(e.Supported, ", "))
}

// ProtectedOverwriteError indicates that decrypt would overwrite local changes.
type ProtectedOverwriteError struct {
	SourceFile string
	TargetFile string
}

func (e *ProtectedOverwriteError) Error() string {
	return fmt.Sprintf(
		"refusing to overwrite %s because it differs from decrypted %s\nRerun decrypt with --force/-f to overwrite",
		e.TargetFile,
		e.SourceFile,
	)
}

// ExternalCommandError wraps an external command failure while retaining stderr.
type ExternalCommandError struct {
	Op     string
	Cmd    string
	Args   []string
	Stderr string
	Err    error
}

func (e *ExternalCommandError) Unwrap() error {
	return e.Err
}

func (e *ExternalCommandError) Error() string {
	op := strings.TrimSpace(e.Op)
	if op == "" {
		op = "run command"
	}

	// Preserve existing "...: %w\n%s" style.
	if strings.TrimSpace(e.Stderr) != "" {
		return fmt.Sprintf("%s: %v\n%s", op, e.Err, strings.TrimRight(e.Stderr, "\n"))
	}
	return fmt.Sprintf("%s: %v", op, e.Err)
}
