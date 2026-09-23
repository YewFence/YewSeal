package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

type verifyRun struct {
	stdout, stderr string
	code           int
}

func TestVerifyExitCodesAndChannels(t *testing.T) {
	binary := buildYews(t)
	clearCommandEnvironment(t)
	dir := t.TempDir()
	run := func(args ...string) verifyRun {
		t.Helper()
		ctx, cancel := context.WithTimeout(t.Context(), subprocessTimeout)
		defer cancel()
		cmd := exec.CommandContext(ctx, binary, args...)
		cmd.Dir = dir
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		err := cmd.Run()
		code := 0
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			code = exitErr.ExitCode()
		} else {
			require.NoError(t, err)
		}
		return verifyRun{stdout: stdout.String(), stderr: stderr.String(), code: code}
	}
	mustSucceed := func(args ...string) {
		t.Helper()
		result := run(args...)
		require.Zero(t, result.code, "%s\n%s", result.stdout, result.stderr)
	}
	decodeReport := func(result verifyRun) (report struct {
		OK       bool     `json:"ok"`
		Skipped  []string `json:"skipped"`
		Findings []struct {
			Code string `json:"code"`
		} `json:"findings"`
	}) {
		t.Helper()
		require.NoError(t, json.Unmarshal([]byte(result.stdout), &report), "stdout must be only JSON: %q", result.stdout)
		return report
	}

	plain := []byte("token: supersecret\n")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "config.yaml"), plain, 0600))
	mustSucceed("init", "--input", "config.yaml", "--output", "config.enc.yaml")
	mustSucceed("encrypt")

	healthy := run("verify", "--json")
	require.Zero(t, healthy.code, "%s\n%s", healthy.stdout, healthy.stderr)
	require.Empty(t, healthy.stderr)
	report := decodeReport(healthy)
	require.True(t, report.OK)
	require.Empty(t, report.Findings)
	require.NotEmpty(t, report.Skipped, "the missing repository must be reported as a skip")

	require.NoError(t, os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("token: changed\n"), 0600))
	drifted := run("verify", "--json")
	require.Equal(t, 1, drifted.code, drifted.stderr)
	report = decodeReport(drifted)
	require.False(t, report.OK)
	require.Len(t, report.Findings, 1)
	require.Equal(t, "plaintext_drift", report.Findings[0].Code)
	require.NotContains(t, drifted.stdout+drifted.stderr, "supersecret")
	require.NotContains(t, drifted.stdout+drifted.stderr, "changed\n")

	staticOnly := run("verify", "--no-decrypt")
	require.Zero(t, staticOnly.code, "%s\n%s", staticOnly.stdout, staticOnly.stderr)
	require.Contains(t, staticOnly.stdout, "plaintext consistency not checked")

	conflicting := run("verify", "--decrypt", "--no-decrypt")
	require.Equal(t, 2, conflicting.code)
	require.Empty(t, conflicting.stdout)
	require.Contains(t, conflicting.stderr, "mutually exclusive")

	require.NoError(t, os.Rename(filepath.Join(dir, ".age", "keys.txt"), filepath.Join(dir, "moved-keys.txt")))
	noIdentity := run("verify", "--decrypt")
	require.Equal(t, 2, noIdentity.code)
	require.Empty(t, noIdentity.stdout)
	require.Contains(t, noIdentity.stderr, "--decrypt requires an Age identity")

	withoutIdentity := run("verify")
	require.Zero(t, withoutIdentity.code, "%s\n%s", withoutIdentity.stdout, withoutIdentity.stderr)
	require.Contains(t, withoutIdentity.stdout, "no Age identity available")

	missingConfig := t.TempDir()
	dir = missingConfig
	require.Equal(t, 2, run("verify").code)
}
