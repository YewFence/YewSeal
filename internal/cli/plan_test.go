package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"filippo.io/age"
	"github.com/YewFence/YewSeal/internal/config"
	"github.com/stretchr/testify/require"
)

func TestPlanChecksMappingsWithoutExecutionEnvironment(t *testing.T) {
	clearCLIEnvironment(t)
	t.Setenv("YEWSEAL_ENCRYPT_OUTPUT", "ignored-output.json")
	t.Setenv("SOPS_AGE_KEY_CMD", "exit 29")
	t.Setenv("YEWSEAL_DECRYPT_STRICT", "invalid")
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "remote.enc.yaml"), []byte("not valid ciphertext"), 0600))
	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	defaults := []string{"owner"}
	cfg := &config.Config{
		CurrentDir: root,
		Recipients: config.RecipientConfig{Defaults: &defaults, Registry: map[string]string{"owner": identity.Recipient().String()}},
		Encryption: config.EncryptionConfig{Groups: []config.GroupConfig{{ConfigDir: root, Patterns: []string{"*.yaml"}}}},
	}
	for _, args := range [][]string{{"plan", "--json"}, {"plan", "remote.enc.yaml", "--json"}, {"plan", root, "--json"}} {
		calls := 0
		cmd := newRootCommand("test", func() (*config.Config, error) { calls++; return cfg, nil })
		var output bytes.Buffer
		cmd.SetOut(&output)
		cmd.SetArgs(args)
		require.NoError(t, cmd.Execute())
		require.Equal(t, 1, calls)
		var result struct {
			Command   string            `json:"command"`
			Metadata  json.RawMessage   `json:"metadata"`
			FilePairs []json.RawMessage `json:"file_pairs"`
		}
		require.NoError(t, json.Unmarshal(output.Bytes(), &result))
		require.Equal(t, "plan", result.Command)
		require.Nil(t, result.Metadata, "plan does not describe metadata writes")
		require.Len(t, result.FilePairs, 1)
		require.NotContains(t, output.String(), "ignored-output")
	}
	entries, err := os.ReadDir(root)
	require.NoError(t, err)
	require.Len(t, entries, 1, "plan must not write plaintext or metadata")
}
