package main

import (
	"testing"

	"github.com/YewFence/YewSeal/internal/config"
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
