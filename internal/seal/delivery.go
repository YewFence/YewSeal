package seal

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func writeDeliveredFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("failed to create delivery directory: %w", err)
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("failed to create delivery file: %w", err)
	}
	_, writeErr := file.Write(data)
	if err := errors.Join(writeErr, file.Chmod(0600), file.Close()); err != nil {
		return fmt.Errorf("failed to write delivery file: %w", err)
	}
	return nil
}
