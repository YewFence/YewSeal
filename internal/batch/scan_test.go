package batch

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanPathProtocol(t *testing.T) {
	encryptTests := []struct {
		input    string
		format   string
		expected string
	}{
		{input: "config.toml", expected: "config.enc.toml.yaml"},
		{input: "config.yaml", expected: "config.enc.yaml"},
		{input: "config.yml", expected: "config.enc.yaml"},
		{input: "config.json", expected: "config.enc.json"},
		{input: "config.env", expected: "config.enc.env"},
		{input: "config.ini", expected: "config.enc.ini"},
		{input: "config.bin", expected: "config.enc.bin"},
		{input: "config.binary", expected: "config.enc.bin"},
		{input: ".dev.vars", format: "env", expected: ".dev.vars.enc.env"},
	}

	for _, tt := range encryptTests {
		t.Run("encrypt_"+tt.input, func(t *testing.T) {
			actual, err := EncryptPathForPlaintext(tt.input, tt.format)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, actual)
		})
	}

	decryptTests := []struct {
		input    string
		expected string
		format   string
	}{
		{input: "config.enc.toml.yaml", expected: "config.toml", format: "toml"},
		{input: "config.enc.yaml", expected: "config.yaml", format: "yaml"},
		{input: "config.enc.json", expected: "config.json", format: "json"},
		{input: "config.enc.env", expected: "config.env", format: "env"},
		{input: "config.enc.ini", expected: "config.ini", format: "ini"},
		{input: "config.enc.bin", expected: "config.bin", format: "binary"},
	}

	for _, tt := range decryptTests {
		t.Run("decrypt_"+tt.input, func(t *testing.T) {
			actual, format, err := PlaintextPathForEncrypted(tt.input, "")
			require.NoError(t, err)
			assert.Equal(t, tt.expected, actual)
			assert.Equal(t, tt.format, format)
		})
	}
}

func TestPlaintextPathForEncryptedRequiresProtocolWithoutFormatRule(t *testing.T) {
	_, _, err := PlaintextPathForEncrypted("secret.sops", "")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not follow the yewseal scan protocol")
}

func TestBuildScanFilePairsUsesPatternsAndFormatRules(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "app.toml"), []byte("token = \"secret\"\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "app.example.toml"), []byte("token = \"example\"\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".dev.vars"), []byte("TOKEN=secret\n"), 0644))

	pairs, err := BuildScanFilePairs(ScanOptions{
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

func TestBuildProjectScanFilePairsDecryptUsesPlaintextPatterns(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".dev.vars.enc.env"), []byte("encrypted"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "app.example.enc.toml.yaml"), []byte("encrypted"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "app.enc.toml.yaml"), []byte("encrypted"), 0644))

	pairs, err := BuildProjectScanFilePairs(ScanOptions{
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

func TestBuildScanFilePairsAllowsUnknownAsBinary(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "secret.data"), []byte("secret"), 0644))

	pairs, err := BuildScanFilePairs(ScanOptions{
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
