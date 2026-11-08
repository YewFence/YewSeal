package app

import (
	"github.com/YewFence/YewSeal/internal/config"
)

// prepareEncryptSource rewrites each selected mapping's plaintext path to the
// mirrored location under the caller-supplied source tree, the inverse of
// decrypt's --output projection. Ciphertext destinations stay registered.
func prepareEncryptSource(cfg *config.Config, selection *config.ResolvedSelection, req EncryptRequest) error {
	if req.SourceDir == "" {
		return nil
	}
	root, err := resolveMirroredTreeRoot(cfg, req.SourceDir, "--source")
	if err != nil {
		return err
	}
	return projectMirroredTree(cfg, selection, root, "--source", "read from")
}
