package app

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/YewFence/YewSeal/internal/errx"
	"github.com/YewFence/YewSeal/internal/seal"
	"github.com/google/shlex"
)

type EditRequest struct {
	File     string
	Editor   string
	KeyFile  string
	Output   io.Writer
	Warnings io.Writer
}

func EditEncryptedFile(req EditRequest) error {
	out := outputWriter(req.Output)

	if _, err := os.Stat(req.File); os.IsNotExist(err) {
		return &errx.NotFoundError{What: "file", Path: req.File}
	}

	editPlaintextPath := plaintextFormatPathForEdit(req.File)
	formatOverride, err := editFormatOverride(editPlaintextPath)
	if err != nil {
		return err
	}

	plainData, err := seal.DecryptToBytes(seal.DecryptBytesOptions{
		InputFile:      req.File,
		OutputFile:     editPlaintextPath,
		KeyFile:        req.KeyFile,
		FormatOverride: formatOverride,
		Output:         req.Output,
		Warnings:       req.Warnings,
	})
	if err != nil {
		return err
	}

	tmpFile, err := os.CreateTemp("", "yews-edit-*."+formatOverride)
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

	_, _ = fmt.Fprintf(out, "✏️  Opening %s in %s...\n", req.File, editorCmd)

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

	publicKey, err := seal.ExtractAgeRecipientFromEncryptedFile(req.File, editPlaintextPath, formatOverride)
	if err != nil {
		return err
	}

	newEncData, err := seal.EncryptToBytes(editedData, seal.EncryptBytesOptions{
		FormatFile:     editPlaintextPath,
		FormatOverride: formatOverride,
		PublicKey:      publicKey,
		Output:         req.Output,
		Warnings:       req.Warnings,
	})
	if err != nil {
		return err
	}

	if err := os.WriteFile(req.File, newEncData, 0644); err != nil {
		return fmt.Errorf("failed to write encrypted file: %w", err)
	}

	_, _ = fmt.Fprintln(out, "✅ File edited and re-encrypted successfully")
	return nil
}

func plaintextFormatPathForEdit(encryptedPath string) string {
	dir := filepath.Dir(encryptedPath)
	base := filepath.Base(encryptedPath)
	lowerBase := strings.ToLower(base)
	suffixes := []struct {
		encrypted string
		plain     string
	}{
		{".enc.toml.yaml", ".toml"},
		{".enc.toml.yml", ".toml"},
		{".enc.toml", ".toml"},
		{".enc.yaml", ".yaml"},
		{".enc.yml", ".yml"},
		{".enc.json", ".json"},
		{".enc.env", ".env"},
		{".enc.ini", ".ini"},
		{".enc.bin", ".bin"},
		{".enc.binary", ".binary"},
	}

	for _, suffix := range suffixes {
		if strings.HasSuffix(lowerBase, suffix.encrypted) {
			stem := base[:len(base)-len(suffix.encrypted)]
			return filepath.Join(dir, stem+suffix.plain)
		}
	}
	return encryptedPath
}

func editFormatOverride(editPlaintextPath string) (string, error) {
	format, ok := seal.NormalizeFormatForPath(editPlaintextPath)
	if !ok {
		return "", fmt.Errorf("could not detect format for %s (supported: toml, yaml, json, env, ini, binary)", editPlaintextPath)
	}
	return format, nil
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
