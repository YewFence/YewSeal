package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunMigratesRepositoryAndReportsSkippedFiles(t *testing.T) {
	root := t.TempDir()
	output, err := exec.Command("git", "init", "-q", root).CombinedOutput()
	require.NoError(t, err, "%s", output)
	child := filepath.Join(root, "service")
	oldCwd := filepath.Join(child, "nested")
	ambiguous := filepath.Join(root, "ambiguous")
	for _, dir := range []string{oldCwd, ambiguous} {
		require.NoError(t, os.MkdirAll(dir, 0700))
	}
	for _, dir := range []string{root, child} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, ".yewseal.toml"), []byte("[[encryption.files]]\nplaintext = 'current.yaml'\nencrypted = 'current.enc.yaml'\n"), 0600))
	}
	legacy := "# YewSeal - Decrypted configuration files\nobsolete.yaml\n\n# YewSeal - Age private keys\n.age/keys.txt\n"
	for _, dir := range []string{root, child, oldCwd} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("# Custom\ncache/\n"+legacy), 0600))
	}
	broken := "# YewSeal - Decrypted configuration files\nhandwritten.yaml\n"
	require.NoError(t, os.WriteFile(filepath.Join(ambiguous, ".gitignore"), []byte(broken), 0600))
	t.Chdir(oldCwd)
	var out bytes.Buffer
	require.Error(t, run(&out))
	require.Contains(t, out.String(), "3 updated, 1 skipped")
	require.Contains(t, out.String(), "Audit the version-control diff")
	require.Contains(t, out.String(), "No commit was created")
	for _, dir := range []string{root, child} {
		data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
		require.NoError(t, err)
		require.Equal(t, "# Custom\ncache/\n\n# BEGIN YewSeal managed entries\n/current.yaml\n/.age/keys.txt\n# END YewSeal managed entries\n", string(data))
	}
	data, err := os.ReadFile(filepath.Join(oldCwd, ".gitignore"))
	require.NoError(t, err)
	require.Equal(t, "# Custom\ncache/\n", string(data))
	data, err = os.ReadFile(filepath.Join(ambiguous, ".gitignore"))
	require.NoError(t, err)
	require.Equal(t, broken, string(data))
	out.Reset()
	require.Error(t, run(&out))
	require.Contains(t, out.String(), "0 updated, 1 skipped")
}

func TestRunPreservesIndependentChildProjects(t *testing.T) {
	for _, parentRepo := range []bool{false, true} {
		for _, childKind := range []string{"config", ".config", ".yewseal", "git-directory", "git-file", "jj"} {
			t.Run(fmt.Sprintf("repository=%t/child=%s", parentRepo, childKind), func(t *testing.T) {
				root := t.TempDir()
				if parentRepo {
					output, err := exec.Command("git", "init", "-q", root).CombinedOutput()
					require.NoError(t, err, "%s", output)
				}
				child := filepath.Join(root, "child")
				nested := filepath.Join(child, "nested")
				require.NoError(t, os.MkdirAll(nested, 0700))
				configText := []byte("[[encryption.files]]\nplaintext = 'current.yaml'\nencrypted = 'current.enc.yaml'\n")
				require.NoError(t, os.WriteFile(filepath.Join(root, ".yewseal.toml"), configText, 0600))
				childConfig := filepath.Join(child, ".yewseal.toml")
				switch childKind {
				case ".config", ".yewseal":
					require.NoError(t, os.Mkdir(filepath.Join(child, childKind), 0700))
					childConfig = filepath.Join(child, childKind, ".yewseal.toml")
				case "git-directory":
					output, err := exec.Command("git", "init", "-q", child).CombinedOutput()
					require.NoError(t, err, "%s", output)
				case "git-file":
					require.NoError(t, os.WriteFile(filepath.Join(child, ".git"), []byte("gitdir: fixture\n"), 0600))
				case "jj":
					require.NoError(t, os.Mkdir(filepath.Join(child, ".jj"), 0700))
				}
				require.NoError(t, os.WriteFile(childConfig, configText, 0600))
				legacy := []byte("# YewSeal - Decrypted configuration files\ncurrent.yaml\n\n# YewSeal - Age private keys\n.age/keys.txt\n")
				for _, dir := range []string{child, nested} {
					require.NoError(t, os.WriteFile(filepath.Join(dir, ".gitignore"), legacy, 0600))
				}
				if parentRepo && (childKind == "config" || childKind == ".config" || childKind == ".yewseal") {
					require.NoError(t, os.WriteFile(filepath.Join(root, ".gitignore"), []byte("child/\n"), 0600))
				}
				t.Chdir(root)
				var out bytes.Buffer
				require.ErrorContains(t, run(&out), "independent project requires separate migration")
				require.Contains(t, out.String(), "1 updated, 1 skipped")
				require.Contains(t, out.String(), child)
				for _, dir := range []string{child, nested} {
					data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
					require.NoError(t, err)
					require.Equal(t, legacy, data)
				}
				data, err := os.ReadFile(filepath.Join(root, ".gitignore"))
				require.NoError(t, err)
				require.Contains(t, string(data), "\n/current.yaml\n")
			})
		}
	}
}

