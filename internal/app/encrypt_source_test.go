package app

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/errx"
	"github.com/YewFence/YewSeal/internal/presentation"
	"github.com/YewFence/YewSeal/internal/seal"
	"github.com/YewFence/YewSeal/internal/sopsx"
	"github.com/stretchr/testify/require"
)

func writeSourceTree(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))
}

func TestEncryptSourceRoundTripsDecryptOutputTree(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	require.NoError(t, os.MkdirAll("config/nested", 0755))
	ciphertext, err := sopsx.Encrypt([]byte("token: secret\n"), "yaml", []string{env.publicKey})
	require.NoError(t, err)
	cfg := configWithOwnerRecipient(&config.Config{CurrentDir: ".", Encryption: config.EncryptionConfig{Files: []config.FilePair{
		{PlaintextPath: "config/first.yaml", EncryptedPath: "config/first.enc.yaml", Format: "yaml"},
		{PlaintextPath: "config/nested/second.yaml", EncryptedPath: "config/nested/second.enc.yaml", Format: "yaml", PlaintextMode: config.PlaintextDelivery},
	}}}, env.publicKey)
	for _, pair := range cfg.Encryption.Files {
		require.NoError(t, os.WriteFile(pair.EncryptedPath, ciphertext, 0600))
	}

	delivery := t.TempDir()
	require.NoError(t, DecryptFiles(cfg, DecryptRequest{OutputDir: delivery, KeyFile: env.keyFile, Strict: true}))
	writeSourceTree(t, delivery, "config/first.yaml", "token: edited\n")
	// The delivery-classified mapping keeps its project-relative slot in the tree.

	require.NoError(t, EncryptFiles(cfg, EncryptRequest{SourceDir: delivery, KeyFile: env.keyFile, Parallel: 2}))

	decrypted, err := seal.DecryptToBytes(seal.DecryptBytesOptions{InputFile: "config/first.enc.yaml", OutputFile: "config/first.yaml", FormatOverride: "yaml", IdentityBundle: mustTestBundle(t, env.keyFile)})
	require.NoError(t, err)
	require.Equal(t, "token: edited\n", string(decrypted))
	decrypted, err = seal.DecryptToBytes(seal.DecryptBytesOptions{InputFile: "config/nested/second.enc.yaml", OutputFile: "config/nested/second.yaml", FormatOverride: "yaml", IdentityBundle: mustTestBundle(t, env.keyFile)})
	require.NoError(t, err)
	require.Equal(t, "token: secret\n", string(decrypted))
	require.NoFileExists(t, "config/first.yaml")
	require.NoFileExists(t, "config/nested/second.yaml")
}

func TestEncryptSourceMissingFileSkipsWithoutRegisteredFallback(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	require.NoError(t, os.WriteFile("registered.yaml", []byte("stale plaintext\n"), 0600))
	require.NoError(t, os.WriteFile("missing.enc.yaml", []byte("existing ciphertext\n"), 0600))
	cfg := configWithOwnerRecipient(&config.Config{CurrentDir: ".", Encryption: config.EncryptionConfig{Files: []config.FilePair{
		{PlaintextPath: "registered.yaml", EncryptedPath: "registered.enc.yaml", Format: "yaml"},
		{PlaintextPath: "missing.yaml", EncryptedPath: "missing.enc.yaml", Format: "yaml"},
	}}}, env.publicKey)
	root := t.TempDir()
	writeSourceTree(t, root, "registered.yaml", "token: fresh\n")

	var diagnostics bytes.Buffer
	err := EncryptFiles(cfg, EncryptRequest{SourceDir: root, KeyFile: env.keyFile, Presentation: presentation.New(nil, &diagnostics, false)})
	require.NoError(t, err)
	require.Contains(t, diagnostics.String(), "missing plaintext")
	require.FileExists(t, "registered.enc.yaml")

	decrypted, err := seal.DecryptToBytes(seal.DecryptBytesOptions{InputFile: "registered.enc.yaml", OutputFile: "registered.yaml", FormatOverride: "yaml", IdentityBundle: mustTestBundle(t, env.keyFile)})
	require.NoError(t, err)
	require.Equal(t, "token: fresh\n", string(decrypted))
	// The missing tree file must not fall back to the stale registered plaintext.
	missing, err := os.ReadFile("missing.enc.yaml")
	require.NoError(t, err)
	require.Equal(t, "existing ciphertext\n", string(missing))
}

