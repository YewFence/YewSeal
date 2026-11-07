package app

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/errx"
)

// prepareEncryptSource rewrites each selected mapping's plaintext path to the
// mirrored location under the caller-supplied source tree, the inverse of
// decrypt's --output projection. Ciphertext destinations stay registered.
func prepareEncryptSource(cfg *config.Config, selection *config.ResolvedSelection, req EncryptRequest) error {
	if req.SourceDir == "" {
		return nil
	}
	root := req.SourceDir
	if !filepath.IsAbs(root) {
		root = filepath.Join(config.CurrentDir(cfg), root)
	}
	root = filepath.Clean(root)
	info, err := os.Lstat(root)
	if err != nil {
		return errx.Usage(fmt.Errorf("--source must be an existing, real directory: %w", err))
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errx.Usage(fmt.Errorf("--source %s must be a real directory, not a file or symlink", root))
	}

	projectRoot := cfg.ProjectRoot
	if projectRoot == "" {
		projectRoot = config.CurrentDir(cfg)
	}
	selection.FilePairs = append([]config.ResolvedFilePair(nil), selection.FilePairs...)
	sources := make(map[string]struct{}, len(selection.FilePairs))
	for i, pair := range selection.FilePairs {
		rel, within := config.RelativePathWithin(projectRoot, pair.PlaintextPath)
		if !within || rel == "." {
			return errx.Usage(fmt.Errorf("plaintext path %s is not a file within project root %s", pair.PlaintextPath, projectRoot))
		}
		source := filepath.Join(root, filepath.FromSlash(rel))
		if _, exists := sources[source]; exists {
			return errx.Usage(fmt.Errorf("multiple file pairs read from %s", source))
		}
		sources[source] = struct{}{}
		selection.FilePairs[i].PlaintextPath = source
		selection.FilePairs[i].PlaintextSource = config.ValueSource{Kind: config.ValueSourceArgument, Detail: "--source"}
	}
	return nil
}
