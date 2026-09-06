package app

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"filippo.io/age"
	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/seal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateCLIFormatOverride_Empty(t *testing.T) {
	format, err := ValidateCLIFormatOverride("")
	require.NoError(t, err)
	assert.Equal(t, "", format)
}

func TestValidateCLIFormatOverride_NormalizesAlias(t *testing.T) {
	format, err := ValidateCLIFormatOverride("dotenv")
	require.NoError(t, err)
	assert.Equal(t, "env", format)
}

func TestValidateCLIFormatOverride_Binary(t *testing.T) {
	format, err := ValidateCLIFormatOverride("bin")
	require.NoError(t, err)
	assert.Equal(t, "binary", format)
}

func TestValidateCLIFormatOverride_Invalid(t *testing.T) {
	_, err := ValidateCLIFormatOverride("xml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported format")
}

func TestWriteViewedTarget_WritesPlaintextToWriterOnly(t *testing.T) {
	tempDir := t.TempDir()
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tempDir))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(oldWd))
	})

	keyFile, publicKey := createAgeKeyFile(t, tempDir)

	plaintextFile := "config.yaml"
	encryptedFile := "config.enc.yaml"
	outputFile := "view-output.yaml"
	plaintext := []byte("database:\n  host: localhost\n  password: secret123\n")

	require.NoError(t, os.WriteFile(plaintextFile, plaintext, 0644))
	require.NoError(t, seal.Encrypt(seal.EncryptOptions{
		InputFile:      plaintextFile,
		OutputFile:     encryptedFile,
		Recipients:     []string{publicKey},
		FormatOverride: "yaml",
	}))
	require.NoError(t, os.Remove(plaintextFile))

	cfg := &config.Config{
		Encryption: config.EncryptionConfig{
			Files: []config.FilePair{
				{PlaintextPath: outputFile, EncryptedPath: encryptedFile, Format: "yaml"},
			},
		},
	}

	var out bytes.Buffer
	err = WriteViewedTarget(&out, cfg, encryptedFile, keyFile, "", false)
	require.NoError(t, err)

	assert.Contains(t, out.String(), "localhost")
	assert.Contains(t, out.String(), "secret123")
	_, statErr := os.Stat(outputFile)
	assert.True(t, os.IsNotExist(statErr))
}

func createAgeKeyFile(t *testing.T, dir string) (string, string) {
	t.Helper()

	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	keyDir := filepath.Join(dir, ".age")
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
