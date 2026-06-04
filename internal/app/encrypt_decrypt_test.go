package app

import (
	"os"
	"testing"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryptFiles_SingleFileOverrideUsesProvidedPathsAndFormat(t *testing.T) {
	env := newAppCryptoTestEnv(t)

	require.NoError(t, os.WriteFile("secrets.vars", []byte("TOKEN=secret\n"), 0644))
	cfg := config.DefaultConfig()

	err := EncryptFiles(cfg, EncryptRequest{
		KeyFile:   env.keyFile,
		PublicKey: env.publicKey,
		Input:     "secrets.vars",
		InputSet:  true,
		Output:    "secrets.vars.enc.yaml",
		OutputSet: true,
		Format:    "env",
		Parallel:  1,
	})
	require.NoError(t, err)

	require.NoError(t, os.Remove("secrets.vars"))
	err = DecryptFiles(cfg, DecryptRequest{
		KeyFile:   env.keyFile,
		Input:     "secrets.vars.enc.yaml",
		InputSet:  true,
		Output:    "secrets.vars",
		OutputSet: true,
		Format:    "env",
		Parallel:  1,
	})
	require.NoError(t, err)

	content, err := os.ReadFile("secrets.vars")
	require.NoError(t, err)
	assert.Equal(t, "TOKEN=secret\n", string(content))
}

func TestEncryptFiles_DirModeRejectsFormatOverride(t *testing.T) {
	cfg := config.DefaultConfig()
	tempDir := t.TempDir()

	err := EncryptFiles(cfg, EncryptRequest{
		Target: tempDir,
		Format: "yaml",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--format is only supported in single-file mode")
}

func TestDecryptFiles_ConfigModeRejectsFormatOverride(t *testing.T) {
	cfg := config.DefaultConfig()

	err := DecryptFiles(cfg, DecryptRequest{
		Format: "yaml",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--format is only supported in single-file mode")
}

type appCryptoTestEnv struct {
	keyFile   string
	publicKey string
}

func newAppCryptoTestEnv(t *testing.T) appCryptoTestEnv {
	t.Helper()

	tempDir := t.TempDir()
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tempDir))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(oldWd))
	})

	keyFile, publicKey := createAgeKeyFile(t, tempDir)
	return appCryptoTestEnv{keyFile: keyFile, publicKey: publicKey}
}
