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

	err := UpdateGitignore(".", []config.FilePair{
		{PlaintextPath: "   "},
		{},
	})
	require.NoError(t, err)

	data, err := os.ReadFile(".gitignore")
	require.NoError(t, err)
	assert.Equal(t, renderGitignoreSection(nil), string(data))
}

func TestMergeGitignoreEntriesPreservesLegacySection(t *testing.T) {
	content := "# YewSeal - Decrypted configuration files\nconfig.toml\n\n# YewSeal - Age private keys\n.age/keys.txt\n"
	updated, err := mergeGitignoreEntries(content, []string{"other.toml"})
	require.NoError(t, err)
	assert.Equal(t, content+"\n"+renderGitignoreSection([]string{"other.toml"}), updated)
}

func TestRenderGitignoreSection(t *testing.T) {
	rendered := renderGitignoreSection([]string{"config.toml", ".env"})

	assert.Equal(
		t,
		"# BEGIN YewSeal managed entries\n/config.toml\n/.env\n/.age/keys.txt\n# END YewSeal managed entries\n",
		rendered,
	)
}

func TestUniquePlaintextFiles(t *testing.T) {
	files, err := uniquePlaintextFiles([]config.FilePair{
		{PlaintextPath: "config.toml"},
		{PlaintextPath: "config.toml"},
		{PlaintextPath: ".env"},
		{PlaintextPath: "   "},
	}, t.TempDir())

	require.NoError(t, err)
	assert.Equal(t, []string{".env", "config.toml"}, files)
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
	require.NoError(t, UpdateGitignore(".", pairs))
	data, err := os.ReadFile(".gitignore")
	require.NoError(t, err)
	require.Equal(t, "# Custom rules\nbuild/\n\n"+renderGitignoreSection([]string{"nested/child.yaml", "secret.yaml"}), string(data))
	require.NoError(t, UpdateGitignore(".", pairs))
	repeated, err := os.ReadFile(".gitignore")
	require.NoError(t, err)
	require.Equal(t, data, repeated)
}
