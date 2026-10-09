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
		if !req.Inplace {
			for _, pair := range selection.FilePairs {
				if pair.PlaintextMode == config.PlaintextDelivery {
					return errx.Usage(fmt.Errorf("delivery plaintext %s requires --output DIR for delivery or --inplace to write the configured plaintext path", pair.PlaintextPath))
				}
			}
		}
		return nil
	}
	if req.Force || req.Inplace {
		return errx.Usage(fmt.Errorf("--output conflicts with --force=true or --inplace=true"))
	}
	root := req.OutputDir
	if !filepath.IsAbs(root) {
		root = filepath.Join(config.CurrentDir(cfg), root)
	}
	root = filepath.Clean(root)
	info, err := os.Lstat(root)
	if err != nil {
		return errx.Usage(fmt.Errorf("--output must be an existing, real, empty directory: %w", err))
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errx.Usage(fmt.Errorf("--output %s must be a real directory, not a file or symlink", root))
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return errx.Usage(fmt.Errorf("failed to inspect --output directory %s: %w", root, err))
	}
	if len(entries) != 0 {
		return errx.Usage(fmt.Errorf("--output directory %s must be completely empty (including hidden entries)", root))
	}
	projectRoot := cfg.ProjectRoot
	if projectRoot == "" {
		projectRoot = config.CurrentDir(cfg)
	}
	selection.FilePairs = append([]config.ResolvedFilePair(nil), selection.FilePairs...)
	destinations := make(map[string]struct{}, len(selection.FilePairs))
	for i, pair := range selection.FilePairs {
		rel, within := config.RelativePathWithin(projectRoot, pair.PlaintextPath)
		if !within || rel == "." {
			return errx.Usage(fmt.Errorf("plaintext path %s is not a file within project root %s", pair.PlaintextPath, projectRoot))
		}
		destination := filepath.Join(root, filepath.FromSlash(rel))
		if _, exists := destinations[destination]; exists {
			return errx.Usage(fmt.Errorf("multiple file pairs write to %s", destination))
		}
		destinations[destination] = struct{}{}
		selection.FilePairs[i].PlaintextPath = destination
		selection.FilePairs[i].PlaintextSource = config.ValueSource{Kind: config.ValueSourceArgument, Detail: "--output"}
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
