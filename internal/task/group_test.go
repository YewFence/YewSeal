package task

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildGroupFilePairsUsesPatternsAndFormatRules(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "app.toml"), []byte("token = \"secret\"\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "app.example.toml"), []byte("token = \"example\"\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".dev.vars"), []byte("TOKEN=secret\n"), 0644))

	pairs, err := BuildGroupFilePairs(GroupOptions{
		Root: dir,
		Patterns: []string{
			"*.toml",
			".dev.vars",
			"!*.example.toml",
		},
		FormatRules: []string{".dev.vars=env"},
		Mode:        ModeEncrypt,
	})
	require.NoError(t, err)
	require.Len(t, pairs, 2)

	assert.Equal(t, filepath.Join(dir, ".dev.vars"), pairs[0].PlaintextPath)
	assert.Equal(t, filepath.Join(dir, ".dev.vars.enc.env"), pairs[0].EncryptedPath)
	assert.Equal(t, "env", pairs[0].Format)
	assert.Equal(t, filepath.Join(dir, "app.toml"), pairs[1].PlaintextPath)
	assert.Equal(t, filepath.Join(dir, "app.enc.toml.yaml"), pairs[1].EncryptedPath)
	assert.Equal(t, "toml", pairs[1].Format)
}

func TestBuildProjectGroupFilePairsDecryptUsesPlaintextPatterns(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".dev.vars.enc.env"), []byte("encrypted"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "app.example.enc.toml.yaml"), []byte("encrypted"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "app.enc.toml.yaml"), []byte("encrypted"), 0644))

	pairs, err := BuildProjectGroupFilePairs(GroupOptions{
		Root: dir,
		Patterns: []string{
			"*.toml",
			".dev.vars",
			"!*.example.toml",
		},
		FormatRules: []string{".dev.vars=env"},
		Mode:        ModeDecrypt,
	})
	require.NoError(t, err)
	require.Len(t, pairs, 2)

	assert.Equal(t, filepath.Join(dir, ".dev.vars"), pairs[0].PlaintextPath)
	assert.Equal(t, filepath.Join(dir, ".dev.vars.enc.env"), pairs[0].EncryptedPath)
	assert.Equal(t, "env", pairs[0].Format)
	assert.Equal(t, filepath.Join(dir, "app.toml"), pairs[1].PlaintextPath)
	assert.Equal(t, filepath.Join(dir, "app.enc.toml.yaml"), pairs[1].EncryptedPath)
	assert.Equal(t, "toml", pairs[1].Format)
}

func TestBuildGroupFilePairsAllowsUnknownAsBinary(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "secret.data"), []byte("secret"), 0644))

	pairs, err := BuildGroupFilePairs(GroupOptions{
		Root:            dir,
		Patterns:        []string{"*.data"},
		UnknownAsBinary: true,
		Mode:            ModeEncrypt,
	})
	require.NoError(t, err)
	require.Len(t, pairs, 1)

	assert.Equal(t, "binary", pairs[0].Format)
	assert.Equal(t, filepath.Join(dir, "secret.data.enc.bin"), pairs[0].EncryptedPath)
}

func TestBuildGroupFilePairsDirectoryPatternIncludesDescendants(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "secrets"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "secrets", "app.toml"), []byte("token = \"secret\"\n"), 0644))

	pairs, err := BuildGroupFilePairs(GroupOptions{
		Root:     dir,
		Patterns: []string{"secrets/"},
		Mode:     ModeEncrypt,
	})
	require.NoError(t, err)
	require.Len(t, pairs, 1)

	assert.Equal(t, filepath.Join(dir, "secrets", "app.toml"), pairs[0].PlaintextPath)
}

func TestBuildGroupFilePairsSkipsExcludedDirectories(t *testing.T) {
	dir := t.TempDir()
	ignored := filepath.Join(dir, "ignored")
	require.NoError(t, os.MkdirAll(ignored, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "app.toml"), []byte("token = \"secret\"\n"), 0644))
	require.NoError(t, os.Chmod(ignored, 0000))
	t.Cleanup(func() {
		_ = os.Chmod(ignored, 0755)
	})

	pairs, err := BuildGroupFilePairs(GroupOptions{
		Root: dir,
		Patterns: []string{
			"*.toml",
			"!ignored/",
		},
		Mode: ModeEncrypt,
	})
	require.NoError(t, err)
	require.Len(t, pairs, 1)

	assert.Equal(t, filepath.Join(dir, "app.toml"), pairs[0].PlaintextPath)
}
