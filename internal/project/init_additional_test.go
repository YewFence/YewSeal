package project

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mockProjectInput(t *testing.T, input string) {
	t.Helper()

	r, w, err := os.Pipe()
	require.NoError(t, err)

	oldStdin := os.Stdin
	os.Stdin = r
	t.Cleanup(func() {
		os.Stdin = oldStdin
	})

	go func() {
		_, _ = w.Write([]byte(input))
		_ = w.Close()
	}()
}

func TestConfirmInitOverwrite_Force(t *testing.T) {
	allowed, err := confirmInitOverwrite(true, false)
	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestConfirmInitOverwrite_NoExistingConfig(t *testing.T) {
	tempDir := t.TempDir()
	withProjectWorkingDir(t, tempDir)

	allowed, err := confirmInitOverwrite(false, false)
	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestDetectInitFormat(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{name: "toml", filename: "app.toml", want: "toml"},
		{name: "yaml", filename: "config.yaml", want: "yaml"},
		{name: "yml", filename: "config.yml", want: "yaml"},
		{name: "json", filename: "config.json", want: "json"},
		{name: "env", filename: "config.env", want: "env"},
		{name: "ini", filename: "config.ini", want: "ini"},
		{name: "binary", filename: "secret.bin", want: "binary"},
		{name: "unknown", filename: ".dev.vars", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, detectInitFormat(tt.filename))
		})
	}
}

func TestNormalizeInitFormat(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
		valid bool
	}{
		{name: "toml", input: "toml", want: "toml", valid: true},
		{name: "yaml alias", input: "YML", want: "yaml", valid: true},
		{name: "dotenv alias", input: " dotenv ", want: "env", valid: true},
		{name: "json", input: "json", want: "json", valid: true},
		{name: "ini", input: "ini", want: "ini", valid: true},
		{name: "binary alias", input: "bin", want: "binary", valid: true},
		{name: "invalid", input: "xml", want: "", valid: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := normalizeInitFormat(tt.input)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.valid, ok)
		})
	}
}

func TestResolveInitFormatOverride(t *testing.T) {
	t.Run("normalizes provided override", func(t *testing.T) {
		format, err := resolveInitFormatOverride(".dev.vars", "dotenv", false)
		require.NoError(t, err)
		assert.Equal(t, "env", format)
	})

	t.Run("rejects unsupported explicit override", func(t *testing.T) {
		format, err := resolveInitFormatOverride("config.yaml", "xml", false)
		assert.Empty(t, format)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `unsupported format override "xml"`)
	})

	t.Run("returns empty override when format is auto detected", func(t *testing.T) {
		format, err := resolveInitFormatOverride("config.yaml", "", false)
		require.NoError(t, err)
		assert.Empty(t, format)
	})

	t.Run("rejects ambiguous format in non-interactive mode", func(t *testing.T) {
		format, err := resolveInitFormatOverride(".dev.vars", "", false)
		assert.Empty(t, format)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "please pass --format")
		assert.Contains(t, err.Error(), "Hint: pass --format binary")
	})

	t.Run("prompts when format is ambiguous and interactive", func(t *testing.T) {
		mockProjectInput(t, "env\n")
		format, err := resolveInitFormatOverride(".dev.vars", "", true)
		require.NoError(t, err)
		assert.Equal(t, "env", format)
	})
}

func TestNewInitFilePair(t *testing.T) {
	t.Run("uses default encrypted filename", func(t *testing.T) {
		filePair, err := newInitFilePair("app.toml", "", "", false)
		require.NoError(t, err)
		assert.Equal(t, "app.toml", filePair.PlaintextPath)
		assert.Equal(t, "app.enc.toml", filePair.EncryptedPath)
		assert.Empty(t, filePair.Format)
	})

	t.Run("uses provided encrypted filename and normalized format", func(t *testing.T) {
		filePair, err := newInitFilePair(".dev.vars", "secret.enc.yaml", "dotenv", false)
		require.NoError(t, err)
		assert.Equal(t, ".dev.vars", filePair.PlaintextPath)
		assert.Equal(t, "secret.enc.yaml", filePair.EncryptedPath)
		assert.Equal(t, "env", filePair.Format)
	})
}

func TestExtractPublicKey(t *testing.T) {
	t.Run("extracts key from generated output", func(t *testing.T) {
		output := "# created: 2026-04-05T00:00:00Z\n# public key: age1testpublickey\nAGE-SECRET-KEY-1TEST\n"
		assert.Equal(t, "age1testpublickey", extractPublicKey(output))
	})

	t.Run("returns empty when key line is missing", func(t *testing.T) {
		assert.Empty(t, extractPublicKey("AGE-SECRET-KEY-1TEST\n"))
	})
}

func TestSetupAgeKey_ForceRemovesPreviousKeyFile(t *testing.T) {
	tempDir := t.TempDir()
	withProjectWorkingDir(t, tempDir)

	require.NoError(t, os.MkdirAll(".age", 0o700))
	require.NoError(t, os.WriteFile(".age/keys.txt", []byte("stale"), 0o600))

	publicKey, err := setupAgeKey(true)
	require.NoError(t, err)
	assert.NotEmpty(t, publicKey)

	content, err := os.ReadFile(".age/keys.txt")
	require.NoError(t, err)
	assert.Contains(t, string(content), "# public key: ")
	assert.NotContains(t, string(content), "stale")
}

func TestInitProject_NonInteractiveSkipSopsConfig(t *testing.T) {
	tempDir := t.TempDir()
	withProjectWorkingDir(t, tempDir)

	require.NoError(t, os.WriteFile("app.toml", []byte("[app]\nname = \"demo\"\n"), 0o644))

	err := InitProject(false, "app.toml", "", "", true, true)
	require.NoError(t, err)

	_, err = os.Stat(".age/keys.txt")
	assert.NoError(t, err)
	_, err = os.Stat(".yewseal.toml")
	assert.NoError(t, err)
	_, err = os.Stat(".gitignore")
	assert.NoError(t, err)
	_, err = os.Stat("app.example.toml")
	assert.NoError(t, err)
	_, err = os.Stat(".sops.yaml")
	assert.True(t, os.IsNotExist(err))
}

func TestInitProject_NonInteractiveCreatesSopsConfig(t *testing.T) {
	tempDir := t.TempDir()
	withProjectWorkingDir(t, tempDir)
	mockProjectInput(t, "\n")

	require.NoError(t, os.WriteFile("config.toml", []byte("[service]\nport = 8080\n"), 0o644))

	err := InitProject(false, "config.toml", "", "", false, false)
	require.NoError(t, err)

	_, err = os.Stat(".sops.yaml")
	assert.NoError(t, err)
	_, err = os.Stat(".yewseal.toml")
	assert.NoError(t, err)
	_, err = os.Stat("config.example.toml")
	assert.True(t, os.IsNotExist(err))
}
