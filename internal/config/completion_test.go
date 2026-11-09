package config

import (
	"os"
	"path/filepath"
	"testing"

	"filippo.io/age"
	"github.com/YewFence/YewSeal/internal/task"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompletionPathsListsCommandSidesWithoutDuplicates(t *testing.T) {
	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	root := t.TempDir()
	configPath := filepath.Join(root, ".yewseal.toml")
	defaults := []string{"owner"}
	// The second pair's plaintext is the first pair's ciphertext: plan adds
	// both sides of every mapping, so that path would be listed twice.
	cfg := &Config{
		CurrentDir: root, ProjectRoot: root,
		Recipients: RecipientConfig{Defaults: &defaults, Registry: map[string]string{"owner": identity.Recipient().String()}},
		Encryption: EncryptionConfig{Files: []FilePair{
			{PlaintextPath: filepath.Join(root, "app.toml"), EncryptedPath: filepath.Join(root, "prod.enc.toml"), Format: "toml", ConfigPath: configPath, ConfigDir: root},
			{PlaintextPath: filepath.Join(root, "prod.enc.toml"), EncryptedPath: filepath.Join(root, "prod.enc.enc.toml"), Format: "toml", ConfigPath: configPath, ConfigDir: root},
		}},
	}

	paths, err := CompletionPaths(cfg, task.ModePlan)
	require.NoError(t, err)
	assert.Equal(t, []string{"app.toml", "prod.enc.enc.toml", "prod.enc.toml"}, paths)

	paths, err = CompletionPaths(cfg, task.ModeEncrypt)
	require.NoError(t, err)
	assert.Equal(t, []string{"app.toml", "prod.enc.toml"}, paths)
}

func TestCompletionPathsPropagatesDiscoveryErrors(t *testing.T) {
	first, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	second, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	firstAliases := []string{"first"}
	secondAliases := []string{"second"}
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "config.yaml"), []byte("key: value\n"), 0600))
	cfg := &Config{
		CurrentDir: root,
		Recipients: RecipientConfig{Registry: map[string]string{
			"first":  first.Recipient().String(),
			"second": second.Recipient().String(),
		}},
		Encryption: EncryptionConfig{Groups: []GroupConfig{
			{Patterns: []string{"config.yaml"}, Recipients: &firstAliases, ConfigDir: root},
			{Patterns: []string{"config.yaml"}, Recipients: &secondAliases, ConfigDir: root},
		}},
	}

	_, err = CompletionPaths(cfg, task.ModeEncrypt)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "conflicting recipient sets")
}
