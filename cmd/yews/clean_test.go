package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"filippo.io/age"
	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/sopsx"
	toml "github.com/pelletier/go-toml/v2"
	"github.com/stretchr/testify/require"
)

type cleanE2E struct {
	t          *testing.T
	binary     string
	dir        string
	owner      *age.X25519Identity
	pairs      []config.FilePair
	withKey    bool
	withConfig bool
}

func newCleanE2E(t *testing.T, binary string) *cleanE2E {
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0755))
	owner, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	return &cleanE2E{t: t, binary: binary, dir: dir, owner: owner}
}

// writeRegistered 落盘一对由 owner 加密的映射并登记到 .yewseal.toml。
func (e *cleanE2E) writeRegistered(plainName string, plain, local []byte) {
	e.t.Helper()
	cipher, err := sopsx.Encrypt(plain, "yaml", []string{e.owner.Recipient().String()})
	require.NoError(e.t, err)
	cipherName := strings.Replace(plainName, ".yaml", ".enc.yaml", 1)
	require.NoError(e.t, os.WriteFile(filepath.Join(e.dir, cipherName), cipher, 0600))
	if local != nil {
		require.NoError(e.t, os.WriteFile(filepath.Join(e.dir, plainName), local, 0600))
	}
	e.pairs = append(e.pairs, config.FilePair{PlaintextPath: plainName, EncryptedPath: cipherName, Format: "yaml"})
	defaults := []string{"owner"}
	cfg := config.Config{
		Encryption: config.EncryptionConfig{Files: e.pairs},
		Recipients: config.RecipientConfig{Defaults: &defaults, Registry: map[string]string{"owner": e.owner.Recipient().String()}},
	}
	data, err := toml.Marshal(cfg)
	require.NoError(e.t, err)
	require.NoError(e.t, os.WriteFile(filepath.Join(e.dir, ".yewseal.toml"), data, 0600))
	e.withConfig = true
}

// provideKey 落盘 cwd 级 .age/keys.txt 作为 identity 来源。
func (e *cleanE2E) provideKey() {
	e.t.Helper()
	e.withKey = true
	require.NoError(e.t, os.Mkdir(filepath.Join(e.dir, ".age"), 0700))
	require.NoError(e.t, os.WriteFile(filepath.Join(e.dir, ".age/keys.txt"), []byte(e.owner.String()), 0600))
}

func (e *cleanE2E) run(stdin string, env []string, args ...string) (stdout, stderr string, code int) {
	e.t.Helper()
	require.True(e.t, e.withConfig, "scenario must register at least one mapping")
	cmd := exec.Command(e.binary, args...)
	cmd.Dir = e.dir
	cmd.Stdin = strings.NewReader(stdin)
	cmd.Env = append(os.Environ(), env...)
	var out, errBuf strings.Builder
	cmd.Stdout, cmd.Stderr = &out, &errBuf
	err := cmd.Run()
	if err != nil {
		exit, ok := err.(*exec.ExitError)
		require.True(e.t, ok, "%s\n%s", out.String(), errBuf.String())
		code = exit.ExitCode()
	}
	return out.String(), errBuf.String(), code
}

