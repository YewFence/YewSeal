package app

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/YewFence/YewSeal/internal/agekey"
	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/errx"
	"github.com/YewFence/YewSeal/internal/presentation"
	"github.com/YewFence/YewSeal/internal/seal"
)

type EditRequest struct {
	Config       *config.Config
	File         string
	KeyFile      string
	Presentation *presentation.Output
}

func EditEncryptedFile(req EditRequest) (err error) {
	out := presentation.OrDiscard(req.Presentation)
	defer func() { err = out.Finish(err) }()

	cfg := req.Config
	if cfg == nil {
		return fmt.Errorf("edit requires a loaded YewSeal configuration")
	}
	out.SetDirectory(config.CurrentDir(cfg))
	if strings.TrimSpace(req.File) == "" {
		return fmt.Errorf("edit requires exactly one configured target")
	}
	selection, err := config.ResolveSelection(cfg, config.SelectionOptions{
		Command:             "decrypt",
		Targets:             []string{req.File},
		RequireSingleTarget: true,
		AllowEmptyTarget:    false,
		StrictRecipients:    true,
	})
	if err != nil {
		return err
	}
	resolved := selection.FilePairs[0]

	if _, err := os.Stat(resolved.EncryptedPath); os.IsNotExist(err) {
		return &errx.NotFoundError{What: "file", Path: resolved.EncryptedPath}
	}

	identityBundle, err := agekey.GetIdentityBundle(req.KeyFile)
	if err != nil {
		return err
	}

	plainData, err := seal.DecryptToBytes(seal.DecryptBytesOptions{
		InputFile:      resolved.EncryptedPath,
		OutputFile:     resolved.PlaintextPath,
		IdentityBundle: identityBundle,
		FormatOverride: resolved.Format,
	})
	if err != nil {
		return err
	}

	tmpFile, err := os.CreateTemp("", "yews-edit-*."+resolved.Format)
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
	editorCmd := resolveEditor()

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
		out.Edited(resolved.EncryptedPath, false)
		return nil
	}

	newEncData, err := seal.EncryptToBytes(editedData, seal.EncryptBytesOptions{
		FormatFile:     resolved.PlaintextPath,
		FormatOverride: resolved.Format,
		Recipients:     resolved.Recipients,
	})
	if err != nil {
		return err
	}

	if err := os.WriteFile(resolved.EncryptedPath, newEncData, 0644); err != nil {
		return fmt.Errorf("failed to write encrypted file: %w", err)
	}

	out.Edited(resolved.EncryptedPath, true)
	return nil
}
