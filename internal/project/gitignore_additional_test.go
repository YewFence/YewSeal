package project

import (
	"os"
	"testing"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withProjectWorkingDir(t *testing.T, dir string) {
	t.Helper()

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() {
		_ = os.Chdir(oldWd)
	})
}

func TestUpdateGitignore_NoPlaintextFiles(t *testing.T) {
	tempDir := t.TempDir()
	withProjectWorkingDir(t, tempDir)

	err := UpdateGitignore([]config.FilePair{
		{PlaintextPath: "   "},
		{},
	})
	require.NoError(t, err)

	_, statErr := os.Stat(".gitignore")
	assert.True(t, os.IsNotExist(statErr))
}

func TestMergeGitignoreEntries_AddsPrivateKeySectionWhenMissing(t *testing.T) {
	content := decryptedFilesHeader + "\nconfig.toml\n"

	updated, changed := mergeGitignoreEntries(content, []string{"other.toml"})

	assert.True(t, changed)
	assert.Equal(
		t,
		decryptedFilesHeader+"\nconfig.toml\nother.toml\n\n"+privateKeysHeader+"\n"+privateKeyPath+"\n",
		updated,
	)
}

func TestRenderGitignoreSection(t *testing.T) {
	rendered := renderGitignoreSection([]string{"config.toml", ".env"})

	assert.Equal(
		t,
		decryptedFilesHeader+"\nconfig.toml\n.env\n\n"+privateKeysHeader+"\n"+privateKeyPath+"\n",
		rendered,
	)
}

func TestUniquePlaintextFiles(t *testing.T) {
	files := uniquePlaintextFiles([]config.FilePair{
		{PlaintextPath: " config.toml "},
		{PlaintextPath: "config.toml"},
		{PlaintextPath: ".env"},
		{PlaintextPath: "   "},
	})

	assert.Equal(t, []string{"config.toml", ".env"}, files)
}

func TestContainsExactLine(t *testing.T) {
	content := "  config.toml  \nconfig.json\n"

	assert.True(t, containsExactLine(content, "config.toml"))
	assert.False(t, containsExactLine(content, "config"))
}
