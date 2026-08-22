package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSavePublicKeyToConfig_CreateNew(t *testing.T) {
	tmpDir := t.TempDir()
	withProjectWorkingDir(t, tmpDir)

	err := SavePublicKeyToConfig("age1testpublickey", []config.FilePair{
		{PlaintextPath: "secrets.toml", EncryptedPath: "secrets.enc.yaml"},
	})
	require.NoError(t, err)

	content, err := os.ReadFile(".yewseal.toml")
	require.NoError(t, err)

	assert.Contains(t, string(content), "[[encryption.files]]")
	assert.Contains(t, string(content), "plaintext = \"secrets.toml\"")
	assert.Contains(t, string(content), "encrypted = \"secrets.enc.yaml\"")
	assert.Contains(t, string(content), "[key]")
	assert.Contains(t, string(content), "file_path = \".age/keys.txt\"")
	assert.Contains(t, string(content), "public_key = \"age1testpublickey\"")
}

func TestSavePublicKeyToConfig_OverwriteExisting(t *testing.T) {
	tmpDir := t.TempDir()
	withProjectWorkingDir(t, tmpDir)

	existingContent := `[encryption]

[[encryption.files]]
plaintext = "old.toml"
encrypted = "old.enc.yaml"

[key]
file_path = ".age/keys.txt"
public_key = "age1existingkey"
`
	err := os.WriteFile(".yewseal.toml", []byte(existingContent), 0644)
	require.NoError(t, err)

	err = SavePublicKeyToConfig("age1newkey", []config.FilePair{
		{PlaintextPath: "new.toml", EncryptedPath: "new.enc.yaml"},
	})
	require.NoError(t, err)

	content, err := os.ReadFile(".yewseal.toml")
	require.NoError(t, err)

	assert.Contains(t, string(content), "age1newkey")
	assert.Contains(t, string(content), "new.toml")
	assert.Contains(t, string(content), "new.enc.yaml")
	assert.NotContains(t, string(content), "age1existingkey")
	assert.NotContains(t, string(content), "old.toml")
}

func TestSavePublicKeyToConfig_MultipleFilePairs(t *testing.T) {
	tmpDir := t.TempDir()
	withProjectWorkingDir(t, tmpDir)

	err := SavePublicKeyToConfig("age1test", []config.FilePair{
		{PlaintextPath: "app.toml", EncryptedPath: "app.enc.toml"},
		{PlaintextPath: ".dev.vars", EncryptedPath: ".dev.vars.enc.yaml", Format: "env"},
	})
	require.NoError(t, err)

	content, err := os.ReadFile(".yewseal.toml")
	require.NoError(t, err)

	assert.Contains(t, string(content), "plaintext = \"app.toml\"")
	assert.Contains(t, string(content), "encrypted = \"app.enc.toml\"")
	assert.Contains(t, string(content), "plaintext = \".dev.vars\"")
	assert.Contains(t, string(content), "encrypted = \".dev.vars.enc.yaml\"")
	assert.Contains(t, string(content), "format = \"env\"")
}

func TestSavePublicKeyToConfig_ConfigCanBeLoaded(t *testing.T) {
	tmpDir := t.TempDir()
	withProjectWorkingDir(t, tmpDir)

	err := SavePublicKeyToConfig("age1testkey", []config.FilePair{
		{PlaintextPath: "input.toml", EncryptedPath: "output.yaml"},
	})
	require.NoError(t, err)

	cfg, err := config.LoadConfig()
	require.NoError(t, err)
	require.Len(t, cfg.GetFiles(), 1)
	assert.Equal(t, filepath.Join(tmpDir, "input.toml"), cfg.GetFiles()[0].PlaintextPath)
	assert.Equal(t, filepath.Join(tmpDir, "output.yaml"), cfg.GetFiles()[0].EncryptedPath)
	assert.Equal(t, "age1testkey", cfg.GetPublicKey())
}

func TestSavePublicKeyToConfig_WritesWithoutIndentation(t *testing.T) {
	tmpDir := t.TempDir()
	withProjectWorkingDir(t, tmpDir)

	err := SavePublicKeyToConfig("age1test", []config.FilePair{
		{PlaintextPath: "app.toml", EncryptedPath: "app.enc.toml"},
	})
	require.NoError(t, err)

	content, err := os.ReadFile(".yewseal.toml")
	require.NoError(t, err)

	assert.Contains(t, string(content), "plaintext = \"app.toml\"")
	assert.Contains(t, string(content), "encrypted = \"app.enc.toml\"")
	assert.NotContains(t, string(content), "\n  plaintext =")
	assert.NotContains(t, string(content), "\n  encrypted =")
	assert.NotContains(t, string(content), "\n  file_path =")
	assert.NotContains(t, string(content), "\n  public_key =")
}
