package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCLIInitOutput(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	binary := filepath.Join(t.TempDir(), "yews.exe")
	output, err := exec.CommandContext(ctx, "go", "build", "-o", binary, ".").CombinedOutput()
	require.NoError(t, err, "%s", output)
	for _, name := range []string{"AGE_KEY_FILE", "YEWSEAL_AGE_IDENTITIES", "SOPS_AGE_KEY", "SOPS_AGE_KEY_FILE", "SOPS_AGE_KEY_CMD", "SOPS_OUTPUT_FILE", "YEWSEAL_STRICT"} {
		t.Setenv(name, "")
		require.NoError(t, os.Unsetenv(name))
	}
	t.Run("init-clean-stdout", func(t *testing.T) {
		cmd := exec.CommandContext(ctx, binary, "init", "--input", "config.yaml")
		cmd.Dir = t.TempDir()
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		require.NoError(t, cmd.Run(), stderr.String())
		require.Empty(t, stdout.String())
		require.Equal(t, 1, strings.Count(stderr.String(), "Initialized"))
		require.FileExists(t, filepath.Join(cmd.Dir, ".sops.yaml"))
	})
}
