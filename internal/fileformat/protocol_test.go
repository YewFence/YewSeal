package fileformat

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroupPathProtocol(t *testing.T) {
	encryptTests := []struct {
		input    string
		format   string
		expected string
	}{
		{input: "config.toml", expected: "config.enc.toml"},
		{input: "config.yaml", expected: "config.enc.yaml"},
		{input: "config.yml", expected: "config.enc.yaml"},
		{input: "config.json", expected: "config.enc.json"},
		{input: "config.env", expected: "config.enc.env"},
		{input: "config.ini", expected: "config.enc.ini"},
		{input: "config.bin", expected: "config.enc.bin"},
		{input: "config.binary", expected: "config.enc.bin"},
		{input: ".dev.vars", format: "env", expected: ".dev.vars.enc.env"},
		{input: "secret", format: "env", expected: "secret.enc.env"},
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
		{input: "config.enc.toml", expected: "config.toml", format: "toml"},
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
	assert.Contains(t, err.Error(), "does not follow the yewseal group protocol")
}
