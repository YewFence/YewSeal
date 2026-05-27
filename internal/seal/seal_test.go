package seal

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"filippo.io/age"
	"github.com/YewFence/YewSeal/internal/errx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testEnv struct {
	keyFile   string
	publicKey string
}

func setupTestEnv(t *testing.T) testEnv {
	t.Helper()

	tempDir := t.TempDir()
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tempDir))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(oldWd))
	})

	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	keyDir := filepath.Join(tempDir, ".age")
	require.NoError(t, os.MkdirAll(keyDir, 0700))

	keyFile := filepath.Join(keyDir, "keys.txt")
	publicKey := identity.Recipient().String()
	keyContent := fmt.Sprintf("# created: %s\n# public key: %s\n%s\n",
		time.Now().UTC().Format(time.RFC3339),
		publicKey,
		identity.String(),
	)
	require.NoError(t, os.WriteFile(keyFile, []byte(keyContent), 0600))

	return testEnv{keyFile: keyFile, publicKey: publicKey}
}

func TestEncryptDecryptYAMLRoundTrip(t *testing.T) {
	env := setupTestEnv(t)
	plain := []byte("database:\n  host: localhost\n  password: secret123\n")
	require.NoError(t, os.WriteFile("config.yaml", plain, 0644))

	require.NoError(t, Encrypt(EncryptOptions{
		InputFile:      "config.yaml",
		OutputFile:     "config.enc.yaml",
		KeyFile:        env.keyFile,
		PublicKey:      env.publicKey,
		FormatOverride: "yaml",
	}))

	require.NoError(t, os.Remove("config.yaml"))
	require.NoError(t, Decrypt(DecryptOptions{
		InputFile:      "config.enc.yaml",
		OutputFile:     "config.yaml",
		KeyFile:        env.keyFile,
		FormatOverride: "yaml",
	}))

	decrypted, err := os.ReadFile("config.yaml")
	require.NoError(t, err)
	assert.Contains(t, string(decrypted), "host: localhost")
	assert.Contains(t, string(decrypted), "password: secret123")
}

func TestDecryptToBytesDoesNotWriteOutput(t *testing.T) {
	env := setupTestEnv(t)
	require.NoError(t, os.WriteFile("config.yaml", []byte("secret: value\n"), 0644))
	require.NoError(t, Encrypt(EncryptOptions{
		InputFile:      "config.yaml",
		OutputFile:     "config.enc.yaml",
		KeyFile:        env.keyFile,
		PublicKey:      env.publicKey,
		FormatOverride: "yaml",
	}))

	plainData, err := DecryptToBytes(DecryptBytesOptions{
		InputFile:      "config.enc.yaml",
		OutputFile:     "view.yaml",
		KeyFile:        env.keyFile,
		FormatOverride: "yaml",
	})
	require.NoError(t, err)
	assert.Contains(t, string(plainData), "secret: value")

	_, statErr := os.Stat("view.yaml")
	assert.True(t, os.IsNotExist(statErr))
}

func TestDecryptRefusesToOverwriteDifferentPlaintextUnlessForced(t *testing.T) {
	env := setupTestEnv(t)
	require.NoError(t, os.WriteFile("config.yaml", []byte("secret: original\n"), 0644))
	require.NoError(t, Encrypt(EncryptOptions{
		InputFile:      "config.yaml",
		OutputFile:     "config.enc.yaml",
		KeyFile:        env.keyFile,
		PublicKey:      env.publicKey,
		FormatOverride: "yaml",
	}))
	require.NoError(t, os.WriteFile("config.yaml", []byte("secret: local\n"), 0644))

	err := Decrypt(DecryptOptions{
		InputFile:      "config.enc.yaml",
		OutputFile:     "config.yaml",
		KeyFile:        env.keyFile,
		FormatOverride: "yaml",
	})
	require.Error(t, err)

	var overwriteErr *errx.ProtectedOverwriteError
	assert.ErrorAs(t, err, &overwriteErr)

	require.NoError(t, Decrypt(DecryptOptions{
		InputFile:      "config.enc.yaml",
		OutputFile:     "config.yaml",
		KeyFile:        env.keyFile,
		FormatOverride: "yaml",
		Force:          true,
	}))

	content, readErr := os.ReadFile("config.yaml")
	require.NoError(t, readErr)
	assert.Contains(t, string(content), "original")
}

func TestEncryptUnsupportedFormat(t *testing.T) {
	env := setupTestEnv(t)
	require.NoError(t, os.WriteFile("config.vars", []byte("TOKEN=secret\n"), 0644))

	err := Encrypt(EncryptOptions{
		InputFile:  "config.vars",
		OutputFile: "config.enc.yaml",
		KeyFile:    env.keyFile,
		PublicKey:  env.publicKey,
	})
	require.Error(t, err)

	var unsupportedErr *errx.UnsupportedFormatError
	assert.ErrorAs(t, err, &unsupportedErr)
}