func TestRunMigratesLoadedHiddenConfiguration(t *testing.T) {
	for _, location := range []string{".config", ".yewseal"} {
		t.Run(location, func(t *testing.T) {
			root := t.TempDir()
			t.Chdir(root)
			require.NoError(t, os.Mkdir(location, 0700))
			require.NoError(t, os.WriteFile(filepath.Join(location, ".yewseal.toml"), []byte("[[encryption.files]]\nplaintext = 'current.yaml'\nencrypted = 'current.enc.yaml'\n"), 0600))
			var out bytes.Buffer
			require.NoError(t, run(&out))
			require.Contains(t, out.String(), "1 updated, 0 skipped")
			data, err := os.ReadFile(".gitignore")
			require.NoError(t, err)
			require.Contains(t, string(data), "\n/current.yaml\n")
			require.NoFileExists(t, filepath.Join(location, ".gitignore"))
		})
	}
}

func TestExecute(t *testing.T) {
	for _, tc := range []struct {
		name       string
		args       []string
		config     string
		wantCode   int
		wantStdout string
		wantStderr string
	}{
		{name: "success", config: "[[encryption.files]]\nplaintext = 'current.yaml'\nencrypted = 'current.enc.yaml'\n", wantStdout: "1 updated, 0 skipped"},
		{name: "help", args: []string{"-h"}, wantStderr: "Usage of migrate-gitignore"},
		{name: "unknown flag", args: []string{"-unknown"}, wantCode: 2, wantStderr: "flag provided but not defined"},
		{name: "positional argument", args: []string{"project"}, wantCode: 2, wantStderr: "unexpected arguments; use -dir PATH"},
		{name: "missing directory", args: []string{"-dir", "missing"}, wantCode: 1, wantStderr: "missing"},
		{name: "missing config", wantCode: 1, wantStdout: "Audit the version-control diff", wantStderr: "no YewSeal configuration found"},
		{name: "invalid config", config: "[broken", wantCode: 1, wantStdout: "Audit the version-control diff", wantStderr: "failed to parse config"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			t.Chdir(root)
			if tc.config != "" {
				require.NoError(t, os.WriteFile(".yewseal.toml", []byte(tc.config), 0600))
			}
			var stdout, stderr bytes.Buffer
			require.Equal(t, tc.wantCode, execute(tc.args, &stdout, &stderr))
			if tc.wantStdout != "" {
				require.Contains(t, stdout.String(), tc.wantStdout)
			} else {
				require.Empty(t, stdout.String())
			}
			if tc.wantStderr != "" {
				require.Contains(t, stderr.String(), tc.wantStderr)
			} else {
				require.Empty(t, stderr.String())
			}
			if tc.wantCode != 0 || tc.name == "help" {
				require.NoFileExists(t, ".gitignore")
			} else {
				data, err := os.ReadFile(".gitignore")
				require.NoError(t, err)
				require.Contains(t, string(data), "\n/current.yaml\n")
			}
		})
	}
}

