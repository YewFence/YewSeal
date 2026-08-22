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
		KeyFile:        env.keyFile,
		PublicKey:      env.publicKey,
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
		Config:  config.DefaultConfig(),
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

	inferred, err := ResolveEncryptedTarget(config.DefaultConfig(), "config.enc.toml", "edit")
	require.NoError(t, err)
	assert.Equal(t, "config.toml", inferred.PlaintextPath)
	assert.Equal(t, "config.enc.toml", inferred.EncryptedPath)
	assert.Equal(t, "toml", inferred.FormatOverride)
	assert.Equal(t, "toml", inferred.Format)
}

func TestResolveEncryptedTargetWithOverrides_OutputOverridesConfiguredPlaintextPath(t *testing.T) {
	cfg := &config.Config{
		Encryption: config.EncryptionConfig{
			Files: []config.FilePair{
				{PlaintextPath: ".dev.vars", EncryptedPath: ".dev.vars.enc.yaml", Format: "env"},
			},
		},
	}

	target, err := ResolveEncryptedTargetWithOverrides(cfg, ".dev.vars.enc.yaml", "decrypt", "", "local.env")
	require.NoError(t, err)
	assert.Equal(t, "local.env", target.PlaintextPath)
	assert.Equal(t, ".dev.vars.enc.yaml", target.EncryptedPath)
	assert.Equal(t, "env", target.FormatOverride)
	assert.Equal(t, "env", target.Format)
}

func TestResolveEncryptedTargetWithOverrides_OutputDoesNotChangeConfiguredFormat(t *testing.T) {
	cfg := &config.Config{
		Encryption: config.EncryptionConfig{
			Files: []config.FilePair{
				{PlaintextPath: "config.yaml", EncryptedPath: "config.enc.yaml"},
			},
		},
	}

	target, err := ResolveEncryptedTargetWithOverrides(cfg, "config.enc.yaml", "decrypt", "", "custom")
	require.NoError(t, err)
	assert.Equal(t, "custom", target.PlaintextPath)
	assert.Equal(t, "config.enc.yaml", target.EncryptedPath)
	assert.Equal(t, "yaml", target.FormatOverride)
	assert.Equal(t, "yaml", target.Format)
}

func TestResolveEncryptedTargetWithOverrides_AcceptsConfiguredPlaintextSide(t *testing.T) {
	cfg := &config.Config{
		Encryption: config.EncryptionConfig{
			Files: []config.FilePair{
				{PlaintextPath: ".dev.vars", EncryptedPath: ".dev.vars.enc.yaml", Format: "env"},
			},
		},
	}

	target, err := ResolveEncryptedTargetWithOverrides(cfg, ".dev.vars", "decrypt", "", "")
	require.NoError(t, err)
	assert.Equal(t, ".dev.vars", target.PlaintextPath)
	assert.Equal(t, ".dev.vars.enc.yaml", target.EncryptedPath)
	assert.Equal(t, "env", target.FormatOverride)
	assert.Equal(t, "env", target.Format)
}

func TestResolveEncryptedTargetWithOverrides_FormatOverridesProtocol(t *testing.T) {
	target, err := ResolveEncryptedTargetWithOverrides(config.DefaultConfig(), "config.enc.yaml", "decrypt", "toml", "")
	require.NoError(t, err)
	assert.Equal(t, "config.toml", target.PlaintextPath)
	assert.Equal(t, "config.enc.yaml", target.EncryptedPath)
	assert.Equal(t, "toml", target.FormatOverride)
	assert.Equal(t, "toml", target.Format)
}

func TestResolveEncryptedTargetWithOverrides_OutputDoesNotChangeProtocolFormat(t *testing.T) {
	target, err := ResolveEncryptedTargetWithOverrides(config.DefaultConfig(), "config.enc.yaml", "decrypt", "", "custom")
	require.NoError(t, err)
	assert.Equal(t, "custom", target.PlaintextPath)
	assert.Equal(t, "config.enc.yaml", target.EncryptedPath)
	assert.Equal(t, "yaml", target.FormatOverride)
	assert.Equal(t, "yaml", target.Format)
}
