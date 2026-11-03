package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/sopsx"
	"github.com/stretchr/testify/require"
)

func TestTargetedDecryptRebuildsCompleteGitignoreWithoutRequiringCurrentAuthorization(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	root, err := os.Getwd()
	require.NoError(t, err)
	for _, path := range []string{"first.yaml", "obsolete.yaml", "group-plain.yaml"} {
		require.NoError(t, os.WriteFile(path, []byte("token: fixture\n"), 0600))
	}
	ciphertext, err := sopsx.Encrypt([]byte("token: fixture\n"), "yaml", []string{env.publicKey})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile("group-history.enc.yaml", ciphertext, 0600))
	cfg := configWithOwnerRecipient(&config.Config{
		CurrentDir: root,
		Encryption: config.EncryptionConfig{
			Files: []config.FilePair{
				{PlaintextPath: "first.yaml", EncryptedPath: "first.enc.yaml"},
				{PlaintextPath: "obsolete.yaml", EncryptedPath: "obsolete.enc.yaml"},
			},
			Groups: []config.GroupConfig{{ConfigDir: root, Patterns: []string{"group-*.yaml"}}},
		},
	}, env.publicKey)
	require.NoError(t, EncryptFiles(cfg, EncryptRequest{Targets: []string{"first.yaml"}, UpdateGitignore: true}))
	before, err := os.ReadFile(".gitignore")
	require.NoError(t, err)
	require.Contains(t, string(before), "\n/obsolete.yaml\n")
	cfg.Encryption.Files = cfg.Encryption.Files[:1]
	missing := []string{"removed-alias"}
	cfg.Encryption.Groups[0].Recipients = &missing
	require.NoError(t, os.Remove("first.yaml"))
	require.NoError(t, DecryptFiles(cfg, DecryptRequest{Targets: []string{"first.enc.yaml"}, KeyFile: env.keyFile, UpdateGitignore: true}))
	data, err := os.ReadFile(".gitignore")
	require.NoError(t, err)
	require.Equal(t, "# BEGIN YewSeal managed entries\n/first.yaml\n/group-history.yaml\n/group-plain.yaml\n/.age/keys.txt\n# END YewSeal managed entries\n", string(data))
	require.FileExists(t, filepath.Join(root, "first.yaml"))
	require.NoFileExists(t, "group-history.yaml")
}

func TestEncryptClearsStaleGitignoreWhenConfigurationHasNoMappings(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	root, err := os.Getwd()
	require.NoError(t, err)
	cfg := configWithOwnerRecipient(&config.Config{CurrentDir: root, LoadedFiles: []config.LoadedFile{{Path: filepath.Join(root, ".yewseal.toml"), Dir: root}}}, env.publicKey)
	require.NoError(t, os.WriteFile(".gitignore", []byte("# BEGIN YewSeal managed entries\n/obsolete.yaml\n# END YewSeal managed entries\n"), 0600))
	require.NoError(t, EncryptFiles(cfg, EncryptRequest{UpdateGitignore: true}))
	data, err := os.ReadFile(".gitignore")
	require.NoError(t, err)
	require.Equal(t, "# BEGIN YewSeal managed entries\n/.age/keys.txt\n# END YewSeal managed entries\n", string(data))
}
