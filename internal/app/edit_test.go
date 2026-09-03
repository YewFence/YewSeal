package app

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/seal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEditEncryptedFileTOMLRoundTrip(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	plain := []byte(`[database]
host = "localhost"
password = "old"
`)
	require.NoError(t, os.WriteFile("config.toml", plain, 0644))
	require.NoError(t, seal.Encrypt(seal.EncryptOptions{
		InputFile:      "config.toml",
		OutputFile:     "config.enc.toml",
		Recipients:     []string{env.publicKey},
		FormatOverride: "toml",
	}))

	editorPath := filepath.Join(t.TempDir(), "edit.sh")
	editorScript := `#!/usr/bin/env bash
set -euo pipefail
if ! grep -q "password = 'old'" "$1"; then
  echo "expected TOML content in edit buffer" >&2
  exit 1
fi
sed -i "s/password = 'old'/password = 'new'/" "$1"
`
	require.NoError(t, os.WriteFile(editorPath, []byte(editorScript), 0700))

	var output bytes.Buffer
	require.NoError(t, EditEncryptedFile(EditRequest{
		Config:  &config.Config{Encryption: config.EncryptionConfig{Files: []config.FilePair{{PlaintextPath: "config.toml", EncryptedPath: "config.enc.toml", Format: "toml"}}}},
		File:    "config.enc.toml",
		Editor:  editorPath,
		KeyFile: env.keyFile,
		Output:  &output,
	}))

	decrypted, err := seal.DecryptToBytes(seal.DecryptBytesOptions{
		InputFile:      "config.enc.toml",
		OutputFile:     "config.toml",
		KeyFile:        env.keyFile,
		FormatOverride: "toml",
	})
	require.NoError(t, err)
	assert.Contains(t, string(decrypted), `password = 'new'`)
}

func TestEditEncryptedFileUsesConfiguredFormat(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	require.NoError(t, os.WriteFile(".dev.vars", []byte("TOKEN=old\n"), 0644))
	require.NoError(t, seal.Encrypt(seal.EncryptOptions{
		InputFile:      ".dev.vars",
		OutputFile:     ".dev.vars.enc.yaml",
		Recipients:     []string{env.publicKey},
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

func TestEditEncryptedFileRequiresConfiguredTarget(t *testing.T) {
	err := EditEncryptedFile(EditRequest{Config: &config.Config{}})
	require.EqualError(t, err, "edit requires exactly one configured target")
}

func TestSplitEditorCommandPreservesQuotedExecutable(t *testing.T) {
	parts, err := splitEditorCommand(`"/Applications/Sublime Text.app/Contents/SharedSupport/bin/subl" -w`)
	require.NoError(t, err)
	assert.Equal(t, []string{"/Applications/Sublime Text.app/Contents/SharedSupport/bin/subl", "-w"}, parts)
}
