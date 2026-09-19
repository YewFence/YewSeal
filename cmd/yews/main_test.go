package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const subprocessTimeout = time.Minute

func clearCommandEnvironment(t *testing.T) {
	t.Helper()
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		if !strings.HasPrefix(name, "YEWSEAL_") && !strings.HasPrefix(name, "SOPS_") && name != "VISUAL" && name != "EDITOR" {
			continue
		}
		t.Setenv(name, "")
		require.NoError(t, os.Unsetenv(name))
	}
}

func TestClearCommandEnvironmentUsesNamespaces(t *testing.T) {
	t.Setenv("YEWSEAL_FUTURE_OPTION", "enabled")
	t.Setenv("SOPS_FUTURE_OPTION", "enabled")
	t.Setenv("VISUAL", "editor")
	t.Setenv("UNRELATED_OPTION", "kept")
	clearCommandEnvironment(t)
	for _, name := range []string{"YEWSEAL_FUTURE_OPTION", "SOPS_FUTURE_OPTION", "VISUAL"} {
		_, exists := os.LookupEnv(name)
		require.False(t, exists)
	}
	require.Equal(t, "kept", os.Getenv("UNRELATED_OPTION"))
}

func TestCLIConfigurationLoading(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "yews.exe")
	ctx, cancel := context.WithTimeout(t.Context(), subprocessTimeout)
	defer cancel()
	build := exec.CommandContext(ctx, "go", "build", "-o", binary, ".")
	output, err := build.CombinedOutput()
	require.NoError(t, err, "%s", output)
	clearCommandEnvironment(t)
	t.Setenv("SOPS_AGE_KEY_CMD", "exit 29")

	infoCommands := [][]string{
		{}, {"--version"}, {"--help"}, {"help", "decrypt"},
		{"init", "--help"}, {"encrypt", "--help"}, {"decrypt", "--help"},
		{"clean", "--help"},
		{"plan", "--help"}, {"edit", "--help"}, {"view", "--help"}, {"diff", "--help"},
		{"init", "--help", "--format", "invalid"},
		{"completion", "bash"}, {"completion", "zsh"}, {"completion", "fish"}, {"completion", "powershell"},
		{"__complete", ""}, {"__complete", "decrypt", "--"},
	}
	for _, fixture := range []struct{ name, content string }{
		{"missing", ""},
		{"malformed", "[broken"},
		{"removed-field", "[key]\nfile_path = 'keys.txt'\n"},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0755))
			if fixture.content != "" {
				require.NoError(t, os.WriteFile(filepath.Join(dir, ".yewseal.toml"), []byte(fixture.content), 0600))
			}
			for _, args := range infoCommands {
				t.Run(strings.Join(args, " "), func(t *testing.T) {
					ctx, cancel := context.WithTimeout(t.Context(), subprocessTimeout)
					defer cancel()
					cmd := exec.CommandContext(ctx, binary, args...)
					cmd.Dir = dir
					output, err := cmd.CombinedOutput()
					require.NoError(t, err, "%s", output)
					require.NotEmpty(t, output)
					require.NotContains(t, string(output), "failed to load config")
					if len(args) == 1 && args[0] == "--version" {
						require.Equal(t, "yews version dev\n", string(output))
					}
				})
			}
		})
	}

	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"version-unknown-flag", []string{"--version", "--unknown-option"}, "unknown flag"},
		{"help-invalid-number", []string{"decrypt", "--help", "--parallel", "nope"}, "invalid argument"},
		{"missing-target", []string{"view"}, "accepts 1 arg"},
		{"removed-format", []string{"decrypt", "--format", "yaml"}, "unknown flag: --format"},
		{"missing-edit-file", []string{"edit"}, "edit requires exactly one configured target"},
		{"invalid-color", []string{"diff", "--color", "invalid"}, "unsupported color mode"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(dir, ".yewseal.toml"), []byte("[broken"), 0600))
			ctx, cancel := context.WithTimeout(t.Context(), subprocessTimeout)
			defer cancel()
			cmd := exec.CommandContext(ctx, binary, tc.args...)
			cmd.Dir = dir
			output, err := cmd.CombinedOutput()
			require.Error(t, err)
			require.Contains(t, string(output), tc.want)
			require.NotContains(t, string(output), "failed to load config")
		})
	}

	t.Run("workflow", func(t *testing.T) {
		t.Setenv("YEWSEAL_FORMAT", "invalid")
		t.Setenv("SOPS_FORMAT", "invalid")
		dir := t.TempDir()
		require.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0755))
		plainPath := filepath.Join(dir, "config.yaml")
		plain := []byte("token: value\n")
		require.NoError(t, os.WriteFile(plainPath, plain, 0600))
		run := func(args ...string) []byte {
			t.Helper()
			ctx, cancel := context.WithTimeout(t.Context(), subprocessTimeout)
			defer cancel()
			cmd := exec.CommandContext(ctx, binary, args...)
			cmd.Dir = dir
			output, err := cmd.CombinedOutput()
			require.NoError(t, err, "%s", output)
			return output
		}
		run("init", "--input", "config.yaml", "--output", "config.enc.yaml", "--sync-sops-config=false")
		require.NoFileExists(t, filepath.Join(dir, ".sops.yaml"))
		run("plan")
		run("encrypt")
		require.FileExists(t, filepath.Join(dir, ".sops.yaml"))
		require.NoError(t, os.Remove(plainPath))
		run("decrypt", "--key-file", ".age/keys.txt")
		decrypted, err := os.ReadFile(plainPath)
		require.NoError(t, err)
		require.Equal(t, plain, decrypted)
		require.Equal(t, plain, run("view", "config.enc.yaml", "--key-file", ".age/keys.txt"))
		run("diff", "--key-file", ".age/keys.txt", "--color", "never")
	})

	t.Run("configured-format-and-output", func(t *testing.T) {
		t.Setenv("YEWSEAL_FORMAT", "json")
		t.Setenv("SOPS_FORMAT", "yaml")
		dir := t.TempDir()
		require.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0755))
		plain := []byte("TOKEN=value\n")
		require.NoError(t, os.WriteFile(filepath.Join(dir, "secret"), plain, 0600))
		run := func(args ...string) []byte {
			t.Helper()
			ctx, cancel := context.WithTimeout(t.Context(), subprocessTimeout)
			defer cancel()
			cmd := exec.CommandContext(ctx, binary, args...)
			cmd.Dir = dir
			output, err := cmd.CombinedOutput()
			require.NoError(t, err, "%s", output)
			return output
		}
		run("init", "--input", "secret", "--output", "secret.enc.env", "--format", "env", "--sync-sops-config=false")
		run("encrypt", "secret", "-o", "export.json")
		require.NoFileExists(t, filepath.Join(dir, "secret.enc.env"))
		run("encrypt")
		run("plan", "secret", "--json")
		run("decrypt", "secret.enc.env", "-o", "export.yaml", "--key-file", ".age/keys.txt")
		exported, err := os.ReadFile(filepath.Join(dir, "export.yaml"))
		require.NoError(t, err)
		require.Equal(t, plain, exported)
		require.Equal(t, plain, run("view", "secret.enc.env", "--key-file", ".age/keys.txt"))
		run("diff", "--key-file", ".age/keys.txt", "--color", "never")
	})
}
