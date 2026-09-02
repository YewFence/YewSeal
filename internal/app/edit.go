package app

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/errx"
	"github.com/YewFence/YewSeal/internal/seal"
	"github.com/google/shlex"
)

type EditRequest struct {
	Config  *config.Config
	File    string
	Editor  string
	KeyFile string
	Output  io.Writer
}

func EditEncryptedFile(req EditRequest) error {
	out := outputWriter(req.Output)

	cfg := req.Config
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	target, err := ResolveEncryptedTarget(cfg, req.File, "edit")
	if err != nil {
		return err
	}

	if _, err := os.Stat(target.EncryptedPath); os.IsNotExist(err) {
		return &errx.NotFoundError{What: "file", Path: target.EncryptedPath}
	}

	plainData, err := seal.DecryptToBytes(seal.DecryptBytesOptions{
		InputFile:      target.EncryptedPath,
		OutputFile:     target.PlaintextPath,
		KeyFile:        req.KeyFile,
		FormatOverride: target.FormatOverride,
		Output:         req.Output,
	})
	if err != nil {
		return err
	}

	tmpFile, err := os.CreateTemp("", "yews-edit-*."+target.Format)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	if _, err := tmpFile.Write(plainData); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	originalHash := sha256.Sum256(plainData)
	editorCmd := resolveEditor(req.Editor)

	_, _ = fmt.Fprintf(out, "✏️  Opening %s in %s...\n", target.EncryptedPath, editorCmd)

	parts, err := splitEditorCommand(editorCmd)
	if err != nil {
		return err
	}
	args := append(parts[1:], tmpPath)
	cmd := exec.Command(parts[0], args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("editor exited with error: %w", err)
	}

	editedData, err := os.ReadFile(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to read edited file: %w", err)
	}

	editedHash := sha256.Sum256(editedData)
	if originalHash == editedHash {
		_, _ = fmt.Fprintln(out, "⏭️  No changes detected, skipping re-encryption")
		return nil
	}

	recipients, err := seal.ExtractAgeRecipientsFromEncryptedFile(target.EncryptedPath, target.PlaintextPath, target.FormatOverride)
	if err != nil {
		return err
	}
	newEncData, err := seal.EncryptToBytes(editedData, seal.EncryptBytesOptions{
		FormatFile:     target.PlaintextPath,
		FormatOverride: target.FormatOverride,
		Recipients:     recipients,
		Output:         req.Output,
	})
	if err != nil {
		return err
	}

	if err := os.WriteFile(target.EncryptedPath, newEncData, 0644); err != nil {
		return fmt.Errorf("failed to write encrypted file: %w", err)
	}

	_, _ = fmt.Fprintln(out, "✅ File edited and re-encrypted successfully")
	return nil
}

func resolveEditor(editor string) string {
	if editor != "" {
		return editor
	}
	if e := os.Getenv("EDITOR"); e != "" {
		return e
	}
	if e := os.Getenv("VISUAL"); e != "" {
		return e
	}
	if runtime.GOOS == "windows" {
		for _, candidate := range []string{"code -w", "notepad"} {
			parts, err := splitEditorCommand(candidate)
			if err != nil || len(parts) == 0 {
				continue
			}
			name := parts[0]
			if p, _ := exec.LookPath(name); p != "" {
				return candidate
			}
		}
		return "notepad"
	}
	return "vi"
}

func splitEditorCommand(editorCmd string) ([]string, error) {
	parts, err := shlex.Split(editorCmd)
	if err != nil {
		return nil, fmt.Errorf("failed to parse editor command: %w", err)
	}
	if len(parts) == 0 {
		return nil, fmt.Errorf("editor command is empty")
	}
	return parts, nil
}

func outputWriter(w io.Writer) io.Writer {
	if w != nil {
		return w
	}
	return os.Stdout
}
