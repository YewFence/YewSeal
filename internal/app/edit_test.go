package app

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

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
		OutputFile:     "config.enc.toml",
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
	assert.Contains(t, string(decrypted), `password = "new"`)
}

func TestSplitEditorCommandPreservesQuotedExecutable(t *testing.T) {
	parts, err := splitEditorCommand(`"/Applications/Sublime Text.app/Contents/SharedSupport/bin/subl" -w`)
	require.NoError(t, err)
	assert.Equal(t, []string{"/Applications/Sublime Text.app/Contents/SharedSupport/bin/subl", "-w"}, parts)
}

func TestPlaintextFormatPathForEdit(t *testing.T) {
	assert.Equal(t, "config.toml", plaintextFormatPathForEdit("config.enc.toml.yaml"))
	assert.Equal(t, filepath.Join("dir", "config.yaml"), plaintextFormatPathForEdit(filepath.Join("dir", "config.enc.yaml")))
	assert.Equal(t, ".dev.vars.yaml", plaintextFormatPathForEdit(".dev.vars.enc.yaml"))
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
