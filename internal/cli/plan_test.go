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

func TestPlanSourceFlagAndEnvironment(t *testing.T) {
	clearCLIEnvironment(t)
	root := t.TempDir()
	configPath := filepath.Join(root, ".yewseal.toml")
	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	defaults := []string{"owner"}
	cfg := &config.Config{
		CurrentDir: root, LoadedFiles: []config.LoadedFile{{Path: configPath, Dir: root}},
		Encryption: config.EncryptionConfig{Files: []config.FilePair{{PlaintextPath: filepath.Join(root, "config.toml"), EncryptedPath: filepath.Join(root, "config.enc.toml"), Format: "toml", ConfigPath: configPath, ConfigDir: root}}},
		Recipients: config.RecipientConfig{Defaults: &defaults, DefaultsConfigPath: configPath,
			Registry: map[string]string{"owner": identity.Recipient().String()}, RegistrySources: map[string]string{"owner": configPath}},
	}
	for _, tc := range []struct {
		name   string
		args   []string
		env    string
		source bool
	}{
		{name: "default", args: []string{"plan"}},
		{name: "flag", args: []string{"plan", "--source"}, source: true},
		{name: "environment", args: []string{"plan"}, env: "true", source: true},
		{name: "flag overrides environment", args: []string{"plan", "--source=false"}, env: "true"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("YEWSEAL_PLAN_SOURCE", tc.env)
			calls := 0
			cmd := newRootCommand("test", func() (*config.Config, error) { calls++; return cfg, nil })
			var output bytes.Buffer
			cmd.SetOut(&output)
			cmd.SetArgs(tc.args)
			require.NoError(t, cmd.Execute())
			require.Equal(t, 1, calls)
			if tc.source {
				require.Contains(t, output.String(), "config.toml -> config.enc.toml  format=toml  aliases=owner")
				require.Contains(t, output.String(), "  authz      .yewseal.toml recipients.defaults")
				require.Contains(t, output.String(), "  registry   owner=.yewseal.toml")
			} else {
				require.Contains(t, output.String(), "Plaintext    Encrypted")
				require.NotContains(t, output.String(), "  authz ")
			}
		})
	}
}
