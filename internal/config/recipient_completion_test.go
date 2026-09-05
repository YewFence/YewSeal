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

func TestResolveFileRecipientsPreservesAuthorizationProvenance(t *testing.T) {
	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	aliases := []string{"owner"}
	cfg := &Config{
		Recipients: RecipientConfig{
			Registry:        map[string]string{"owner": identity.Recipient().String()},
			RegistrySources: map[string]string{"owner": "/project/.yewseal.toml"},
		},
	}
	pair := FilePair{
		PlaintextPath:   "config.yaml",
		ConfigPath:      "/project/.yewseal.toml",
		Recipients:      &aliases,
		RecipientSource: ValueSource{Kind: "file", ConfigPath: "/project/.yewseal.toml", Detail: "recipients"},
	}

	resolved, err := cfg.ResolveFileRecipients(pair)
	require.NoError(t, err)
	assert.Equal(t, "file", resolved.Provenance.Kind)
	assert.Equal(t, "/project/.yewseal.toml", resolved.Provenance.EffectiveSource.ConfigPath)
	assert.Equal(t, "/project/.yewseal.toml", resolved.Provenance.RegistrySources["owner"])
}

func TestConfiguredGroupsRejectCanonicalRecipientConflict(t *testing.T) {
	first, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	second, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	firstAliases := []string{"first"}
	secondAliases := []string{"second"}
	root := t.TempDir()
	file := filepath.Join(root, "config.yaml")
	require.NoError(t, os.WriteFile(file, []byte("key: value\n"), 0600))
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

	_, err = configuredFilePairs(cfg, task.ModeEncrypt, groupRequestOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "conflicting recipient sets")
}

func TestDecryptResolutionWarnsForUnknownAlias(t *testing.T) {
	aliases := []string{"removed"}
	cfg := &Config{
		CurrentDir: rootForTest(t),
		Recipients: RecipientConfig{Registry: map[string]string{}},
		Encryption: EncryptionConfig{Files: []FilePair{
			{PlaintextPath: "config.yaml", EncryptedPath: "config.enc.yaml", Format: "yaml", Recipients: &aliases},
		}},
	}

	selection, err := ResolveSelection(cfg, SelectionOptions{Command: task.ModeDecrypt, Target: "config.enc.yaml"})
	require.NoError(t, err)
	require.Len(t, selection.FilePairs, 1)
	assert.Contains(t, selection.FilePairs[0].RecipientWarning, `unknown recipient alias "removed"`)
}

func rootForTest(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}
