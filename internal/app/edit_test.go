package app

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/seal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEditEncryptedFileTOMLRoundTripRemarshal(t *testing.T) {
	skipIfToolMissing(t, "toml2yaml")
	skipIfToolMissing(t, "yaml2toml")

	env := newAppCryptoTestEnv(t)
	plain := []byte(`[database]
host = "localhost"
password = "old"
`)
	require.NoError(t, os.WriteFile("config.toml", plain, 0644))
	require.NoError(t, seal.Encrypt(seal.EncryptOptions{
		InputFile:      "config.toml",
		OutputFile:     "config.enc.toml.yaml",
		KeyFile:        env.keyFile,
		PublicKey:      env.publicKey,
		FormatOverride: "toml",
	}))

	editorPath := filepath.Join(t.TempDir(), "edit.sh")
	editorScript := `#!/usr/bin/env bash
set -euo pipefail
if ! grep -q 'password = "old"' "$1"; then
  echo "expected TOML content in edit buffer" >&2
  exit 1
fi
sed -i 's/password = "old"/password = "new"/' "$1"
`
	require.NoError(t, os.WriteFile(editorPath, []byte(editorScript), 0700))

	var output bytes.Buffer
	require.NoError(t, EditEncryptedFile(EditRequest{
		Config:  config.DefaultConfig(),
		File:    "config.enc.toml.yaml",
		Editor:  editorPath,
		KeyFile: env.keyFile,
		Output:  &output,
	}))

	decrypted, err := seal.DecryptToBytes(seal.DecryptBytesOptions{
		InputFile:      "config.enc.toml.yaml",
		OutputFile:     "config.toml",
		KeyFile:        env.keyFile,
		FormatOverride: "toml",
	})
	require.NoError(t, err)
	assert.Contains(t, string(decrypted), `password = "new"`)
}

func TestEditEncryptedFileUsesConfiguredFormat(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	require.NoError(t, os.WriteFile(".dev.vars", []byte("TOKEN=old\n"), 0644))
	require.NoError(t, seal.Encrypt(seal.EncryptOptions{
		InputFile:      ".dev.vars",
		OutputFile:     ".dev.vars.enc.yaml",
		KeyFile:        env.keyFile,
		PublicKey:      env.publicKey,
		FormatOverride: "env",
	}))

	editorPath := filepath.Join(t.TempDir(), "edit.sh")
	editorScript := `#!/usr/bin/env bash
set -euo pipefail
if ! grep -q 'TOKEN=old' "$1"; then
  echo "expected dotenv content in edit buffer" >&2
  exit 1
fi
sed -i 's/TOKEN=old/TOKEN=new/' "$1"
`
	require.NoError(t, os.WriteFile(editorPath, []byte(editorScript), 0700))

	cfg := &config.Config{
		Encryption: config.EncryptionConfig{
			Files: []config.FilePair{
				{PlaintextPath: ".dev.vars", EncryptedPath: ".dev.vars.enc.yaml", Format: "env"},
			},
		},
	}

	require.NoError(t, EditEncryptedFile(EditRequest{
		Config:  cfg,
		File:    ".dev.vars.enc.yaml",
		Editor:  editorPath,
		KeyFile: env.keyFile,
	}))

	decrypted, err := seal.DecryptToBytes(seal.DecryptBytesOptions{
		InputFile:      ".dev.vars.enc.yaml",
		OutputFile:     ".dev.vars",
		KeyFile:        env.keyFile,
		FormatOverride: "env",
	})
	require.NoError(t, err)
	assert.Equal(t, "TOKEN=new\n", string(decrypted))
}

func TestSplitEditorCommandPreservesQuotedExecutable(t *testing.T) {
	parts, err := splitEditorCommand(`"/Applications/Sublime Text.app/Contents/SharedSupport/bin/subl" -w`)
	require.NoError(t, err)
	assert.Equal(t, []string{"/Applications/Sublime Text.app/Contents/SharedSupport/bin/subl", "-w"}, parts)
}

func TestResolveEncryptedTarget(t *testing.T) {
	cfg := &config.Config{
		Encryption: config.EncryptionConfig{
			Files: []config.FilePair{
				{PlaintextPath: ".dev.vars", EncryptedPath: ".dev.vars.enc.yaml", Format: "env"},
			},
		},
	}

	configured, err := ResolveEncryptedTarget(cfg, ".dev.vars.enc.yaml", "edit")
	require.NoError(t, err)
	assert.Equal(t, ".dev.vars", configured.PlaintextPath)
	assert.Equal(t, ".dev.vars.enc.yaml", configured.EncryptedPath)
	assert.Equal(t, "env", configured.FormatOverride)
	assert.Equal(t, "env", configured.Format)

	inferred, err := ResolveEncryptedTarget(config.DefaultConfig(), "config.enc.toml.yaml", "edit")
	require.NoError(t, err)
	assert.Equal(t, "config.toml", inferred.PlaintextPath)
	assert.Equal(t, "config.enc.toml.yaml", inferred.EncryptedPath)
	assert.Equal(t, "toml", inferred.FormatOverride)
	assert.Equal(t, "toml", inferred.Format)
}

func skipIfToolMissing(t *testing.T, tool string) {
	t.Helper()

	if _, err := os.Stat(tool); err == nil {
		return
	}
	if _, err := os.Stat(filepath.Join("/usr/bin", tool)); err == nil {
		return
	}
	if _, err := os.Stat(filepath.Join("/usr/local/bin", tool)); err == nil {
		return
	}
	if _, err := exec.LookPath(tool); err != nil {
		t.Skipf("Skipping test: %s not found", tool)
	}
}
