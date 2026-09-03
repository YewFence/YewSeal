package app

import (
	"filippo.io/age"
	"io"
	"os"
	"testing"

	"github.com/YewFence/YewSeal/internal/agekey"
	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/seal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryptFiles_SingleFileOverrideUsesProvidedPathsAndFormat(t *testing.T) {
	env := newAppCryptoTestEnv(t)

	require.NoError(t, os.WriteFile("secrets.vars", []byte("TOKEN=secret\n"), 0644))
	cfg := configWithOwnerRecipient(&config.Config{Encryption: config.EncryptionConfig{Files: []config.FilePair{{PlaintextPath: "secrets.vars", EncryptedPath: "secrets.vars.enc.yaml", Format: "env"}}}}, env.publicKey)

	err := EncryptFiles(cfg, EncryptRequest{
		Target:    "secrets.vars",
		Output:    "secrets.vars.enc.yaml",
		OutputSet: true,
		Format:    "env",
		Parallel:  1,
	})
	require.NoError(t, err)

	require.NoError(t, os.Remove("secrets.vars"))
	err = DecryptFiles(cfg, DecryptRequest{
		KeyFile:   env.keyFile,
		Target:    "secrets.vars.enc.yaml",
		Output:    "secrets.vars",
		OutputSet: true,
		Format:    "env",
		Parallel:  1,
	})
	require.NoError(t, err)

	content, err := os.ReadFile("secrets.vars")
	require.NoError(t, err)
	assert.Equal(t, "TOKEN=secret\n", string(content))
}

func TestEncryptFilesUsesPerFileRecipientAliases(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	second, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	require.NoError(t, os.WriteFile("config.yaml", []byte("secret: value\n"), 0644))

	defaults := []string{"owner"}
	fileRecipients := []string{"backup"}
	cfg := &config.Config{
		Encryption: config.EncryptionConfig{
			Files: []config.FilePair{{
				PlaintextPath: "config.yaml",
				EncryptedPath: "config.enc.yaml",
				Recipients:    &fileRecipients,
			}},
		},
		Recipients: config.RecipientConfig{
			Defaults: &defaults,
			Registry: map[string]string{
				"owner":  env.publicKey,
				"backup": second.Recipient().String(),
			},
		},
	}

	require.NoError(t, EncryptFiles(cfg, EncryptRequest{Target: "config.yaml", Parallel: 1}))

	recipients, err := seal.ExtractAgeRecipientsFromEncryptedFile("config.enc.yaml", "config.yaml", "")
	require.NoError(t, err)
	require.Equal(t, []string{second.Recipient().String()}, recipients)
}

func TestEncryptFiles_DirModeRejectsFormatOverride(t *testing.T) {
	cfg := config.DefaultConfig()
	tempDir := t.TempDir()

	err := EncryptFiles(cfg, EncryptRequest{
		Target: tempDir,
		Format: "yaml",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--format is only supported in single-file mode")
}

func TestEncryptFiles_DirModeRejectsOutput(t *testing.T) {
	cfg := config.DefaultConfig()
	tempDir := t.TempDir()

	err := EncryptFiles(cfg, EncryptRequest{
		Target:    tempDir,
		Output:    "out.enc.yaml",
		OutputSet: true,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--output is only supported when the path target is a file")
}

func TestDecryptFiles_DirModeRejectsOutput(t *testing.T) {
	cfg := config.DefaultConfig()
	tempDir := t.TempDir()

	err := DecryptFiles(cfg, DecryptRequest{
		Target:    tempDir,
		Output:    "out.yaml",
		OutputSet: true,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--output is only supported when the path target is a file")
}

func TestEncryptFiles_TargetFileUsesConfiguredPair(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	require.NoError(t, os.WriteFile(".dev.vars", []byte("TOKEN=secret\n"), 0644))

	cfg := configWithOwnerRecipient(&config.Config{
		Encryption: config.EncryptionConfig{
			Files: []config.FilePair{
				{PlaintextPath: ".dev.vars", EncryptedPath: "configured.enc.yaml", Format: "env"},
			},
		},
	}, env.publicKey)

	err := EncryptFiles(cfg, EncryptRequest{
		Target:   ".dev.vars",
		Parallel: 1,
	})
	require.NoError(t, err)

	_, err = os.Stat("configured.enc.yaml")
	require.NoError(t, err)
	_, err = os.Stat(".dev.vars.enc.env")
	assert.True(t, os.IsNotExist(err))
}

func TestEncryptFiles_TargetFileFormatOverrideDeterminesGeneratedOutput(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	require.NoError(t, os.WriteFile("secret", []byte("TOKEN=secret\n"), 0644))

	cfg := configWithOwnerRecipient(&config.Config{Encryption: config.EncryptionConfig{Files: []config.FilePair{{PlaintextPath: "secret", EncryptedPath: "secret.enc.env", Format: "env"}}}}, env.publicKey)
	err := EncryptFiles(cfg, EncryptRequest{
		Target:   "secret",
		Format:   "env",
		Parallel: 1,
	})
	require.NoError(t, err)

	_, err = os.Stat("secret.enc.env")
	require.NoError(t, err)
}

func TestDecryptFiles_TargetFileOutputOverride(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	require.NoError(t, os.WriteFile("config.yaml", []byte("token: secret\n"), 0644))
	cfg := configWithOwnerRecipient(&config.Config{Encryption: config.EncryptionConfig{Files: []config.FilePair{{PlaintextPath: "config.yaml", EncryptedPath: "config.enc.yaml", Format: "yaml"}}}}, env.publicKey)
	err := EncryptFiles(cfg, EncryptRequest{
		Target:   "config.yaml",
		Parallel: 1,
	})
	require.NoError(t, err)
	require.NoError(t, os.Remove("config.yaml"))

	err = DecryptFiles(cfg, DecryptRequest{
		KeyFile:   env.keyFile,
		Target:    "config.enc.yaml",
		Output:    "custom.yaml",
		OutputSet: true,
		Parallel:  1,
	})
	require.NoError(t, err)

	content, err := os.ReadFile("custom.yaml")
	require.NoError(t, err)
	assert.Equal(t, "token: secret\n", string(content))
	_, err = os.Stat("config.yaml")
	assert.True(t, os.IsNotExist(err))
}

func TestDecryptFiles_ConfigModeRejectsFormatOverride(t *testing.T) {
	cfg := config.DefaultConfig()

	err := DecryptFiles(cfg, DecryptRequest{
		Format: "yaml",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--format is only supported in single-file mode")
}

type appCryptoTestEnv struct {
	keyFile   string
	publicKey string
}

func newAppCryptoTestEnv(t *testing.T) appCryptoTestEnv {
	t.Helper()

	tempDir := t.TempDir()
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tempDir))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(oldWd))
	})

	keyFile, publicKey := createAgeKeyFile(t, tempDir)
	return appCryptoTestEnv{keyFile: keyFile, publicKey: publicKey}
}

func configWithOwnerRecipient(cfg *config.Config, publicKey string) *config.Config {
	defaults := []string{"owner"}
	cfg.Recipients = config.RecipientConfig{
		Defaults: &defaults,
		Registry: map[string]string{"owner": publicKey},
	}
	return cfg
}

func TestEncryptFilesRejectsMissingAuthorizationBeforeWrites(t *testing.T) {
	root := t.TempDir()
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(root))
	t.Cleanup(func() { require.NoError(t, os.Chdir(oldWd)) })
	require.NoError(t, os.WriteFile("secret.yaml", []byte("token: value\n"), 0644))
	cfg := &config.Config{CurrentDir: root, UserConfig: true, Encryption: config.EncryptionConfig{Files: []config.FilePair{{PlaintextPath: "secret.yaml", EncryptedPath: "secret.enc.yaml", Format: "yaml"}}}}
	err = EncryptFiles(cfg, EncryptRequest{Target: "secret.yaml", UpdateProjectMetadata: true})
	require.Error(t, err)
	require.Contains(t, err.Error(), "no recipient set")
	_, statErr := os.Stat("secret.enc.yaml")
	require.ErrorIs(t, statErr, os.ErrNotExist)
	_, statErr = os.Stat(".sops.yaml")
	require.ErrorIs(t, statErr, os.ErrNotExist)
}

func TestEncryptFilesPreflightsEntireBatchBeforeFirstCiphertext(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	require.NoError(t, os.WriteFile("first.yaml", []byte("token: first\n"), 0644))
	require.NoError(t, os.WriteFile("second.yaml", []byte("token: second\n"), 0644))
	valid := []string{"owner"}
	invalid := []string{"missing"}
	cfg := &config.Config{CurrentDir: config.CurrentDir(&config.Config{}), Recipients: config.RecipientConfig{Registry: map[string]string{"owner": env.publicKey}}, Encryption: config.EncryptionConfig{Files: []config.FilePair{{PlaintextPath: "first.yaml", EncryptedPath: "first.enc.yaml", Format: "yaml", Recipients: &valid}, {PlaintextPath: "second.yaml", EncryptedPath: "second.enc.yaml", Format: "yaml", Recipients: &invalid}}}}
	err := EncryptFiles(cfg, EncryptRequest{Parallel: 1})
	require.Error(t, err)
	require.Contains(t, err.Error(), `unknown recipient alias "missing"`)
	_, statErr := os.Stat("first.enc.yaml")
	require.ErrorIs(t, statErr, os.ErrNotExist)
	_, statErr = os.Stat("second.enc.yaml")
	require.ErrorIs(t, statErr, os.ErrNotExist)
}

func TestEncryptFilesWritesPortableSopsPaths(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	require.NoError(t, os.WriteFile("secret.yaml", []byte("token: value\n"), 0644))
	cfg := configWithOwnerRecipient(&config.Config{CurrentDir: config.CurrentDir(&config.Config{}), Encryption: config.EncryptionConfig{Files: []config.FilePair{{PlaintextPath: "secret.yaml", EncryptedPath: "secret.enc.yaml", Format: "yaml"}}}}, env.publicKey)
	require.NoError(t, EncryptFiles(cfg, EncryptRequest{Target: "secret.yaml", Parallel: 1, UpdateProjectMetadata: true}))
	content, err := os.ReadFile(".sops.yaml")
	require.NoError(t, err)
	assert.Contains(t, string(content), `path_regex: ^secret\.enc\.yaml$`)
	assert.NotContains(t, string(content), config.CurrentDir(cfg))
}

func TestDecryptFilesWarnsForStaleAliasAndUsesEncryptedMetadata(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	require.NoError(t, os.WriteFile("secret.yaml", []byte("token: value\n"), 0644))
	require.NoError(t, seal.Encrypt(seal.EncryptOptions{InputFile: "secret.yaml", OutputFile: "secret.enc.yaml", Recipients: []string{env.publicKey}, FormatOverride: "yaml"}))
	require.NoError(t, os.Remove("secret.yaml"))
	aliases := []string{"retired-owner"}
	cfg := &config.Config{Encryption: config.EncryptionConfig{Files: []config.FilePair{{PlaintextPath: "secret.yaml", EncryptedPath: "secret.enc.yaml", Format: "yaml", Recipients: &aliases, ConfigPath: ".yewseal.toml"}}}}
	read, write, err := os.Pipe()
	require.NoError(t, err)
	originalStderr := os.Stderr
	os.Stderr = write
	t.Cleanup(func() { os.Stderr = originalStderr })
	err = DecryptFiles(cfg, DecryptRequest{KeyFile: env.keyFile, Target: "secret.enc.yaml", Parallel: 1})
	require.NoError(t, write.Close())
	os.Stderr = originalStderr
	require.NoError(t, err)
	warning, err := io.ReadAll(read)
	require.NoError(t, err)
	require.NoError(t, read.Close())
	assert.Contains(t, string(warning), `unknown recipient alias "retired-owner"`)
	assert.Contains(t, string(warning), ".yewseal.toml")
	content, err := os.ReadFile("secret.yaml")
	require.NoError(t, err)
	assert.Equal(t, "token: value\n", string(content))
}

func TestDecryptFilesEnvironmentBundlePrecedesConfiguredKeyFile(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	bundle, err := agekey.GetIdentityBundle(env.keyFile)
	require.NoError(t, err)
	t.Setenv("YEWSEAL_AGE_IDENTITIES", bundle.String())
	require.NoError(t, os.WriteFile("secret.yaml", []byte("token: value\n"), 0644))
	require.NoError(t, seal.Encrypt(seal.EncryptOptions{InputFile: "secret.yaml", OutputFile: "secret.enc.yaml", Recipients: []string{env.publicKey}, FormatOverride: "yaml"}))
	require.NoError(t, os.Remove("secret.yaml"))
	cfg := &config.Config{Key: config.KeyConfig{FilePath: "missing-configured-key.txt"}, Encryption: config.EncryptionConfig{Files: []config.FilePair{{PlaintextPath: "secret.yaml", EncryptedPath: "secret.enc.yaml", Format: "yaml"}}}}
	require.NoError(t, DecryptFiles(cfg, DecryptRequest{Target: "secret.enc.yaml", Parallel: 1}))
	content, err := os.ReadFile("secret.yaml")
	require.NoError(t, err)
	assert.Equal(t, "token: value\n", string(content))
}
