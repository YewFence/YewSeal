package app

import (
	"bytes"
	"os"
	"testing"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/seal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiffPlaintextAgainstEncryptedTargets_WritesDiffForDifferentTarget(t *testing.T) {
	env := newDiffTestEnv(t)

	require.NoError(t, os.WriteFile("config.yaml", []byte("database:\n  host: localhost\n"), 0644))
	require.NoError(t, seal.Encrypt(seal.EncryptOptions{
		InputFile:      "config.yaml",
		OutputFile:     "config.enc.yaml",
		KeyFile:        env.keyFile,
		PublicKey:      env.publicKey,
		FormatOverride: "yaml",
	}))
	require.NoError(t, os.WriteFile("config.yaml", []byte("database:\n  host: local-change\n"), 0644))

	cfg := &config.Config{
		Encryption: config.EncryptionConfig{
			Files: []config.FilePair{
				{PlaintextPath: "config.yaml", EncryptedPath: "config.enc.yaml", Format: "yaml"},
			},
		},
	}

	var out bytes.Buffer
	result, err := DiffPlaintextAgainstEncryptedTargets(&out, cfg, "config.yaml", env.keyFile, "", false)
	require.NoError(t, err)

	assert.True(t, result.Different)
	assert.Contains(t, out.String(), "--- config.yaml")
	assert.Contains(t, out.String(), "+++ config.enc.yaml (decrypted)")
	assert.Contains(t, out.String(), "-  host: local-change")
	assert.Contains(t, out.String(), "+    host: localhost")
}

func TestDiffPlaintextAgainstEncryptedTargets_NoOutputForIdenticalTarget(t *testing.T) {
	env := newDiffTestEnv(t)

	require.NoError(t, os.WriteFile("config.yaml", []byte("database:\n  host: localhost\n"), 0644))
	require.NoError(t, seal.Encrypt(seal.EncryptOptions{
		InputFile:      "config.yaml",
		OutputFile:     "config.enc.yaml",
		KeyFile:        env.keyFile,
		PublicKey:      env.publicKey,
		FormatOverride: "yaml",
	}))
	decrypted, err := seal.DecryptToBytes(seal.DecryptBytesOptions{
		InputFile:      "config.enc.yaml",
		OutputFile:     "config.yaml",
		KeyFile:        env.keyFile,
		FormatOverride: "yaml",
	})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile("config.yaml", decrypted, 0644))

	cfg := &config.Config{
		Encryption: config.EncryptionConfig{
			Files: []config.FilePair{
				{PlaintextPath: "config.yaml", EncryptedPath: "config.enc.yaml", Format: "yaml"},
			},
		},
	}

	var out bytes.Buffer
	result, err := DiffPlaintextAgainstEncryptedTargets(&out, cfg, "", env.keyFile, "", false)
	require.NoError(t, err)

	assert.False(t, result.Different)
	assert.Empty(t, out.String())
}

type diffTestEnv struct {
	keyFile   string
	publicKey string
}

func newDiffTestEnv(t *testing.T) diffTestEnv {
	t.Helper()

	tempDir := t.TempDir()
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tempDir))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(oldWd))
	})

	keyFile, publicKey := createAgeKeyFile(t, tempDir)
	return diffTestEnv{keyFile: keyFile, publicKey: publicKey}
}
