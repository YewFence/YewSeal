package crypto

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSavePublicKeyToConfig_CreateNew(t *testing.T) {
	oldWd, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	err := SavePublicKeyToConfig("age1testpublickey", "secrets.toml", "secrets.enc.yaml")
	require.NoError(t, err)

	content, err := os.ReadFile(".yewseal.toml")
	require.NoError(t, err)

	// Check content structure
	assert.Contains(t, string(content), "[encryption]")
	assert.Contains(t, string(content), "input_file = \"secrets.toml\"")
	assert.Contains(t, string(content), "output_file = \"secrets.enc.yaml\"")
	assert.Contains(t, string(content), "[key]")
	assert.Contains(t, string(content), "file_path = \".age/keys.txt\"")
	assert.Contains(t, string(content), "public_key = \"age1testpublickey\"")
}

func TestSavePublicKeyToConfig_AlreadyExists(t *testing.T) {
	oldWd, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	// Create config with public_key already present
	existingContent := `[encryption]
input_file = "secrets.toml"
output_file = "secrets.enc.yaml"

[key]
file_path = ".age/keys.txt"
public_key = "age1existingkey"
`
	err := os.WriteFile(".yewseal.toml", []byte(existingContent), 0644)
	require.NoError(t, err)

	// Try to save a new key - should be skipped
	err = SavePublicKeyToConfig("age1newkey", "new.toml", "new.enc.yaml")
	require.NoError(t, err)

	content, err := os.ReadFile(".yewseal.toml")
	require.NoError(t, err)

	// Original key should still be there, new key should not
	assert.Contains(t, string(content), "age1existingkey")
	assert.NotContains(t, string(content), "age1newkey")
}

func TestSavePublicKeyToConfig_UpdateExisting(t *testing.T) {
	oldWd, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	// Create config without public_key
	existingContent := `[encryption]
input_file = "secrets.toml"
output_file = "secrets.enc.yaml"

[key]
file_path = ".age/keys.txt"
`
	err := os.WriteFile(".yewseal.toml", []byte(existingContent), 0644)
	require.NoError(t, err)

	// Save public key
	err = SavePublicKeyToConfig("age1newpublickey", "secrets.toml", "secrets.enc.yaml")
	require.NoError(t, err)

	content, err := os.ReadFile(".yewseal.toml")
	require.NoError(t, err)

	// New key should be added after file_path
	assert.Contains(t, string(content), "public_key = \"age1newpublickey\"")
}

func TestSavePublicKeyToConfig_PreservesOtherSections(t *testing.T) {
	oldWd, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	// Create config with additional sections
	existingContent := `[encryption]
input_file = "secrets.toml"
output_file = "secrets.enc.yaml"

[key]
file_path = ".age/keys.txt"

[other_section]
some_value = "test"
`
	err := os.WriteFile(".yewseal.toml", []byte(existingContent), 0644)
	require.NoError(t, err)

	err = SavePublicKeyToConfig("age1testkey", "secrets.toml", "secrets.enc.yaml")
	require.NoError(t, err)

	content, err := os.ReadFile(".yewseal.toml")
	require.NoError(t, err)

	// Other sections should be preserved
	assert.Contains(t, string(content), "[other_section]")
	assert.Contains(t, string(content), "some_value = \"test\"")
	assert.Contains(t, string(content), "public_key = \"age1testkey\"")
}

func TestSavePublicKeyToConfig_NoKeySection(t *testing.T) {
	oldWd, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	// Create config without [key] section
	existingContent := `[encryption]
input_file = "secrets.toml"
output_file = "secrets.enc.yaml"
`
	err := os.WriteFile(".yewseal.toml", []byte(existingContent), 0644)
	require.NoError(t, err)

	// This should not crash, even if it can't add the key properly
	err = SavePublicKeyToConfig("age1testkey", "secrets.toml", "secrets.enc.yaml")
	require.NoError(t, err)

	// File should still be readable
	_, err = os.ReadFile(".yewseal.toml")
	require.NoError(t, err)
}

func TestSavePublicKeyToConfig_SpecialCharactersInPaths(t *testing.T) {
	oldWd, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	err := SavePublicKeyToConfig("age1test", "path/to/secrets.toml", "path/to/secrets.enc.yaml")
	require.NoError(t, err)

	content, err := os.ReadFile(".yewseal.toml")
	require.NoError(t, err)

	assert.Contains(t, string(content), "path/to/secrets.toml")
	assert.Contains(t, string(content), "path/to/secrets.enc.yaml")
}

func TestSavePublicKeyToConfig_ConfigHasCorrectFormat(t *testing.T) {
	oldWd, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	err := SavePublicKeyToConfig("age1testkey", "input.toml", "output.yaml")
	require.NoError(t, err)

	content, err := os.ReadFile(".yewseal.toml")
	require.NoError(t, err)

	// Check it's valid TOML-like structure (has section headers)
	lines := strings.Split(string(content), "\n")
	sectionCount := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			sectionCount++
		}
	}
	assert.GreaterOrEqual(t, sectionCount, 2) // At least [encryption] and [key]
}
