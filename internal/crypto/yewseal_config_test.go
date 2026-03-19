package crypto

import (
	"os"
	"testing"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSavePublicKeyToConfig_CreateNew(t *testing.T) {
	oldWd, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	err := SavePublicKeyToConfig("age1testpublickey", []config.FilePair{
		{Dec: "secrets.toml", Enc: "secrets.enc.yaml"},
	})
	require.NoError(t, err)

	content, err := os.ReadFile(".yewseal.toml")
	require.NoError(t, err)

	assert.Contains(t, string(content), "[[encryption.files]]")
	assert.Contains(t, string(content), "dec = \"secrets.toml\"")
	assert.Contains(t, string(content), "enc = \"secrets.enc.yaml\"")
	assert.Contains(t, string(content), "[key]")
	assert.Contains(t, string(content), "file_path = \".age/keys.txt\"")
	assert.Contains(t, string(content), "public_key = \"age1testpublickey\"")
}

func TestSavePublicKeyToConfig_OverwriteExisting(t *testing.T) {
	oldWd, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	existingContent := `[encryption]

[[encryption.files]]
dec = "old.toml"
enc = "old.enc.yaml"

[key]
file_path = ".age/keys.txt"
public_key = "age1existingkey"
`
	err := os.WriteFile(".yewseal.toml", []byte(existingContent), 0644)
	require.NoError(t, err)

	err = SavePublicKeyToConfig("age1newkey", []config.FilePair{
		{Dec: "new.toml", Enc: "new.enc.yaml"},
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
	oldWd, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	err := SavePublicKeyToConfig("age1test", []config.FilePair{
		{Dec: "app.toml", Enc: "app.enc.toml.yaml"},
		{Dec: ".dev.vars", Enc: ".dev.vars.enc.yaml", Format: "env"},
	})
	require.NoError(t, err)

	content, err := os.ReadFile(".yewseal.toml")
	require.NoError(t, err)

	assert.Contains(t, string(content), "dec = \"app.toml\"")
	assert.Contains(t, string(content), "enc = \"app.enc.toml.yaml\"")
	assert.Contains(t, string(content), "dec = \".dev.vars\"")
	assert.Contains(t, string(content), "enc = \".dev.vars.enc.yaml\"")
	assert.Contains(t, string(content), "format = \"env\"")
}

func TestSavePublicKeyToConfig_ConfigCanBeLoaded(t *testing.T) {
	oldWd, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	err := SavePublicKeyToConfig("age1testkey", []config.FilePair{
		{Dec: "input.toml", Enc: "output.yaml"},
	})
	require.NoError(t, err)

	cfg, err := config.LoadConfig()
	require.NoError(t, err)
	require.Len(t, cfg.GetFiles(), 1)
	assert.Equal(t, "input.toml", cfg.GetFiles()[0].Dec)
	assert.Equal(t, "output.yaml", cfg.GetFiles()[0].Enc)
	assert.Equal(t, "age1testkey", cfg.GetPublicKey())
}
