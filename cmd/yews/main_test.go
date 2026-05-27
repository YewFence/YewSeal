package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"filippo.io/age"
	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateCLIFormatOverride_Empty(t *testing.T) {
	format, err := validateCLIFormatOverride("")
	require.NoError(t, err)
	assert.Equal(t, "", format)
}

func TestValidateCLIFormatOverride_NormalizesAlias(t *testing.T) {
	format, err := validateCLIFormatOverride("dotenv")
	require.NoError(t, err)
	assert.Equal(t, "env", format)
}

func TestValidateCLIFormatOverride_Invalid(t *testing.T) {
	_, err := validateCLIFormatOverride("xml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported format")
}

func TestResolveFormatOverride_UsesConfigValue(t *testing.T) {
	format, err := resolveFormatOverride("", config.FilePair{Format: "env"})
	require.NoError(t, err)
	assert.Equal(t, "env", format)
}

func TestResolveFormatOverride_PrefersCLIValue(t *testing.T) {
	format, err := resolveFormatOverride("json", config.FilePair{Format: "env"})
	require.NoError(t, err)
	assert.Equal(t, "json", format)
}

func TestResolveTargetFilePairs_EmptyUsesAllConfiguredFiles(t *testing.T) {
	cfg := &config.Config{
		Encryption: config.EncryptionConfig{
			Files: []config.FilePair{
				{PlaintextPath: "wrangler.toml", EncryptedPath: "wrangler.enc.toml.yaml"},
				{PlaintextPath: ".dev.vars", EncryptedPath: ".dev.vars.enc.yaml", Format: "env"},
			},
		},
	}

	filePairs, err := resolveTargetFilePairs(cfg, "")
	require.NoError(t, err)
	assert.Len(t, filePairs, 2)
}

func TestResolveTargetFilePairs_MatchesPlaintextPath(t *testing.T) {
	cfg := &config.Config{
		Encryption: config.EncryptionConfig{
			Files: []config.FilePair{
				{PlaintextPath: "wrangler.toml", EncryptedPath: "wrangler.enc.toml.yaml"},
			},
		},
	}

	filePairs, err := resolveTargetFilePairs(cfg, "wrangler.toml")
	require.NoError(t, err)
	require.Len(t, filePairs, 1)
	assert.Equal(t, "wrangler.enc.toml.yaml", filePairs[0].EncryptedPath)
}

func TestResolveTargetFilePairs_MatchesEncryptedPath(t *testing.T) {
	cfg := &config.Config{
		Encryption: config.EncryptionConfig{
			Files: []config.FilePair{
				{PlaintextPath: "wrangler.toml", EncryptedPath: "wrangler.enc.toml.yaml"},
			},
		},
	}

	filePairs, err := resolveTargetFilePairs(cfg, "wrangler.enc.toml.yaml")
	require.NoError(t, err)
	require.Len(t, filePairs, 1)
	assert.Equal(t, "wrangler.toml", filePairs[0].PlaintextPath)
}

func TestResolveTargetFilePairs_UnknownTarget(t *testing.T) {
	cfg := config.DefaultConfig()

	_, err := resolveTargetFilePairs(cfg, "unknown.toml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is not configured")
}

func TestResolveSingleTargetFilePair_RequiresTarget(t *testing.T) {
	cfg := config.DefaultConfig()

	_, err := resolveSingleTargetFilePair(cfg, "", "view")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "view requires exactly one target")
}

func TestResolveSingleTargetFilePair_MatchesPlaintextPath(t *testing.T) {
	cfg := &config.Config{
		Encryption: config.EncryptionConfig{
			Files: []config.FilePair{
				{PlaintextPath: "wrangler.toml", EncryptedPath: "wrangler.enc.toml.yaml"},
			},
		},
	}

	filePair, err := resolveSingleTargetFilePair(cfg, "wrangler.toml", "view")
	require.NoError(t, err)
	assert.Equal(t, "wrangler.toml", filePair.PlaintextPath)
	assert.Equal(t, "wrangler.enc.toml.yaml", filePair.EncryptedPath)
}

func TestWriteViewedTarget_WritesPlaintextToWriterOnly(t *testing.T) {
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
	keyContent := fmt.Sprintf("# created: %s\n# public key: %s\n%s\n",
		time.Now().UTC().Format(time.RFC3339),
		identity.Recipient().String(),
		identity.String(),
	)
	require.NoError(t, os.WriteFile(keyFile, []byte(keyContent), 0600))

	plaintextFile := "config.yaml"
	encryptedFile := "config.enc.yaml"
	outputFile := "view-output.yaml"
	plaintext := []byte("database:\n  host: localhost\n  password: secret123\n")

	require.NoError(t, os.WriteFile(plaintextFile, plaintext, 0644))
	require.NoError(t, crypto.Encrypt(plaintextFile, encryptedFile, keyFile, identity.Recipient().String(), "yaml", false))
	require.NoError(t, os.Remove(plaintextFile))

	cfg := &config.Config{
		Encryption: config.EncryptionConfig{
			Files: []config.FilePair{
				{PlaintextPath: outputFile, EncryptedPath: encryptedFile, Format: "yaml"},
			},
		},
		Key: config.KeyConfig{FilePath: keyFile},
	}

	var out bytes.Buffer
	err = writeViewedTarget(&out, cfg, encryptedFile, keyFile, "", false)
	require.NoError(t, err)

	assert.Contains(t, out.String(), "localhost")
	assert.Contains(t, out.String(), "secret123")
	_, statErr := os.Stat(outputFile)
	assert.True(t, os.IsNotExist(statErr))
}
