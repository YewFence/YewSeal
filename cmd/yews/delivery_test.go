package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"filippo.io/age"
	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/sopsx"
	toml "github.com/pelletier/go-toml/v2"
	"github.com/stretchr/testify/require"
)

func TestCLIPlaintextDeliveryLifecycle(t *testing.T) {
	binary := buildYews(t)
	clearCommandEnvironment(t)
	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	t.Setenv("SOPS_AGE_KEY", identity.String())
	projectRoot := t.TempDir()
	initTestGitRepository(t, projectRoot)
	require.NoError(t, os.Mkdir(filepath.Join(projectRoot, "config"), 0755))
	aliases := []string{"owner"}
	cfg := config.Config{
		Encryption: config.EncryptionConfig{Files: []config.FilePair{
			{PlaintextPath: "config/first.yaml", EncryptedPath: "config/first.enc.yaml"},
			{PlaintextPath: "secret", EncryptedPath: "secret.enc.env", Format: "env", PlaintextMode: "delivery"},
		}},
		Recipients: config.RecipientConfig{Defaults: &aliases, Registry: map[string]string{"owner": identity.Recipient().String()}},
	}
	data, err := toml.Marshal(cfg)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, ".yewseal.toml"), data, 0644))
	for _, pair := range cfg.Encryption.Files {
		format, plain := "yaml", "token: secret\n"
		if pair.Format == "env" {
			format, plain = "env", "TOKEN=secret\n"
		}
		ciphertext, err := sopsx.Encrypt([]byte(plain), format, []string{identity.Recipient().String()})
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(projectRoot, pair.EncryptedPath), ciphertext, 0644))
	}
	run := func(cwd string, code int, args ...string) (string, string) {
		t.Helper()
		ctx, cancel := context.WithTimeout(t.Context(), subprocessTimeout)
		defer cancel()
		cmd := exec.CommandContext(ctx, binary, args...)
		cmd.Dir = cwd
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		err := cmd.Run()
		if code == 0 {
			require.NoError(t, err, "%s", stderr.String())
		} else {
			var exit *exec.ExitError
			require.ErrorAs(t, err, &exit, "%s", stderr.String())
			require.Equal(t, code, exit.ExitCode(), "%s", stderr.String())
		}
		return stdout.String(), stderr.String()
	}
	stdout, stderr := run(projectRoot, 0, "decrypt", "--parallel", "4")
	require.Empty(t, stdout)
	require.Contains(t, stderr, "--output DIR")
	require.Contains(t, stderr, "--inplace")
	require.Contains(t, stderr, "SKIPPED secret.enc.env")
	require.Contains(t, stderr, "1 succeeded, 1 skipped, 0 failed (2 selected)")
	require.FileExists(t, filepath.Join(projectRoot, "config/first.yaml"))
	require.NoFileExists(t, filepath.Join(projectRoot, "secret"))
	require.FileExists(t, filepath.Join(projectRoot, ".gitignore"))
	stdout, stderr = run(projectRoot, 1, "decrypt", "--strict")
	require.Empty(t, stdout)
	require.Contains(t, stderr, "1 of 2 files skipped")
	require.FileExists(t, filepath.Join(projectRoot, "config/first.yaml"))
	require.NoFileExists(t, filepath.Join(projectRoot, "secret"))
	run(projectRoot, 0, "clean")
	ignoreBefore, err := os.ReadFile(filepath.Join(projectRoot, ".gitignore"))
	require.NoError(t, err)

	root := t.TempDir()
	t.Setenv("YEWSEAL_DECRYPT_OUTPUT", root)
	stdout, stderr = run(projectRoot, 0, "decrypt", "--strict", "--parallel", "4", "--force=false")
	require.Empty(t, stdout)
	require.Contains(t, stderr, "2 succeeded")
	for _, pair := range cfg.Encryption.Files {
		require.FileExists(t, filepath.Join(root, pair.PlaintextPath))
		require.NoFileExists(t, filepath.Join(projectRoot, pair.PlaintextPath))
	}
	ignoreAfter, err := os.ReadFile(filepath.Join(projectRoot, ".gitignore"))
	require.NoError(t, err)
	require.Equal(t, ignoreBefore, ignoreAfter)
	require.NoFileExists(t, filepath.Join(projectRoot, ".sops.yaml"))

	otherRoot := t.TempDir()
	stdout, _ = run(filepath.Join(projectRoot, "config"), 0, "decrypt", "first.enc.yaml", "--output", otherRoot)
	require.Empty(t, stdout)
	require.FileExists(t, filepath.Join(otherRoot, "config/first.yaml"))
	require.NoFileExists(t, filepath.Join(otherRoot, "first.yaml"))

	t.Setenv("YEWSEAL_DECRYPT_OUTPUT", "")
	stdout, _ = run(projectRoot, 0, "view", "secret.enc.env")
	require.Equal(t, "TOKEN=secret\n", stdout)
	stdout, _ = run(projectRoot, 0, "decrypt", "--inplace")
	require.Empty(t, stdout)
	require.FileExists(t, filepath.Join(projectRoot, "secret"))
	stdout, _ = run(projectRoot, 0, "clean")
	require.Empty(t, stdout)
	require.NoFileExists(t, filepath.Join(projectRoot, "secret"))
	require.FileExists(t, filepath.Join(root, "secret"))
}
