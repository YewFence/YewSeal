package config

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateFilePairsInfersFormatFromPaths(t *testing.T) {
	pairs, err := ValidateFilePairs([]FilePair{
		{PlaintextPath: "app.toml", EncryptedPath: "app.enc.toml"},
		{PlaintextPath: "secrets", EncryptedPath: "secrets.enc.yaml"},
	})
	require.NoError(t, err)
	require.Len(t, pairs, 2)
	assert.Equal(t, "toml", pairs[0].Format)
	// 明文扩展名无法识别时回退到密文路径推断格式
	assert.Equal(t, "yaml", pairs[1].Format)
}

func TestValidateFilePairsKeepsExplicitFormat(t *testing.T) {
	pairs, err := ValidateFilePairs([]FilePair{
		{PlaintextPath: ".dev.vars", EncryptedPath: ".dev.vars.enc", Format: "env"},
	})
	require.NoError(t, err)
	assert.Equal(t, "env", pairs[0].Format)
}

func TestValidateFilePairsRejectsUndetectableFormat(t *testing.T) {
	_, err := ValidateFilePairs([]FilePair{
		{PlaintextPath: "notes", EncryptedPath: "notes.enc"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "could not detect format")
}

func TestDisplayFilePairsRewritesPathsRelativeToCwd(t *testing.T) {
	root := t.TempDir()
	pairs := DisplayFilePairs([]FilePair{
		{
			PlaintextPath: filepath.Join(root, "sub", "app.toml"),
			EncryptedPath: filepath.Join(root, "sub", "app.toml.enc"),
		},
		{PlaintextPath: "/outside/app.toml", EncryptedPath: "/outside/app.toml.enc"},
	}, root)

	require.Len(t, pairs, 2)
	assert.Equal(t, filepath.Join("sub", "app.toml"), pairs[0].PlaintextPath)
	assert.Equal(t, filepath.Join("sub", "app.toml.enc"), pairs[0].EncryptedPath)
	// cwd 之外的路径保持原样
	assert.Equal(t, "/outside/app.toml", pairs[1].PlaintextPath)
}

func TestCurrentDirPrefersConfiguredValue(t *testing.T) {
	assert.Equal(t, "/repo", CurrentDir(&Config{CurrentDir: "/repo"}))
}

func TestCurrentDirFallsBackToWorkingDirectory(t *testing.T) {
	assert.NotEmpty(t, CurrentDir(nil))
	assert.NotEmpty(t, CurrentDir(&Config{}))
}

func TestRelativePathWithin(t *testing.T) {
	root := t.TempDir()

	rel, ok := RelativePathWithin(root, filepath.Join(root, "sub", "app.toml"))
	require.True(t, ok)
	assert.Equal(t, "sub/app.toml", rel)

	// 相对路径相对 root 解析
	rel, ok = RelativePathWithin(root, "sub/app.toml")
	require.True(t, ok)
	assert.Equal(t, "sub/app.toml", rel)

	// root 之外
	_, ok = RelativePathWithin(root, filepath.Join(root, "..", "other.toml"))
	assert.False(t, ok)

	_, ok = RelativePathWithin(root, "")
	assert.False(t, ok)
}