func TestEncryptSourceRootPreconditions(t *testing.T) {
	for _, kind := range []string{"missing", "file", "symlink"} {
		t.Run(kind, func(t *testing.T) {
			env := newAppCryptoTestEnv(t)
			cfg := configWithOwnerRecipient(&config.Config{CurrentDir: ".", Encryption: config.EncryptionConfig{Files: []config.FilePair{{PlaintextPath: "secret.yaml", EncryptedPath: "secret.enc.yaml"}}}}, env.publicKey)
			root := filepath.Join(t.TempDir(), "source")
			switch kind {
			case "file":
				require.NoError(t, os.WriteFile(root, []byte("untouched"), 0644))
			case "symlink":
				require.NoError(t, os.Symlink(t.TempDir(), root))
			}
			err := EncryptFiles(cfg, EncryptRequest{SourceDir: root, KeyFile: env.keyFile})
			require.ErrorContains(t, err, "--source")
			var usage *errx.UsageError
			require.ErrorAs(t, err, &usage)
			require.Equal(t, 2, usage.ExitCode())
			require.NoFileExists(t, "secret.enc.yaml")
		})
	}
}

func TestEncryptSourceRejectsOutsideProjectRootAndDuplicates(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	external := filepath.Join(t.TempDir(), "external.yaml")
	cfg := configWithOwnerRecipient(&config.Config{CurrentDir: ".", Encryption: config.EncryptionConfig{Files: []config.FilePair{
		{PlaintextPath: "first.yaml", EncryptedPath: "first.enc.yaml", Format: "yaml"},
		{PlaintextPath: external, EncryptedPath: "external.enc.yaml", Format: "yaml"},
	}}}, env.publicKey)
	root := t.TempDir()

	err := EncryptFiles(cfg, EncryptRequest{SourceDir: root, KeyFile: env.keyFile, Targets: []string{"first.yaml", external}})
	require.ErrorContains(t, err, "not a file within project root")
	var usage *errx.UsageError
	require.ErrorAs(t, err, &usage)
	require.NoFileExists(t, "first.enc.yaml")

	dup := configWithOwnerRecipient(&config.Config{CurrentDir: ".", Encryption: config.EncryptionConfig{Files: []config.FilePair{
		{PlaintextPath: "shared.yaml", EncryptedPath: "one.enc.yaml", Format: "yaml", PlaintextMode: config.PlaintextInplace},
		{PlaintextPath: "shared.yaml", EncryptedPath: "two.enc.yaml", Format: "yaml", PlaintextMode: config.PlaintextDelivery},
	}}}, env.publicKey)
	err = EncryptFiles(dup, EncryptRequest{SourceDir: root, KeyFile: env.keyFile})
	require.ErrorContains(t, err, "conflicting file pairs")
}

func TestEncryptSourceCoversCiphertextOnlyGroupMappings(t *testing.T) {
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

	// A checkout with only remote.enc.yaml: decrypt delivers the tree, the
	// registered plaintext never exists in the project.
	delivery := t.TempDir()
	require.NoError(t, DecryptFiles(cfg, DecryptRequest{OutputDir: delivery, KeyFile: env.keyFile}))
	writeSourceTree(t, delivery, "remote.yaml", "token: edited\n")

	require.NoError(t, EncryptFiles(cfg, EncryptRequest{SourceDir: delivery, KeyFile: env.keyFile}))
	decrypted, err := seal.DecryptToBytes(seal.DecryptBytesOptions{InputFile: "remote.enc.yaml", OutputFile: "remote.yaml", FormatOverride: "yaml", IdentityBundle: mustTestBundle(t, env.keyFile)})
	require.NoError(t, err)
	require.Equal(t, "token: edited\n", string(decrypted))
}

