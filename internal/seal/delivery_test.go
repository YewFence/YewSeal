package seal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/YewFence/YewSeal/internal/sopsx"
	"github.com/stretchr/testify/require"
)

func TestDecryptDeliveryWritesPrivateMirroredFile(t *testing.T) {
	env := setupTestEnv(t)
	plain := []byte("TOKEN=secret\n")
	ciphertext, err := sopsx.Encrypt(plain, "env", []string{env.publicKey})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile("secret.enc.env", ciphertext, 0644))
	root := t.TempDir()
	require.NoError(t, os.Chmod(root, 0751))
	output := filepath.Join(root, "config", "nested", "secret")
	require.NoError(t, Decrypt(DecryptOptions{
		InputFile: "secret.enc.env", OutputFile: output, IdentityBundle: env.bundle,
		FormatOverride: "env", Delivery: true,
	}))
	actual, err := os.ReadFile(output)
	require.NoError(t, err)
	require.Equal(t, plain, actual)
	for path, mode := range map[string]os.FileMode{
		root:                                    0751,
		filepath.Join(root, "config"):           0700,
		filepath.Join(root, "config", "nested"): 0700,
		output:                                  0600,
	} {
		info, err := os.Lstat(path)
		require.NoError(t, err)
		require.Equal(t, mode, info.Mode().Perm())
		if path == output {
			require.True(t, info.Mode().IsRegular())
		} else {
			require.True(t, info.IsDir())
		}
	}
	require.NoFileExists(t, "secret")
}

func TestWriteDeliveredFileRejectsNonDirectoryParent(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(parent, []byte("untouched"), 0644))
	err := writeDeliveredFile(filepath.Join(parent, "secret.yaml"), []byte("secret"))
	require.ErrorContains(t, err, "failed to create delivery directory")
	actual, err := os.ReadFile(parent)
	require.NoError(t, err)
	require.Equal(t, "untouched", string(actual))
}

func TestWriteDeliveredFileNeverReusesExistingTargets(t *testing.T) {
	for _, kind := range []string{"file", "directory", "symlink"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			output := filepath.Join(root, "output")
			protected := filepath.Join(root, "protected")
			require.NoError(t, os.WriteFile(protected, []byte("untouched"), 0644))
			switch kind {
			case "file":
				require.NoError(t, os.WriteFile(output, []byte("untouched"), 0644))
			case "directory":
				require.NoError(t, os.Mkdir(output, 0755))
			case "symlink":
				require.NoError(t, os.Symlink(protected, output))
			}
			err := writeDeliveredFile(output, []byte("secret"))
			require.ErrorIs(t, err, os.ErrExist)
			require.ErrorContains(t, err, "failed to create delivery file")
			actual, err := os.ReadFile(protected)
			require.NoError(t, err)
			require.Equal(t, "untouched", string(actual))
			switch kind {
			case "file", "symlink":
				actual, err := os.ReadFile(output)
				require.NoError(t, err)
				require.Equal(t, "untouched", string(actual))
			case "directory":
				entries, err := os.ReadDir(output)
				require.NoError(t, err)
				require.Empty(t, entries)
			}
		})
	}
}
