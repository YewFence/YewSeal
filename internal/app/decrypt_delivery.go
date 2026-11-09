package app

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/errx"
)

func prepareDecryptDelivery(cfg *config.Config, selection *config.ResolvedSelection, req DecryptRequest) error {
	if req.OutputDir == "" {
		return nil
	}
	if req.Force || req.Inplace {
		return errx.Usage(fmt.Errorf("--output conflicts with --force=true or --inplace=true"))
	}
	root, err := resolveMirroredTreeRoot(cfg, req.OutputDir, "--output")
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return errx.Usage(fmt.Errorf("failed to inspect --output directory %s: %w", root, err))
	}
	if len(entries) != 0 {
		return errx.Usage(fmt.Errorf("--output directory %s must be completely empty (including hidden entries)", root))
	}
	if err := projectMirroredTree(cfg, selection, root, "--output", "write to"); err != nil {
		return err
	}
	destinations := make(map[string]struct{}, len(selection.FilePairs))
	for _, pair := range selection.FilePairs {
		destinations[pair.PlaintextPath] = struct{}{}
	}
	for destination := range destinations {
		for parent := filepath.Dir(destination); parent != root; parent = filepath.Dir(parent) {
			if _, exists := destinations[parent]; exists {
				return errx.Usage(fmt.Errorf("output path conflict: %s is both a file and a parent directory of %s", parent, destination))
			}
		}
	}
	for _, pair := range selection.FilePairs {
		for path := pair.EncryptedPath; path != filepath.Dir(path); path = filepath.Dir(path) {
			if _, exists := destinations[path]; exists {
				return errx.Usage(fmt.Errorf("output path %s conflicts with ciphertext input %s", path, pair.EncryptedPath))
			}
		}
	}
	return nil
}
