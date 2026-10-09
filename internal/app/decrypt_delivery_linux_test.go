package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/errx"
	"github.com/stretchr/testify/require"
)

func TestDecryptDeliveryUnreadableRootFailsBeforeWrites(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.Chmod(root, 0000))
	t.Cleanup(func() { require.NoError(t, os.Chmod(root, 0700)) })
	projectRoot := t.TempDir()
	cfg := &config.Config{CurrentDir: projectRoot, ProjectRoot: projectRoot, Encryption: config.EncryptionConfig{Files: []config.FilePair{
		{PlaintextPath: filepath.Join(projectRoot, "secret.yaml"), EncryptedPath: filepath.Join(projectRoot, "secret.enc.yaml")},
	}}}
	err := DecryptFiles(cfg, DecryptRequest{OutputDir: root, KeyFile: "nonexistent-key", UpdateGitignore: true})
	require.ErrorContains(t, err, "failed to inspect --output directory")
	require.ErrorIs(t, err, os.ErrPermission)
	var usage *errx.UsageError
	require.ErrorAs(t, err, &usage)
	require.NoFileExists(t, filepath.Join(projectRoot, ".gitignore"))
	require.NoError(t, os.Chmod(root, 0700))
	entries, err := os.ReadDir(root)
	require.NoError(t, err)
	require.Empty(t, entries)
}
