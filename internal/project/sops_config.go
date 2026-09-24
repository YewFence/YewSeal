package project

import (
	"fmt"
	"os"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/sopsconfig"
)

const sopsYamlPath = ".sops.yaml"

// SyncResolvedSopsYaml rewrites .sops.yaml from resolved per-file policy.
func SyncResolvedSopsYaml(filePairs []config.ResolvedFilePair) error {
	data, err := sopsconfig.Render(filePairs)
	if err != nil {
		return err
	}
	return writeSopsConfig(data)
}

func writeSopsConfig(data []byte) error {
	tempFile, err := os.CreateTemp(".", ".sops.yaml.*")
	if err != nil {
		return fmt.Errorf("failed to create temporary .sops.yaml: %w", err)
	}
	tempPath := tempFile.Name()
	defer func() { _ = os.Remove(tempPath) }()
	if _, err := tempFile.Write(data); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("failed to write temporary .sops.yaml: %w", err)
	}
	if err := tempFile.Chmod(0644); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("failed to set temporary .sops.yaml permissions: %w", err)
	}
	if err := tempFile.Sync(); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("failed to sync temporary .sops.yaml: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary .sops.yaml: %w", err)
	}
	if err := os.Rename(tempPath, sopsYamlPath); err != nil {
		return fmt.Errorf("failed to replace .sops.yaml: %w", err)
	}

	return nil
}