func TestEncryptSourceRejectsEmptySourceBeforeConfigWork(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	cfg := configWithOwnerRecipient(&config.Config{CurrentDir: ".", Encryption: config.EncryptionConfig{Files: []config.FilePair{
		{PlaintextPath: "secret.yaml", EncryptedPath: "secret.enc.yaml", Format: "yaml"},
	}}}, env.publicKey)
	require.NoError(t, os.WriteFile("secret.yaml", []byte("token: registered\n"), 0600))

	err := EncryptFiles(cfg, EncryptRequest{SourceDir: "  ", SourceSet: true, KeyFile: env.keyFile})
	require.ErrorContains(t, err, "--source requires a directory")
	var usage *errx.UsageError
	require.ErrorAs(t, err, &usage)
	require.NoFileExists(t, "secret.enc.yaml")
}

func TestEncryptSourceDuplicateTreePathsAcrossPairs(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	cfg := configWithOwnerRecipient(&config.Config{CurrentDir: ".", Encryption: config.EncryptionConfig{Files: []config.FilePair{
		{PlaintextPath: "a.yaml", EncryptedPath: "one.enc.yaml", Format: "yaml"},
	}}}, env.publicKey)
	selection := config.ResolvedSelection{FilePairs: []config.ResolvedFilePair{
		{PlaintextPath: "shared.yaml", EncryptedPath: "one.enc.yaml", Format: "yaml"},
		{PlaintextPath: "shared.yaml", EncryptedPath: "two.enc.yaml", Format: "yaml"},
	}}
	err := prepareEncryptSource(cfg, &selection, EncryptRequest{SourceDir: t.TempDir()})
	require.ErrorContains(t, err, "multiple file pairs read from")
}

func TestEncryptSourceWorksWithPatternsAndDirectoryTargets(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	require.NoError(t, os.MkdirAll("config", 0755))
	cfg := configWithOwnerRecipient(&config.Config{CurrentDir: ".", Encryption: config.EncryptionConfig{Files: []config.FilePair{
		{PlaintextPath: "config/first.yaml", EncryptedPath: "config/first.enc.yaml", Format: "yaml"},
		{PlaintextPath: "config/second.yaml", EncryptedPath: "config/second.enc.yaml", Format: "yaml"},
	}}}, env.publicKey)
	root := t.TempDir()
	writeSourceTree(t, root, "config/first.yaml", "token: one\n")
	writeSourceTree(t, root, "config/second.yaml", "token: two\n")

	require.NoError(t, EncryptFiles(cfg, EncryptRequest{SourceDir: root, KeyFile: env.keyFile, Targets: []string{"config/*.yaml"}}))
	require.FileExists(t, "config/first.enc.yaml")
	require.FileExists(t, "config/second.enc.yaml")
}

func TestEncryptSourceKeepsIncrementalUpdateSemantics(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	require.NoError(t, os.WriteFile("secret.yaml", []byte("token: value\n"), 0600))
	cfg := configWithOwnerRecipient(&config.Config{CurrentDir: ".", Encryption: config.EncryptionConfig{Files: []config.FilePair{{PlaintextPath: "secret.yaml", EncryptedPath: "secret.enc.yaml", Format: "yaml"}}}}, env.publicKey)
	require.NoError(t, EncryptFiles(cfg, EncryptRequest{KeyFile: env.keyFile}))
	require.NoError(t, os.Remove("secret.yaml"))
	original, err := os.ReadFile("secret.enc.yaml")
	require.NoError(t, err)

	delivery := t.TempDir()
	require.NoError(t, DecryptFiles(cfg, DecryptRequest{OutputDir: delivery, KeyFile: env.keyFile}))
	require.NoError(t, EncryptFiles(cfg, EncryptRequest{SourceDir: delivery, KeyFile: env.keyFile}))

	current, err := os.ReadFile("secret.enc.yaml")
	require.NoError(t, err)
	require.Equal(t, original, current, "unchanged plaintext from the source tree keeps ciphertext byte-identical")
}