func TestCLICleanEndToEnd(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "yews.exe")
	output, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput()
	require.NoError(t, err, "%s", output)
	clearCommandEnvironment(t)
	same := []byte("token: stable\n")
	wip := []byte("token: original\n")
	changed := []byte("token: wip\n")

	t.Run("default-removes-matching", func(t *testing.T) {
		e := newCleanE2E(t, binary)
		e.writeRegistered("same.yaml", same, same)
		e.provideKey()
		stdout, stderr, code := e.run("", nil, "clean")
		require.Equal(t, 0, code, stderr)
		require.Empty(t, stdout)
		require.Contains(t, stderr, "REMOVED same.yaml")
		require.Contains(t, stderr, "Summary (cleaned): 1 removed, 0 already absent, 0 retained, 0 failed (1 selected)")
		require.NoFileExists(t, filepath.Join(e.dir, "same.yaml"))
		require.FileExists(t, filepath.Join(e.dir, "same.enc.yaml"))
		require.NoFileExists(t, filepath.Join(e.dir, ".gitignore"), "clean must not touch project metadata")
	})

	t.Run("alias-c", func(t *testing.T) {
		e := newCleanE2E(t, binary)
		e.writeRegistered("same.yaml", same, same)
		e.provideKey()
		_, stderr, code := e.run("", nil, "c")
		require.Equal(t, 0, code, stderr)
		require.Contains(t, stderr, "REMOVED same.yaml")
		require.NoFileExists(t, filepath.Join(e.dir, "same.yaml"))
	})

	t.Run("default-prompt-no-retains", func(t *testing.T) {
		e := newCleanE2E(t, binary)
		e.writeRegistered("wip.yaml", wip, changed)
		e.provideKey()
		stdout, stderr, code := e.run("n\n", nil, "clean", "wip.yaml")
		require.Equal(t, 0, code, stderr)
		require.Empty(t, stdout)
		require.Contains(t, stderr, "Delete the local plaintext anyway? [y/N]:")
		require.Contains(t, stderr, "RETAINED wip.yaml: plaintext differs from encrypted content")
		require.Contains(t, stderr, "1 retained, 0 failed")
		require.FileExists(t, filepath.Join(e.dir, "wip.yaml"))
	})

	t.Run("default-prompt-yes-removes", func(t *testing.T) {
		e := newCleanE2E(t, binary)
		e.writeRegistered("wip.yaml", wip, changed)
		e.provideKey()
		_, stderr, code := e.run("y\n", nil, "clean")
		require.Equal(t, 0, code, stderr)
		require.Contains(t, stderr, "REMOVED wip.yaml")
		require.NoFileExists(t, filepath.Join(e.dir, "wip.yaml"))
	})

	t.Run("default-prompt-eof-fails", func(t *testing.T) {
		e := newCleanE2E(t, binary)
		e.writeRegistered("wip.yaml", wip, changed)
		e.provideKey()
		stdout, stderr, code := e.run("", nil, "clean")
		require.Equal(t, 1, code, stderr)
		require.Empty(t, stdout)
		require.Contains(t, stderr, "FAILED wip.yaml")
		require.FileExists(t, filepath.Join(e.dir, "wip.yaml"))
	})

	t.Run("skip-different", func(t *testing.T) {
		e := newCleanE2E(t, binary)
		e.writeRegistered("same.yaml", same, same)
		e.writeRegistered("wip.yaml", wip, changed)
		e.provideKey()
		stdout, stderr, code := e.run("", nil, "clean", "--skip-different")
		require.Equal(t, 0, code, stderr)
		require.Empty(t, stdout)
		require.Contains(t, stderr, "REMOVED same.yaml")
		require.Contains(t, stderr, "RETAINED wip.yaml")
		require.NotContains(t, stderr, "Delete the local plaintext anyway?")
		require.NoFileExists(t, filepath.Join(e.dir, "same.yaml"))
		require.FileExists(t, filepath.Join(e.dir, "wip.yaml"))
	})

	t.Run("force", func(t *testing.T) {
		e := newCleanE2E(t, binary)
		e.writeRegistered("same.yaml", same, same)
		e.writeRegistered("wip.yaml", wip, changed)
		e.provideKey()
		stdout, stderr, code := e.run("", nil, "clean", "--force")
		require.Equal(t, 0, code, stderr)
		require.Empty(t, stdout)
		require.NotContains(t, stderr, "Delete the local plaintext anyway?")
		require.NoFileExists(t, filepath.Join(e.dir, "same.yaml"))
		require.NoFileExists(t, filepath.Join(e.dir, "wip.yaml"))
	})

	t.Run("force-keeps-unrecoverable", func(t *testing.T) {
		e := newCleanE2E(t, binary)
		e.writeRegistered("same.yaml", same, same)
		require.NoError(t, os.Remove(filepath.Join(e.dir, "same.enc.yaml")))
		e.provideKey()
		_, stderr, code := e.run("", nil, "clean", "--force")
		require.Equal(t, 1, code, stderr)
		require.Contains(t, stderr, "FAILED same.yaml: encrypted file is missing")
		require.FileExists(t, filepath.Join(e.dir, "same.yaml"))
	})

	t.Run("force-keeps-foreign-identity", func(t *testing.T) {
		e := newCleanE2E(t, binary)
		e.writeRegistered("same.yaml", same, same)
		e.provideKey()
		_, stderr, code := e.run("", nil, "clean", "--force", "-k", "nonexistent.keys")
		require.Equal(t, 2, code, stderr)
		require.FileExists(t, filepath.Join(e.dir, "same.yaml"))
	})

	t.Run("strategy-flags-conflict", func(t *testing.T) {
		e := newCleanE2E(t, binary)
		e.writeRegistered("same.yaml", same, same)
		e.provideKey()
		_, stderr, code := e.run("", nil, "clean", "--force", "--skip-different")
		require.Equal(t, 2, code, stderr)
		require.Contains(t, stderr, "cannot both be true")
		require.FileExists(t, filepath.Join(e.dir, "same.yaml"))
	})

	t.Run("strategy-env-conflict", func(t *testing.T) {
		e := newCleanE2E(t, binary)
		e.writeRegistered("same.yaml", same, same)
		e.provideKey()
		_, stderr, code := e.run("", []string{"YEWSEAL_CLEAN_FORCE=true"}, "clean", "--skip-different")
		require.Equal(t, 2, code, stderr)
		require.Contains(t, stderr, "cannot both be true")
	})

	t.Run("already-absent-needs-no-key", func(t *testing.T) {
		e := newCleanE2E(t, binary)
		e.writeRegistered("same.yaml", same, nil)
		_, stderr, code := e.run("", nil, "clean", "--verbose")
		require.Equal(t, 0, code, stderr)
		require.Contains(t, stderr, "ALREADY ABSENT same.yaml")
		require.Contains(t, stderr, "0 removed, 1 already absent, 0 retained, 0 failed (1 selected)")
	})

	t.Run("idempotent-second-run", func(t *testing.T) {
		e := newCleanE2E(t, binary)
		e.writeRegistered("same.yaml", same, same)
		e.provideKey()
		_, _, code := e.run("", nil, "clean")
		require.Equal(t, 0, code)
		_, stderr, code := e.run("", nil, "clean", "--verbose")
		require.Equal(t, 0, code, stderr)
		require.Contains(t, stderr, "ALREADY ABSENT same.yaml")
	})
}
