package app

import (
	"fmt"
	"strings"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/errx"
)

// prepareDiffSource validates --source before any config work: a blank
// directory is a calling error even when no config can be loaded.
func prepareDiffSource(req DiffRequest) error {
	if strings.TrimSpace(req.SourceDir) != "" || req.SourceDir == "" {
		return nil
	}
	return errx.Usage(fmt.Errorf("--source requires a directory"))
}

// applyDiffSource rewrites the selected mappings' plaintext paths to the
// mirrored tree, the read-only preview counterpart of prepareEncryptSource.
// It runs after selection so the reported selection already shows projected
// paths.
func applyDiffSource(cfg *config.Config, selection *config.ResolvedSelection, req DiffRequest) error {
	if req.SourceDir == "" {
		return nil
	}
	root, err := resolveMirroredTreeRoot(cfg, req.SourceDir, "--source")
	if err != nil {
		return err
	}
	return projectMirroredTree(cfg, selection, root, "--source", "read from")
}
