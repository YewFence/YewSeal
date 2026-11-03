package main

import (
	"bytes"
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
