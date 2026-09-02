package app

import (
	"filippo.io/age"
	"os"
	"testing"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/seal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryptFiles_SingleFileOverrideUsesProvidedPathsAndFormat(t *testing.T) {
	env := newAppCryptoTestEnv(t)

	require.NoError(t, os.WriteFile("secrets.vars", []byte("TOKEN=secret\n"), 0644))
	cfg := &config.Config{Encryption: config.EncryptionConfig{Files: []config.FilePair{{PlaintextPath: "secrets.vars", EncryptedPath: "secrets.vars.enc.yaml", Format: "env"}}}}

	err := EncryptFiles(cfg, EncryptRequest{
		KeyFile:   env.keyFile,
		PublicKey: env.publicKey,
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

	cfg := &config.Config{
		Encryption: config.EncryptionConfig{
			Files: []config.FilePair{
				{PlaintextPath: ".dev.vars", EncryptedPath: "configured.enc.yaml", Format: "env"},
			},
		},
	}

	err := EncryptFiles(cfg, EncryptRequest{
		KeyFile:   env.keyFile,
		PublicKey: env.publicKey,
		Target:    ".dev.vars",
		Parallel:  1,
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

	cfg := &config.Config{Encryption: config.EncryptionConfig{Files: []config.FilePair{{PlaintextPath: "secret", EncryptedPath: "secret.enc.env", Format: "env"}}}}
	err := EncryptFiles(cfg, EncryptRequest{
		KeyFile:   env.keyFile,
		PublicKey: env.publicKey,
		Target:    "secret",
		Format:    "env",
		Parallel:  1,
	})
	require.NoError(t, err)

	_, err = os.Stat("secret.enc.env")
	require.NoError(t, err)
}

func TestDecryptFiles_TargetFileOutputOverride(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	require.NoError(t, os.WriteFile("config.yaml", []byte("token: secret\n"), 0644))
	cfg := &config.Config{Encryption: config.EncryptionConfig{Files: []config.FilePair{{PlaintextPath: "config.yaml", EncryptedPath: "config.enc.yaml", Format: "yaml"}}}}
	err := EncryptFiles(cfg, EncryptRequest{
		KeyFile:   env.keyFile,
		PublicKey: env.publicKey,
		Target:    "config.yaml",
		Parallel:  1,
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
