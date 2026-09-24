package task

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"filippo.io/age"
	"github.com/YewFence/YewSeal/internal/agekey"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryptAndDecryptRequireResolvedFilePairs(t *testing.T) {
	encryptSummary, err := Encrypt(Options{})
	require.NoError(t, err)
	require.Empty(t, encryptSummary.Results)
	decryptSummary, err := Decrypt(Options{})
	require.NoError(t, err)
	require.Empty(t, decryptSummary.Results)
}

func TestEncryptDecryptFilePairsWithFormatOverride(t *testing.T) {
	_, publicKey, bundle := setupBatchTestEnv(t)
	require.NoError(t, os.WriteFile(".dev.vars", []byte("TOKEN=secret\n"), 0644))

	encSummary, err := Encrypt(Options{
		FilePairs: []FilePair{
			{PlaintextPath: ".dev.vars", EncryptedPath: ".dev.vars.enc.yaml", Format: "env", Recipients: []string{publicKey}},
		},
		IdentityBundle: bundle,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, encSummary.SuccessCount)
	assert.Equal(t, 1, encSummary.EncryptedCount)

	require.NoError(t, os.Remove(".dev.vars"))
	decSummary, err := Decrypt(Options{
		FilePairs: []FilePair{
			{PlaintextPath: ".dev.vars", EncryptedPath: ".dev.vars.enc.yaml", Format: "env"},
		},
		IdentityBundle: bundle,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, decSummary.SuccessCount)

	content, readErr := os.ReadFile(".dev.vars")
	require.NoError(t, readErr)
	assert.Equal(t, "TOKEN=secret\n", string(content))
}

func TestEncryptDistinguishesUnchangedAndMissingPlaintext(t *testing.T) {
	_, publicKey, bundle := setupBatchTestEnv(t)
	require.NoError(t, os.WriteFile("same.yaml", []byte("token: value\n"), 0644))
	pairs := []FilePair{
		{PlaintextPath: "same.yaml", EncryptedPath: "same.enc.yaml", Format: "yaml", Recipients: []string{publicKey}},
		{PlaintextPath: "missing/value.yaml", EncryptedPath: "missing/output/value.enc.yaml", Format: "yaml", Recipients: []string{publicKey}},
	}
	first, err := Encrypt(Options{FilePairs: pairs[:1], IdentityBundle: bundle})
	require.NoError(t, err)
	require.Equal(t, 1, first.EncryptedCount)

	second, err := Encrypt(Options{FilePairs: pairs, IdentityBundle: bundle})
	require.NoError(t, err)
	require.Equal(t, 1, second.UnchangedCount)
	require.Equal(t, 1, second.MissingPlaintextCount)
	require.Equal(t, 1, second.SkippedCount)
	require.NoDirExists(t, "missing/output")
}

func TestEncryptFailureDoesNotCountAsEncrypted(t *testing.T) {
	_, publicKey, bundle := setupBatchTestEnv(t)
	require.NoError(t, os.WriteFile("broken.yaml", []byte("not: [valid\n"), 0644))
	summary, err := Encrypt(Options{FilePairs: []FilePair{{PlaintextPath: "broken.yaml", EncryptedPath: "broken.enc.yaml", Format: "yaml", Recipients: []string{publicKey}}}, IdentityBundle: bundle})
	require.Error(t, err)
	require.Equal(t, 1, summary.FailedCount)
	require.Zero(t, summary.EncryptedCount)
}

func TestEncryptForceFreshlyEncryptsExistingPlaintext(t *testing.T) {
	_, publicKey, bundle := setupBatchTestEnv(t)
	require.NoError(t, os.WriteFile("secret.yaml", []byte("first: value\nsecond: value\n"), 0644))
	pair := FilePair{PlaintextPath: "secret.yaml", EncryptedPath: "secret.enc.yaml", Format: "yaml", Recipients: []string{publicKey}}
	_, err := Encrypt(Options{FilePairs: []FilePair{pair}, IdentityBundle: bundle})
	require.NoError(t, err)
	original, err := os.ReadFile("secret.enc.yaml")
	require.NoError(t, err)

	summary, err := Encrypt(Options{FilePairs: []FilePair{pair}, Force: true})
	require.NoError(t, err)
	require.Equal(t, 1, summary.EncryptedCount)
	refreshed, err := os.ReadFile("secret.enc.yaml")
	require.NoError(t, err)
	require.NotEqual(t, original, refreshed)
}

func setupBatchTestEnv(t *testing.T) (string, string, agekey.IdentityBundle) {
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
	bundle, err := agekey.GetIdentityBundle(keyFile)
	require.NoError(t, err)
	return keyFile, publicKey, bundle
}
