package diff

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"filippo.io/age"
	"github.com/YewFence/YewSeal/internal/seal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnifiedDiffLineOriented(t *testing.T) {
	diff := UnifiedDiff(
		"config.yaml",
		"config.enc.yaml (decrypted)",
		[]byte("database:\n  host: local-change\n"),
		[]byte("database:\n  host: localhost\n"),
	)

	assert.Contains(t, diff, "--- config.yaml")
	assert.Contains(t, diff, "+++ config.enc.yaml (decrypted)")
	assert.Contains(t, diff, "-  host: local-change\n")
	assert.Contains(t, diff, "+  host: localhost\n")
}

func TestHighlightUnifiedDiff(t *testing.T) {
	plain := "--- config.yaml\n+++ config.enc.yaml (decrypted)\n@@\n-  host: local-change\n+  host: localhost\n"

	highlighted := HighlightUnifiedDiff(plain, true)

	assert.Contains(t, highlighted, "\x1b[")
	assert.Contains(t, highlighted, "--- config.yaml")
	assert.Contains(t, highlighted, "-  host: local-change")
	assert.Equal(t, plain, HighlightUnifiedDiff(plain, false))
}

func TestPlaintextAgainstEncrypted(t *testing.T) {
	keyFile, publicKey := setupDiffTestEnv(t)
	require.NoError(t, os.WriteFile("config.yaml", []byte("database:\n  host: localhost\n"), 0644))
	require.NoError(t, seal.Encrypt(seal.EncryptOptions{
		InputFile:      "config.yaml",
		OutputFile:     "config.enc.yaml",
		KeyFile:        keyFile,
		PublicKey:      publicKey,
		FormatOverride: "yaml",
	}))
	require.NoError(t, os.WriteFile("config.yaml", []byte("database:\n  host: local-change\n"), 0644))

	result, err := PlaintextAgainstEncrypted(Options{
		PlaintextFile:  "config.yaml",
		EncryptedFile:  "config.enc.yaml",
		KeyFile:        keyFile,
		FormatOverride: "yaml",
	})
	require.NoError(t, err)

	assert.True(t, result.Different)
	assert.Contains(t, result.Diff, "-  host: local-change")
	assert.Contains(t, result.Diff, "host: localhost")
}

func setupDiffTestEnv(t *testing.T) (string, string) {
	t.Helper()

	tempDir := t.TempDir()
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tempDir))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(oldWd))
	})

	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	keyDir := filepath.Join(tempDir, ".age")
	require.NoError(t, os.MkdirAll(keyDir, 0700))
	keyFile := filepath.Join(keyDir, "keys.txt")
	publicKey := identity.Recipient().String()
	keyContent := fmt.Sprintf("# created: %s\n# public key: %s\n%s\n",
		time.Now().UTC().Format(time.RFC3339),
		publicKey,
		identity.String(),
	)
	require.NoError(t, os.WriteFile(keyFile, []byte(keyContent), 0600))
	return keyFile, publicKey
}
