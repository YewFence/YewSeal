package main

import (
	"bytes"
	"context"
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
	binary := buildYews(t)
	clearCommandEnvironment(t)
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
