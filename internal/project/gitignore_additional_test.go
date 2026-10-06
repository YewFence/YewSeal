package project

import (
	"os"
	"path/filepath"
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
	}, t.TempDir())

	assert.Equal(t, []string{"config.toml", ".env"}, files)
}

func TestUpdateGitignoreFiltersPlaintextPathsByDirectory(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "service")
	require.NoError(t, os.Mkdir(dir, 0755))
	withProjectWorkingDir(t, dir)
	require.NoError(t, os.WriteFile(".gitignore", []byte("# Custom rules\nbuild/\n"), 0644))
	pairs := []config.FilePair{
		{PlaintextPath: filepath.Join(dir, "secret.yaml"), EncryptedPath: filepath.Join(root, "outside.enc.yaml")},
		{PlaintextPath: "nested/child.yaml"},
		{PlaintextPath: "secret.yaml"},
		{PlaintextPath: filepath.Join(root, "outside.yaml"), EncryptedPath: filepath.Join(dir, "inside.enc.yaml")},
		{PlaintextPath: "../outside.yaml"},
		{PlaintextPath: filepath.Join(root, "service-other", "outside.yaml")},
	}
	require.NoError(t, UpdateGitignore(pairs))
	data, err := os.ReadFile(".gitignore")
	require.NoError(t, err)
	require.Equal(t, "# Custom rules\nbuild/\n\n"+renderGitignoreSection([]string{"secret.yaml", "nested/child.yaml"}), string(data))
	require.NoError(t, UpdateGitignore(pairs))
	repeated, err := os.ReadFile(".gitignore")
	require.NoError(t, err)
	require.Equal(t, data, repeated)
}

func TestContainsExactLine(t *testing.T) {
	content := "  config.toml  \nconfig.json\n"

	assert.True(t, containsExactLine(content, "config.toml"))
	assert.False(t, containsExactLine(content, "config"))
}
