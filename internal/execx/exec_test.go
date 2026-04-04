package execx

import (
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecCommand_Success(t *testing.T) {
	var stdout, stderr string
	var err error

	if runtime.GOOS == "windows" {
		stdout, stderr, err = ExecCommand("cmd", "/c", "echo", "hello")
	} else {
		stdout, stderr, err = ExecCommand("echo", "hello")
	}

	require.NoError(t, err)
	assert.Contains(t, stdout, "hello")
	assert.Empty(t, stderr)
}

func TestExecCommand_Failure(t *testing.T) {
	stdout, stderr, err := ExecCommand("nonexistent_command_12345")

	assert.Error(t, err)
	assert.Empty(t, stdout)
	// stderr may or may not have content depending on the error type
	_ = stderr
}

func TestExecCommand_WithArgs(t *testing.T) {
	if runtime.GOOS == "windows" {
		stdout, _, err := ExecCommand("cmd", "/c", "echo", "arg1 arg2")
		require.NoError(t, err)
		assert.Contains(t, stdout, "arg1")
		assert.Contains(t, stdout, "arg2")
	} else {
		stdout, _, err := ExecCommand("echo", "arg1", "arg2")
		require.NoError(t, err)
		assert.Contains(t, stdout, "arg1")
		assert.Contains(t, stdout, "arg2")
	}
}

func TestExecCommand_CapturesStderr(t *testing.T) {
	var stderr string
	var err error

	if runtime.GOOS == "windows" {
		_, stderr, err = ExecCommand("cmd", "/c", "echo error message 1>&2")
	} else {
		_, stderr, err = ExecCommand("sh", "-c", "echo error message >&2")
	}

	require.NoError(t, err)
	assert.Contains(t, stderr, "error message")
}

func TestExecCommand_ExitCode(t *testing.T) {
	var err error

	if runtime.GOOS == "windows" {
		_, _, err = ExecCommand("cmd", "/c", "exit 1")
	} else {
		_, _, err = ExecCommand("sh", "-c", "exit 1")
	}

	assert.Error(t, err)
}

func TestExecCommand_MultilineOutput(t *testing.T) {
	var stdout string
	var err error

	if runtime.GOOS == "windows" {
		stdout, _, err = ExecCommand("cmd", "/c", "echo line1 & echo line2")
	} else {
		stdout, _, err = ExecCommand("sh", "-c", "echo line1; echo line2")
	}

	require.NoError(t, err)
	lines := strings.Split(stdout, "\n")
	assert.GreaterOrEqual(t, len(lines), 2)
}
