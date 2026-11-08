package app

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/errx"
	"github.com/YewFence/YewSeal/internal/sopsx"
	"github.com/stretchr/testify/require"
)

func TestDiffSourcePreviewsTreeAgainstCiphertext(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	ciphertext, err := sopsx.Encrypt([]byte("token: secret\n"), "yaml", []string{env.publicKey})
	require.NoError(t, err)
	cfg := configWithOwnerRecipient(&config.Config{CurrentDir: ".", Encryption: config.EncryptionConfig{Files: []config.FilePair{
		{PlaintextPath: "config.yaml", EncryptedPath: "config.enc.yaml", Format: "yaml"},
	}}}, env.publicKey)
	require.NoError(t, os.WriteFile("config.enc.yaml", ciphertext, 0600))

	delivery := t.TempDir()
	writeSourceTree(t, delivery, "config.yaml", "token: edited\n")

	var out bytes.Buffer
	var diagnostics bytes.Buffer
	result, err := DiffTargets(&out, &diagnostics, cfg, DiffRequest{SourceDir: delivery, KeyFile: env.keyFile, ColorMode: "never"})
	require.NoError(t, err)
	require.True(t, result.Different)
	require.Contains(t, out.String(), "token: edited")
	require.Contains(t, out.String(), "token: secret")
	require.NoFileExists(t, "config.yaml", "diff never writes registered plaintext")
}

func TestDiffSourceMissingTreeFileSkipsWithoutRegisteredFallback(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	ciphertext, err := sopsx.Encrypt([]byte("token: secret\n"), "yaml", []string{env.publicKey})
	require.NoError(t, err)
	cfg := configWithOwnerRecipient(&config.Config{CurrentDir: ".", Encryption: config.EncryptionConfig{Files: []config.FilePair{
		{PlaintextPath: "config.yaml", EncryptedPath: "config.enc.yaml", Format: "yaml"},
	}}}, env.publicKey)
	require.NoError(t, os.WriteFile("config.enc.yaml", ciphertext, 0600))
	// Stale registered plaintext must not silently stand in for the tree.
	require.NoError(t, os.WriteFile("config.yaml", []byte("token: stale\n"), 0600))

	var out bytes.Buffer
	var diagnostics bytes.Buffer
	result, err := DiffTargets(&out, &diagnostics, cfg, DiffRequest{SourceDir: t.TempDir(), KeyFile: env.keyFile, ColorMode: "never"})
	require.NoError(t, err)
	require.False(t, result.Different)
	require.Empty(t, out.String())
	require.Contains(t, diagnostics.String(), "plaintext file is missing")
}

func TestDiffSourceCoversCiphertextOnlyGroupMappings(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	cwd, err := os.Getwd()
	require.NoError(t, err)
	ciphertext, err := sopsx.Encrypt([]byte("token: secret\n"), "yaml", []string{env.publicKey})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile("remote.enc.yaml", ciphertext, 0600))
	cfg := configWithOwnerRecipient(&config.Config{
		CurrentDir: cwd,
		Encryption: config.EncryptionConfig{Groups: []config.GroupConfig{{ConfigDir: cwd, Patterns: []string{"*.yaml"}}}},
	}, env.publicKey)

	delivery := t.TempDir()
	writeSourceTree(t, delivery, "remote.yaml", "token: edited\n")

	var out bytes.Buffer
	var diagnostics bytes.Buffer
	result, err := DiffTargets(&out, &diagnostics, cfg, DiffRequest{SourceDir: delivery, KeyFile: env.keyFile, ColorMode: "never"})
	require.NoError(t, err)
	require.True(t, result.Different)
	require.Contains(t, out.String(), "token: edited")
}

func TestDiffSourceRootPreconditions(t *testing.T) {
	for _, kind := range []string{"missing", "file", "symlink", "blank"} {
		t.Run(kind, func(t *testing.T) {
			env := newAppCryptoTestEnv(t)
			cfg := configWithOwnerRecipient(&config.Config{CurrentDir: ".", Encryption: config.EncryptionConfig{Files: []config.FilePair{
				{PlaintextPath: "secret.yaml", EncryptedPath: "secret.enc.yaml", Format: "yaml"},
			}}}, env.publicKey)
			var source string
			switch kind {
			case "missing":
				source = filepath.Join(t.TempDir(), "absent")
			case "file":
				source = filepath.Join(t.TempDir(), "source")
				require.NoError(t, os.WriteFile(source, []byte("untouched"), 0644))
			case "symlink":
				source = filepath.Join(t.TempDir(), "source")
				require.NoError(t, os.Symlink(t.TempDir(), source))
			case "blank":
				source = "  "
			}
			var out bytes.Buffer
			var diagnostics bytes.Buffer
			_, err := DiffTargets(&out, &diagnostics, cfg, DiffRequest{SourceDir: source, KeyFile: env.keyFile})
			require.ErrorContains(t, err, "--source")
			var usage *errx.UsageError
			require.ErrorAs(t, err, &usage)
			require.Equal(t, 2, usage.ExitCode())
		})
	}
}

func TestDiffSourceProjectionRejectsOutsideProjectRoot(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	external := filepath.Join(t.TempDir(), "external.yaml")
	cfg := configWithOwnerRecipient(&config.Config{CurrentDir: ".", Encryption: config.EncryptionConfig{Files: []config.FilePair{
		{PlaintextPath: external, EncryptedPath: "external.enc.yaml", Format: "yaml"},
	}}}, env.publicKey)

	var out bytes.Buffer
	var diagnostics bytes.Buffer
	_, err := DiffTargets(&out, &diagnostics, cfg, DiffRequest{SourceDir: t.TempDir(), Targets: []string{external}, KeyFile: env.keyFile})
	require.ErrorContains(t, err, "not a file within project root")
	var usage *errx.UsageError
	require.ErrorAs(t, err, &usage)
}
