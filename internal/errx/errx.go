package errx

import (
	"fmt"
	"strings"
)

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

// KeyFileReadError indicates a key file could not be read.
type KeyFileReadError struct {
	Path string
	Err  error
}

func (e *KeyFileReadError) Unwrap() error {
	return e.Err
}

func (e *KeyFileReadError) Error() string {
	return fmt.Sprintf("failed to read key file %s: %v", e.Path, e.Err)
}

// AgeSecretKeyNotFoundError indicates a file did not contain a valid Age secret key.
type AgeSecretKeyNotFoundError struct {
	Path string
}

func (e *AgeSecretKeyNotFoundError) Error() string {
	return fmt.Sprintf("no valid Age secret key found in %s", e.Path)
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

// AgeKeyNotFoundError indicates no Age private key could be located.
type AgeKeyNotFoundError struct {
	Options []string
}

func (e *AgeKeyNotFoundError) Error() string {
	if len(e.Options) == 0 {
		return "no Age key found"
	}
	return fmt.Sprintf("no Age key found. Options: %s", strings.Join(e.Options, ", "))
}
