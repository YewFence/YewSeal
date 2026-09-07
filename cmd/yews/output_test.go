package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"filippo.io/age"
	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/sopsx"
	toml "github.com/pelletier/go-toml/v2"
	"github.com/stretchr/testify/require"
)

func TestCLIClosedOutputPipes(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "yews.exe")
	output, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput()
	require.NoError(t, err, "%s", output)
	for _, name := range []string{"AGE_KEY_FILE", "YEWSEAL_AGE_IDENTITIES", "SOPS_AGE_KEY", "SOPS_AGE_KEY_FILE", "SOPS_AGE_KEY_CMD", "SOPS_OUTPUT_FILE", "YEWSEAL_STRICT"} {
		t.Setenv(name, "")
		require.NoError(t, os.Unsetenv(name))
	}
	t.Run("init-clean-stdout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, binary, "init", "--input", "config.yaml")
		cmd.Dir = t.TempDir()
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		require.NoError(t, cmd.Run(), stderr.String())
		require.Empty(t, stdout.String())
		require.Equal(t, 1, strings.Count(stderr.String(), "Initialized"))
		require.FileExists(t, filepath.Join(cmd.Dir, ".sops.yaml"))
	})
	for _, tc := range []struct {
		command, channel string
		workers          int
	}{
		{"decrypt", "diagnostics", 1}, {"decrypt", "diagnostics", 4},
		{"encrypt", "diagnostics", 4}, {"view", "content", 0}, {"diff", "content", 0}, {"plan", "content", 0},
	} {
		t.Run(fmt.Sprintf("%s/%s/%d", tc.command, tc.channel, tc.workers), func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0700))
			owner, err := age.GenerateX25519Identity()
			require.NoError(t, err)
			keyFile := filepath.Join(dir, "identity.txt")
			require.NoError(t, os.WriteFile(keyFile, []byte(owner.String()), 0600))
			defaults := []string{"owner"}
			cfg := config.Config{Recipients: config.RecipientConfig{Registry: map[string]string{"owner": owner.Recipient().String()}, Defaults: &defaults}}
			plain := []byte("token: saved\n")
			cipher, err := sopsx.Encrypt(plain, "yaml", []string{owner.Recipient().String()})
			require.NoError(t, err)
			for i := range 8 {
				pair := config.FilePair{PlaintextPath: fmt.Sprintf("%d.yaml", i), EncryptedPath: fmt.Sprintf("%d.enc.yaml", i), Format: "yaml"}
				cfg.Encryption.Files = append(cfg.Encryption.Files, pair)
				if tc.command == "encrypt" || tc.command == "diff" {
					require.NoError(t, os.WriteFile(filepath.Join(dir, pair.PlaintextPath), []byte("token: local\n"), 0600))
				}
				if tc.command != "encrypt" {
					require.NoError(t, os.WriteFile(filepath.Join(dir, pair.EncryptedPath), cipher, 0600))
				}
			}
			data, err := toml.Marshal(cfg)
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(filepath.Join(dir, ".yewseal.toml"), data, 0600))
			args := []string{"--key-file", keyFile, tc.command}
			if tc.workers > 0 {
				args = append(args, "--verbose", "--parallel", fmt.Sprint(tc.workers))
			}
			if tc.command == "view" {
				args = append(args, "0.yaml")
			}
			if tc.command == "diff" {
				args = append(args, "--color", "never")
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, binary, args...)
			cmd.Dir = dir
			reader, writer, err := os.Pipe()
			require.NoError(t, err)
			require.NoError(t, reader.Close())
			defer func() { _ = writer.Close() }()
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			if tc.channel == "content" {
				cmd.Stdout = writer
			} else {
				cmd.Stderr = writer
			}
			err = cmd.Run()
			var exit *exec.ExitError
			require.ErrorAs(t, err, &exit)
			require.Equal(t, 1, exit.ExitCode(), "must return controlled failure, not SIGPIPE termination")
			require.NoError(t, ctx.Err())
			if tc.channel == "content" {
				require.Contains(t, stderr.String(), "failed to write content")
			} else {
				require.Empty(t, stdout.String())
				for _, pair := range cfg.Encryption.Files {
					path := pair.PlaintextPath
					if tc.command == "encrypt" {
						path = pair.EncryptedPath
					}
					data, err := os.ReadFile(filepath.Join(dir, path))
					require.NoError(t, err)
					if tc.command == "encrypt" {
						data, err = sopsx.Decrypt(data, "yaml", owner.String())
						require.NoError(t, err)
						require.Equal(t, "token: local\n", string(data))
					} else {
						require.Equal(t, plain, data)
					}
				}
			}
		})
	}
}