func TestExecuteResolvesDirectoryArgument(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	project := filepath.Join(root, "project with spaces")
	require.NoError(t, os.Mkdir(project, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(project, ".yewseal.toml"), []byte("[[encryption.files]]\nplaintext = 'current.yaml'\nencrypted = 'current.enc.yaml'\n"), 0600))
	var stdout, stderr bytes.Buffer
	require.Zero(t, execute([]string{"-dir", project}, &stdout, &stderr))
	require.Empty(t, stderr.String())
	require.Contains(t, stdout.String(), filepath.Join(project, ".gitignore"))
	require.FileExists(t, filepath.Join(project, ".gitignore"))
	require.NoFileExists(t, filepath.Join(root, ".gitignore"))
}

func TestRunRequiresCompleteDiscovery(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	require.NoError(t, os.Mkdir(".git", 0700))
	require.NoError(t, os.WriteFile(".yewseal.toml", []byte("[[encryption.files]]\nplaintext = 'current.yaml'\nencrypted = 'current.enc.yaml'\n"), 0600))
	before := []byte("# Custom rules\ncache/\n")
	require.NoError(t, os.WriteFile(".gitignore", before, 0600))
	var out bytes.Buffer
	require.ErrorContains(t, run(&out), "migration requires complete configuration discovery")
	require.Contains(t, out.String(), "Audit the version-control diff")
	data, err := os.ReadFile(".gitignore")
	require.NoError(t, err)
	require.Equal(t, before, data)
}

func TestRunReportsSelectionFailure(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	require.NoError(t, os.WriteFile(".yewseal.toml", []byte("[[encryption.files]]\nplaintext = 'current.unknown'\nencrypted = 'current.enc.unknown'\n"), 0600))
	var out bytes.Buffer
	require.ErrorContains(t, run(&out), "could not detect format")
	require.Contains(t, out.String(), "Audit the version-control diff")
	require.NoFileExists(t, ".gitignore")
}

type failedOutput struct{ err error }

func (w failedOutput) Write([]byte) (int, error) { return 0, w.err }

func TestRunPropagatesOutputFailureAfterMigration(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	require.NoError(t, os.WriteFile(".yewseal.toml", []byte("[[encryption.files]]\nplaintext = 'current.yaml'\nencrypted = 'current.enc.yaml'\n"), 0600))
	outputErr := errors.New("output unavailable")
	require.ErrorIs(t, run(failedOutput{err: outputErr}), outputErr)
	data, err := os.ReadFile(".gitignore")
	require.NoError(t, err)
	require.Contains(t, string(data), "\n/current.yaml\n")
}

func TestRunReportsUnresolvableConfigurationAndMigratesAccessibleFiles(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	require.NoError(t, os.WriteFile(".yewseal.toml", []byte("[[encryption.files]]\nplaintext = 'current.yaml'\nencrypted = 'current.enc.yaml'\n"), 0600))
	blocked := filepath.Join(root, "blocked")
	require.NoError(t, os.Mkdir(blocked, 0700))
	before := []byte("# YewSeal - Decrypted configuration files\nsecret.yaml\n\n# YewSeal - Age private keys\n.age/keys.txt\n")
	path := filepath.Join(blocked, ".gitignore")
	require.NoError(t, os.WriteFile(path, before, 0600))
	require.NoError(t, os.Symlink(".config", filepath.Join(blocked, ".config")))
	var out bytes.Buffer
	err := run(&out)
	require.ErrorContains(t, err, "scan "+blocked)
	require.Contains(t, out.String(), "1 updated")
	require.Contains(t, out.String(), "Audit the version-control diff")
	require.FileExists(t, filepath.Join(root, ".gitignore"))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, before, data)
}

func TestRunSkipsSymlinkedGitignore(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	require.NoError(t, os.WriteFile(".yewseal.toml", []byte("[[encryption.files]]\nplaintext = 'current.yaml'\nencrypted = 'current.enc.yaml'\n"), 0600))
	external := filepath.Join(t.TempDir(), "ignore")
	require.NoError(t, os.WriteFile(external, []byte("custom\n"), 0600))
	require.NoError(t, os.Symlink(external, ".gitignore"))
	var out bytes.Buffer
	require.Error(t, run(&out))
	require.Contains(t, out.String(), "not a regular file")
	data, err := os.ReadFile(external)
	require.NoError(t, err)
	require.Equal(t, "custom\n", string(data))
}
