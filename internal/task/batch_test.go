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

func TestGenerateOutputFilename(t *testing.T) {
	tests := []struct {
		name         string
		inputFile    string
		outputDir    string
		outputSuffix string
		mode         string
		expected     string
	}{
		{name: "encrypt_toml", inputFile: "config.toml", mode: "encrypt", expected: "config.enc.toml.yaml"},
		{name: "encrypt_yaml_output_dir", inputFile: "config.yaml", outputDir: "encrypted", mode: "encrypt", expected: filepath.Join("encrypted", "config.enc.yaml")},
		{name: "decrypt_enc_toml_yaml", inputFile: "config.enc.toml.yaml", mode: "decrypt", expected: "config.toml"},
		{name: "decrypt_output_dir", inputFile: "config.enc.yaml", outputDir: "decrypted", mode: "decrypt", expected: filepath.Join("decrypted", "config.yaml")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, GenerateOutputFilename(tt.inputFile, tt.outputDir, tt.outputSuffix, tt.mode))
		})
	}
}

func TestEncryptDecryptFilePairsWithFormatOverride(t *testing.T) {
	keyFile, publicKey := setupBatchTestEnv(t)
	require.NoError(t, os.WriteFile(".dev.vars", []byte("TOKEN=secret\n"), 0644))

	encSummary, err := Encrypt(Options{
		FilePairs: []FilePair{
			{PlaintextPath: ".dev.vars", EncryptedPath: ".dev.vars.enc.yaml", Format: "env"},
		},
		KeyFile:   keyFile,
		PublicKey: publicKey,
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
