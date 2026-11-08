package app

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/errx"
)

// resolveMirroredTreeRoot resolves the caller-supplied mirrored-tree root
// relative to the config discovery directory and verifies it is an existing,
// real directory.
func resolveMirroredTreeRoot(cfg *config.Config, dir, flag string) (string, error) {
	root := dir
	if !filepath.IsAbs(root) {
		root = filepath.Join(config.CurrentDir(cfg), root)
	}
	root = filepath.Clean(root)
	info, err := os.Lstat(root)
	if err != nil {
		return "", errx.Usage(fmt.Errorf("%s must be an existing, real directory: %w", flag, err))
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", errx.Usage(fmt.Errorf("%s %s must be a real directory, not a file or symlink", flag, root))
	}
	return root, nil
}

// projectMirroredTree rewrites each selected mapping's plaintext path to the
// mirrored location under root: root / relative(project-root,
// configured-plaintext-path). decrypt --output and encrypt --source share this
// projection so a delivered tree always reads back symmetrically. It never
// touches the filesystem.
func projectMirroredTree(cfg *config.Config, selection *config.ResolvedSelection, root, flag, verb string) error {
	projectRoot := cfg.ProjectRoot
	if projectRoot == "" {
		projectRoot = config.CurrentDir(cfg)
	}
	selection.FilePairs = append([]config.ResolvedFilePair(nil), selection.FilePairs...)
	seen := make(map[string]struct{}, len(selection.FilePairs))
	for i, pair := range selection.FilePairs {
		rel, within := config.RelativePathWithin(projectRoot, pair.PlaintextPath)
		if !within || rel == "." {
			return errx.Usage(fmt.Errorf("plaintext path %s is not a file within project root %s", pair.PlaintextPath, projectRoot))
		}
		path := filepath.Join(root, filepath.FromSlash(rel))
		if _, exists := seen[path]; exists {
			return errx.Usage(fmt.Errorf("multiple file pairs %s %s", verb, path))
		}
		seen[path] = struct{}{}
		selection.FilePairs[i].PlaintextPath = path
		selection.FilePairs[i].PlaintextSource = config.ValueSource{Kind: config.ValueSourceArgument, Detail: flag}
	}
	return nil
}
