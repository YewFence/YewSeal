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

func TestGitignoreFailureStopsConfiguredPathWrites(t *testing.T) {
	for _, command := range []string{"encrypt", "decrypt"} {
		t.Run(command, func(t *testing.T) {
			env := newAppCryptoTestEnv(t)
			plain := []byte("token: fixture\n")
			cfg := configWithOwnerRecipient(&config.Config{Encryption: config.EncryptionConfig{Files: []config.FilePair{{PlaintextPath: "secret.yaml", EncryptedPath: "secret.enc.yaml"}}}}, env.publicKey)
			if command == "encrypt" {
				require.NoError(t, os.WriteFile("secret.yaml", plain, 0600))
			} else {
				ciphertext, err := sopsx.Encrypt(plain, "yaml", []string{env.publicKey})
				require.NoError(t, err)
				require.NoError(t, os.WriteFile("secret.enc.yaml", ciphertext, 0600))
			}
			malformed := []byte("# BEGIN YewSeal managed entries\n/custom.yaml\n")
			require.NoError(t, os.WriteFile(".gitignore", malformed, 0600))
			var err error
			if command == "encrypt" {
				err = EncryptFiles(cfg, EncryptRequest{UpdateGitignore: true, SyncSOPSConfig: true})
				require.NoFileExists(t, "secret.enc.yaml")
				require.NoFileExists(t, ".sops.yaml")
				data, readErr := os.ReadFile("secret.yaml")
				require.NoError(t, readErr)
				require.Equal(t, plain, data)
			} else {
				err = DecryptFiles(cfg, DecryptRequest{UpdateGitignore: true, KeyFile: env.keyFile})
				require.NoFileExists(t, "secret.yaml")
			}
			require.ErrorContains(t, err, "missing END line")
			data, readErr := os.ReadFile(".gitignore")
			require.NoError(t, readErr)
			require.Equal(t, malformed, data)
		})
	}
}

func TestDecryptMetadataFailureStopsBeforePlaintextWrites(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	root, err := os.Getwd()
	require.NoError(t, err)
	ciphertext, err := sopsx.Encrypt([]byte("token: fixture\n"), "yaml", []string{env.publicKey})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile("selected.enc.yaml", ciphertext, 0600))
	require.NoError(t, os.WriteFile("group-plain.unknown", []byte("fixture"), 0600))
	cfg := configWithOwnerRecipient(&config.Config{
		CurrentDir: root,
		Encryption: config.EncryptionConfig{
			Files:  []config.FilePair{{PlaintextPath: "selected.yaml", EncryptedPath: "selected.enc.yaml"}},
			Groups: []config.GroupConfig{{ConfigDir: root, Patterns: []string{"group-*"}}},
		},
	}, env.publicKey)
	err = DecryptFiles(cfg, DecryptRequest{Targets: []string{"selected.enc.yaml"}, KeyFile: env.keyFile, UpdateGitignore: true})
	require.ErrorContains(t, err, "could not detect format")
	require.NoFileExists(t, "selected.yaml")
	require.NoFileExists(t, ".gitignore")
	require.NoError(t, DecryptFiles(cfg, DecryptRequest{Targets: []string{"selected.enc.yaml"}, KeyFile: env.keyFile}))
	require.FileExists(t, "selected.yaml")
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
