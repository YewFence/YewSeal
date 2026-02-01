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

// UnknownProviderError indicates an unknown sync provider.
type UnknownProviderError struct {
	Name      string
	Supported []string
}

func (e *UnknownProviderError) Error() string {
	if len(e.Supported) == 0 {
		return fmt.Sprintf("unknown provider: %s", e.Name)
	}
	return fmt.Sprintf("unknown provider: %s (supported: %s)", e.Name, strings.Join(e.Supported, ", "))
}

// KeyFileNotFoundError indicates a missing key file with a user hint.
type KeyFileNotFoundError struct {
	Path string
}

func (e *KeyFileNotFoundError) Error() string {
	return fmt.Sprintf("key file not found: %s\nRun 'yews init' first to generate keys", e.Path)
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

// MissingDependencyError indicates a missing external tool.
type MissingDependencyError struct {
	Name        string
	InstallHint string
}

func (e *MissingDependencyError) Error() string {
	if strings.TrimSpace(e.InstallHint) == "" {
		return fmt.Sprintf("%s not found", e.Name)
	}
	return fmt.Sprintf("%s not found\nInstall: %s", e.Name, e.InstallHint)
}

// MissingProjectConfigError indicates a missing project config file required by an integration.
type MissingProjectConfigError struct {
	Path string
	Hint string
}

func (e *MissingProjectConfigError) Error() string {
	if strings.TrimSpace(e.Hint) == "" {
		return fmt.Sprintf("%s not found", e.Path)
	}
	return fmt.Sprintf("%s not found\n%s", e.Path, e.Hint)
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

// PublicKeyNotFoundError indicates no public key could be resolved.
// Keep the message stable because it is user-facing guidance.
type PublicKeyNotFoundError struct {
	KeyFile string
	Cause   error
	Tried   []string
}

func (e *PublicKeyNotFoundError) Unwrap() error {
	return e.Cause
}

func (e *PublicKeyNotFoundError) Error() string {
	tried := strings.Join(e.Tried, ", ")
	if tried == "" {
		tried = "CLI parameter, SOPS_AGE_RECIPIENTS env, .yewseal.toml config, and extracting from private key file"
	}

	if e.Cause != nil {
		return fmt.Sprintf("no public key found. Tried: %s, and extracting from %s (failed: %v). Please run 'yews init' or provide --public-key", tried, e.KeyFile, e.Cause)
	}
	return fmt.Sprintf("no public key found. Tried: %s, and extracting from %s (no valid public key found). Please run 'yews init' or provide --public-key", tried, e.KeyFile)
}
