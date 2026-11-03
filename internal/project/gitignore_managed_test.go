package project

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/stretchr/testify/require"
)

func TestUpdateGitignoreRebuildsOnlyManagedBlock(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".gitignore")
	prefix := "# User rules\r\nconfig.yaml\r\n!config.yaml\r\n\r\n"
	suffix := "# Keep this suffix exactly\r\n/cache/"
	old := "# BEGIN YewSeal managed entries\n/obsolete.yaml\n# END YewSeal managed entries\n"
	require.NoError(t, os.WriteFile(path, []byte(prefix+old+suffix), 0600))
	pairs := []config.FilePair{{PlaintextPath: "second.yaml"}, {PlaintextPath: "config.yaml"}, {PlaintextPath: "second.yaml"}}
	require.NoError(t, UpdateGitignore(dir, pairs))
	want := prefix + "# BEGIN YewSeal managed entries\n/config.yaml\n/second.yaml\n/.age/keys.txt\n# END YewSeal managed entries\n" + suffix
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, want, string(data))
	require.NoError(t, UpdateGitignore(dir, pairs[:1]))
	data, err = os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, prefix+"# BEGIN YewSeal managed entries\n/second.yaml\n/.age/keys.txt\n# END YewSeal managed entries\n"+suffix, string(data))
	require.NoError(t, UpdateGitignore(dir, nil))
	data, err = os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, prefix+"# BEGIN YewSeal managed entries\n/.age/keys.txt\n# END YewSeal managed entries\n"+suffix, string(data))
	require.NoError(t, UpdateGitignore(dir, nil))
	repeated, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, data, repeated)
}

func TestUpdateGitignoreRejectsAmbiguousMarkersWithoutWriting(t *testing.T) {
	for _, content := range []string{
		"# BEGIN YewSeal managed entries\n/old.yaml\n",
		"# END YewSeal managed entries\n",
		"# END YewSeal managed entries\n# BEGIN YewSeal managed entries\n",
		"# BEGIN YewSeal managed entries\n# BEGIN YewSeal managed entries\n# END YewSeal managed entries\n",
		"# BEGIN YewSeal managed entries\n# END YewSeal managed entries\n# END YewSeal managed entries\n",
		"# BEGIN YewSeal managed entries\n# END YewSeal managed entries\n# BEGIN YewSeal managed entries\n# END YewSeal managed entries\n",
	} {
		t.Run(content, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, ".gitignore")
			require.NoError(t, os.WriteFile(path, []byte(content), 0600))
			err := UpdateGitignore(dir, []config.FilePair{{PlaintextPath: "config.yaml"}})
			require.ErrorContains(t, err, "YewSeal managed block")
			data, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, content, string(data))
		})
	}
}

func TestUpdateGitignoreAcceptsCRLFMarkersAndEndAtEOF(t *testing.T) {
	for _, block := range []string{
		"# BEGIN YewSeal managed entries\r\n/old.yaml\r\n# END YewSeal managed entries\r\n",
		"# BEGIN YewSeal managed entries\n/old.yaml\n# END YewSeal managed entries",
	} {
		dir := t.TempDir()
		path := filepath.Join(dir, ".gitignore")
		require.NoError(t, os.WriteFile(path, []byte(block), 0600))
		require.NoError(t, UpdateGitignore(dir, []config.FilePair{{PlaintextPath: "config.yaml"}}))
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, "# BEGIN YewSeal managed entries\n/config.yaml\n/.age/keys.txt\n# END YewSeal managed entries\n", string(data))
	}
}

func TestUpdateGitignoreMarkersMustOccupyWholeLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".gitignore")
	custom := "# Notes: # BEGIN YewSeal managed entries\n # END YewSeal managed entries\n"
	require.NoError(t, os.WriteFile(path, []byte(custom), 0600))
	require.NoError(t, UpdateGitignore(dir, []config.FilePair{{PlaintextPath: "config.yaml"}}))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, custom+"\n# BEGIN YewSeal managed entries\n/config.yaml\n/.age/keys.txt\n# END YewSeal managed entries\n", string(data))
}

func TestUpdateGitignoreEscapesLiteralPaths(t *testing.T) {
	dir := t.TempDir()
	output, err := exec.Command("git", "init", "-q", dir).CombinedOutput()
	require.NoError(t, err, "%s", output)
	paths := []string{"*.yaml", "?.yaml", "[ab].yaml", "!secret.yaml", "#secret.yaml", "space name .yaml", "trailing ", `back\slash.yaml`, "config.yaml"}
	pairs := make([]config.FilePair, 0, len(paths))
	for _, path := range paths {
		pairs = append(pairs, config.FilePair{PlaintextPath: path})
	}
	require.NoError(t, UpdateGitignore(dir, pairs))
	for _, path := range append(paths, ".age/keys.txt") {
		cmd := exec.Command("git", "-C", dir, "check-ignore", "--no-index", "-q", "--", path)
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "literal path %q must be ignored: %s", path, output)
	}
	for _, path := range []string{"other.yaml", "a.yaml", "nested/config.yaml", "nested/.age/keys.txt"} {
		cmd := exec.Command("git", "-C", dir, "check-ignore", "--no-index", "-q", "--", path)
		output, err := cmd.CombinedOutput()
		var exit *exec.ExitError
		require.ErrorAs(t, err, &exit, "unrelated path %q must not be ignored: %s", path, output)
		require.Equal(t, 1, exit.ExitCode())
	}
}

func TestUpdateGitignoreReportsFilesystemErrors(t *testing.T) {
	t.Run("read", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, ".gitignore")
		require.NoError(t, os.Mkdir(path, 0700))
		err := UpdateGitignore(dir, []config.FilePair{{PlaintextPath: "config.yaml"}})
		require.ErrorContains(t, err, "failed to read")
		require.DirExists(t, path)
	})
	t.Run("write", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "missing")
		err := UpdateGitignore(dir, []config.FilePair{{PlaintextPath: "config.yaml"}})
		require.ErrorContains(t, err, "failed to write")
		require.ErrorIs(t, err, os.ErrNotExist)
		require.NoDirExists(t, dir)
	})
}

func TestUpdateGitignorePreservesCustomContentWithoutFinalNewline(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".gitignore")
	require.NoError(t, os.WriteFile(path, []byte("# Custom rules\ncache/"), 0600))
	require.NoError(t, UpdateGitignore(dir, []config.FilePair{{PlaintextPath: "config.yaml"}}))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "# Custom rules\ncache/\n\n# BEGIN YewSeal managed entries\n/config.yaml\n/.age/keys.txt\n# END YewSeal managed entries\n", string(data))
}

func TestUpdateGitignoreRejectsLineBreaksInPaths(t *testing.T) {
	for _, path := range []string{"secret\n!other.yaml", "secret\r.yaml"} {
		dir := t.TempDir()
		err := UpdateGitignore(dir, []config.FilePair{{PlaintextPath: path}})
		require.ErrorContains(t, err, "line break")
		require.NoFileExists(t, filepath.Join(dir, ".gitignore"))
	}
}
