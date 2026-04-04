package crypto

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateSopsYaml_CreateNew(t *testing.T) {
	// Change to temp directory
	oldWd, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	err := UpdateSopsYaml("secrets.enc.yaml", "age1testpublickey123", false)
	require.NoError(t, err)

	// Verify file was created
	content, err := os.ReadFile(".sops.yaml")
	require.NoError(t, err)

	// Check content
	assert.Contains(t, string(content), "creation_rules:")
	assert.Contains(t, string(content), "path_regex:")
	assert.Contains(t, string(content), "secrets\\.enc\\.yaml")
	assert.Contains(t, string(content), "age1testpublickey123")
}

func TestUpdateSopsYaml_AppendRule(t *testing.T) {
	oldWd, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	// Create first rule
	err := UpdateSopsYaml("first.enc.yaml", "age1first", false)
	require.NoError(t, err)

	// Append second rule
	err = UpdateSopsYaml("second.enc.yaml", "age1second", false)
	require.NoError(t, err)

	content, err := os.ReadFile(".sops.yaml")
	require.NoError(t, err)

	// Both rules should exist
	assert.Contains(t, string(content), "first\\.enc\\.yaml")
	assert.Contains(t, string(content), "second\\.enc\\.yaml")
	assert.Contains(t, string(content), "age1first")
	assert.Contains(t, string(content), "age1second")
}

func TestUpdateSopsYaml_IdempotentRule(t *testing.T) {
	oldWd, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	// Create rule
	err := UpdateSopsYaml("secrets.enc.yaml", "age1test", false)
	require.NoError(t, err)

	contentBefore, _ := os.ReadFile(".sops.yaml")

	// Try to create the same rule again
	err = UpdateSopsYaml("secrets.enc.yaml", "age1test", false)
	require.NoError(t, err)

	contentAfter, _ := os.ReadFile(".sops.yaml")

	// Content should be the same (idempotent)
	assert.Equal(t, string(contentBefore), string(contentAfter))
}

func TestUpdateSopsYaml_ForceReplace(t *testing.T) {
	oldWd, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	// Create first rule
	err := UpdateSopsYaml("first.enc.yaml", "age1first", false)
	require.NoError(t, err)

	// Force replace with new rule
	err = UpdateSopsYaml("new.enc.yaml", "age1new", true)
	require.NoError(t, err)

	content, err := os.ReadFile(".sops.yaml")
	require.NoError(t, err)

	// Only the new rule should exist
	assert.NotContains(t, string(content), "first\\.enc\\.yaml")
	assert.Contains(t, string(content), "new\\.enc\\.yaml")
	assert.Contains(t, string(content), "age1new")
}

func TestUpdateSopsYaml_SpecialCharactersInPath(t *testing.T) {
	oldWd, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	// Path with special regex characters
	err := UpdateSopsYaml("path.with.dots.yaml", "age1test", false)
	require.NoError(t, err)

	content, err := os.ReadFile(".sops.yaml")
	require.NoError(t, err)

	// Dots should be escaped in regex
	assert.Contains(t, string(content), "path\\.with\\.dots\\.yaml")
}

func TestUpdateSopsYaml_InvalidExistingYaml(t *testing.T) {
	oldWd, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	// Create invalid YAML file
	err := os.WriteFile(".sops.yaml", []byte("invalid: yaml: content: ["), 0644)
	require.NoError(t, err)

	err = UpdateSopsYaml("secrets.enc.yaml", "age1test", false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse existing .sops.yaml")
}

func TestSopsConfig_Struct(t *testing.T) {
	cfg := SopsConfig{
		CreationRules: []CreationRule{
			{PathRegex: "^test\\.yaml$", Age: "age1test"},
		},
	}

	assert.Len(t, cfg.CreationRules, 1)
	assert.Equal(t, "^test\\.yaml$", cfg.CreationRules[0].PathRegex)
	assert.Equal(t, "age1test", cfg.CreationRules[0].Age)
}

func TestUpdateSopsYaml_SubdirectoryPath(t *testing.T) {
	oldWd, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	err := UpdateSopsYaml(filepath.Join("config", "secrets.enc.yaml"), "age1test", false)
	require.NoError(t, err)

	content, err := os.ReadFile(".sops.yaml")
	require.NoError(t, err)

	// Path should be properly escaped
	assert.Contains(t, string(content), "age1test")
}

func TestSyncSopsYaml_ReplacesWithConfiguredRules(t *testing.T) {
	oldWd, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	err := UpdateSopsYaml("legacy.enc.yaml", "age1legacy", false)
	require.NoError(t, err)

	err = SyncSopsYaml([]config.FilePair{
		{Dec: "app.toml", Enc: "app.enc.toml.yaml"},
		{Dec: ".dev.vars", Enc: ".dev.vars.enc.yaml"},
	}, "age1test")
	require.NoError(t, err)

	content, err := os.ReadFile(".sops.yaml")
	require.NoError(t, err)

	assert.NotContains(t, string(content), "legacy\\.enc\\.yaml")
	assert.Contains(t, string(content), "app\\.enc\\.toml\\.yaml")
	assert.Contains(t, string(content), "\\.dev\\.vars\\.enc\\.yaml")
	assert.Contains(t, string(content), "age1test")
}

func TestSyncSopsYaml_DeduplicatesEncryptedFiles(t *testing.T) {
	oldWd, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	err := SyncSopsYaml([]config.FilePair{
		{Dec: "app.toml", Enc: "shared.enc.yaml"},
		{Dec: "db.toml", Enc: "shared.enc.yaml"},
	}, "age1test")
	require.NoError(t, err)

	content, err := os.ReadFile(".sops.yaml")
	require.NoError(t, err)

	assert.Equal(t, 1, strings.Count(string(content), "shared\\.enc\\.yaml"))
}
