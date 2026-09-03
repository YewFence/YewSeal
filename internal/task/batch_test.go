package task

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"filippo.io/age"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryptAndDecryptRequireResolvedFilePairs(t *testing.T) {
	_, err := Encrypt(Options{})
	require.EqualError(t, err, "no configured file pairs to encrypt")
	_, err = Decrypt(Options{})
	require.EqualError(t, err, "no configured file pairs to decrypt")
}

func TestEncryptDecryptFilePairsWithFormatOverride(t *testing.T) {
	keyFile, publicKey := setupBatchTestEnv(t)
	require.NoError(t, os.WriteFile(".dev.vars", []byte("TOKEN=secret\n"), 0644))

	encSummary, err := Encrypt(Options{
		FilePairs: []FilePair{
			{PlaintextPath: ".dev.vars", EncryptedPath: ".dev.vars.enc.yaml", Format: "env", Recipients: []string{publicKey}},
		},
		KeyFile: keyFile,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, encSummary.SuccessCount)

	require.NoError(t, os.Remove(".dev.vars"))
	decSummary, err := Decrypt(Options{
		FilePairs: []FilePair{
			{PlaintextPath: ".dev.vars", EncryptedPath: ".dev.vars.enc.yaml", Format: "env"},
		},
		KeyFile: keyFile,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, decSummary.SuccessCount)

	content, readErr := os.ReadFile(".dev.vars")
	require.NoError(t, readErr)
	assert.Equal(t, "TOKEN=secret\n", string(content))
}

func setupBatchTestEnv(t *testing.T) (string, string) {
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
