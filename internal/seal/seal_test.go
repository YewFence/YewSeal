package seal

import (
	"bytes"
	"fmt"
	"io"
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

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)

	os.Stdout = w
	defer func() {
		os.Stdout = oldStdout
	}()

	fn()
	require.NoError(t, w.Close())

	var output bytes.Buffer
	_, err = io.Copy(&output, r)
	require.NoError(t, err)
	require.NoError(t, r.Close())
	return output.String()
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

	info, err := os.Stat("config.yaml")
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
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

func TestEncryptVerboseWritesToProvidedOutput(t *testing.T) {
	env := setupTestEnv(t)
	require.NoError(t, os.WriteFile("config.yaml", []byte("secret: value\n"), 0644))

	var output bytes.Buffer
	var encryptErr error
	stdout := captureStdout(t, func() {
		encryptErr = Encrypt(EncryptOptions{
			InputFile:      "config.yaml",
			OutputFile:     "config.enc.yaml",
			KeyFile:        env.keyFile,
			PublicKey:      env.publicKey,
			FormatOverride: "yaml",
			Verbose:        true,
			Output:         &output,
		})
	})

	require.NoError(t, encryptErr)
	assert.Empty(t, stdout)
	assert.Contains(t, output.String(), "Using public key from command-line parameter")
	assert.Contains(t, output.String(), "Encrypted config.yaml")
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

	info, statErr := os.Stat("config.yaml")
	require.NoError(t, statErr)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

func TestDecryptTightensMatchingPlaintextPermissions(t *testing.T) {
	env := setupTestEnv(t)
	plain := []byte("secret: original\n")
	require.NoError(t, os.WriteFile("config.yaml", plain, 0644))
	require.NoError(t, Encrypt(EncryptOptions{
		InputFile:      "config.yaml",
		OutputFile:     "config.enc.yaml",
		KeyFile:        env.keyFile,
		PublicKey:      env.publicKey,
		FormatOverride: "yaml",
	}))
	require.NoError(t, os.WriteFile("config.yaml", plain, 0644))

	require.NoError(t, Decrypt(DecryptOptions{
		InputFile:      "config.enc.yaml",
		OutputFile:     "config.yaml",
		KeyFile:        env.keyFile,
		FormatOverride: "yaml",
	}))

	info, err := os.Stat("config.yaml")
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

func TestEncryptUnknownFormatFailsWithBinaryHint(t *testing.T) {
	env := setupTestEnv(t)
	plain := []byte{0, 1, 2, 3, 's', 'e', 'c', 'r', 'e', 't', 255}
	require.NoError(t, os.WriteFile("secret.blob", plain, 0644))

	err := Encrypt(EncryptOptions{
		InputFile:  "secret.blob",
		OutputFile: "secret.blob.enc",
		KeyFile:    env.keyFile,
		PublicKey:  env.publicKey,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "could not detect format for secret.blob")
	assert.Contains(t, err.Error(), "Hint: pass --format binary")
}

func TestDecryptToBytesUnknownFormatFailsWithBinaryHint(t *testing.T) {
	env := setupTestEnv(t)
	plain := []byte("plain text with unknown extension")
	require.NoError(t, os.WriteFile("secret.vars", plain, 0644))
	require.NoError(t, Encrypt(EncryptOptions{
		InputFile:      "secret.vars",
		OutputFile:     "secret.vars.enc",
		KeyFile:        env.keyFile,
		PublicKey:      env.publicKey,
		FormatOverride: "binary",
	}))

	var warningOutput bytes.Buffer
	var decryptErr error
	stdout := captureStdout(t, func() {
		_, decryptErr = DecryptToBytes(DecryptBytesOptions{
			InputFile:  "secret.vars.enc",
			OutputFile: "secret.vars",
			KeyFile:    env.keyFile,
			Warnings:   &warningOutput,
		})
	})

	require.Error(t, decryptErr)
	assert.Empty(t, stdout)
	assert.Empty(t, warningOutput.String())
	assert.Contains(t, decryptErr.Error(), "could not detect format for secret.vars")
	assert.Contains(t, decryptErr.Error(), "Hint: pass --format binary")
}

func TestEncryptDecryptBinaryOverrideRoundTrip(t *testing.T) {
	env := setupTestEnv(t)
	plain := []byte("plain text with unknown extension")
	require.NoError(t, os.WriteFile("secret.vars", plain, 0644))

	require.NoError(t, Encrypt(EncryptOptions{
		InputFile:      "secret.vars",
		OutputFile:     "secret.vars.enc",
		KeyFile:        env.keyFile,
		PublicKey:      env.publicKey,
		FormatOverride: "binary",
	}))
	require.NoError(t, os.Remove("secret.vars"))
	require.NoError(t, Decrypt(DecryptOptions{
		InputFile:      "secret.vars.enc",
		OutputFile:     "secret.vars",
		KeyFile:        env.keyFile,
		FormatOverride: "binary",
	}))

	decrypted, err := os.ReadFile("secret.vars")
	require.NoError(t, err)
	assert.Equal(t, plain, decrypted)
}

func TestEncryptInvalidFormatOverride(t *testing.T) {
	env := setupTestEnv(t)
	require.NoError(t, os.WriteFile("config.yaml", []byte("TOKEN=secret\n"), 0644))

	err := Encrypt(EncryptOptions{
		InputFile:      "config.yaml",
		OutputFile:     "config.enc.yaml",
		KeyFile:        env.keyFile,
		PublicKey:      env.publicKey,
		FormatOverride: "xml",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unsupported format override "xml"`)
}

func TestSplitEditorCommandPreservesQuotedExecutable(t *testing.T) {
	parts, err := splitEditorCommand(`"/Applications/Sublime Text.app/Contents/SharedSupport/bin/subl" -w`)
	require.NoError(t, err)
	assert.Equal(t, []string{"/Applications/Sublime Text.app/Contents/SharedSupport/bin/subl", "-w"}, parts)
}
