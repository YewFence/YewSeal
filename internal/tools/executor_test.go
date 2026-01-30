package tools

import (
	"os"
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

func TestExecCommandWithEnv_Success(t *testing.T) {
	env := map[string]string{
		"TEST_VAR_12345": "test_value",
	}

	var stdout string
	var err error

	if runtime.GOOS == "windows" {
		stdout, _, err = ExecCommandWithEnv(env, "cmd", "/c", "echo", "%TEST_VAR_12345%")
	} else {
		stdout, _, err = ExecCommandWithEnv(env, "sh", "-c", "echo $TEST_VAR_12345")
	}

	require.NoError(t, err)
	assert.Contains(t, stdout, "test_value")
}

func TestExecCommandWithEnv_InheritsParentEnv(t *testing.T) {
	// Set an env var in the parent process
	key := "PARENT_TEST_VAR_12345"
	os.Setenv(key, "parent_value")
	defer os.Unsetenv(key)

	var stdout string
	var err error

	if runtime.GOOS == "windows" {
		stdout, _, err = ExecCommandWithEnv(nil, "cmd", "/c", "echo", "%"+key+"%")
	} else {
		stdout, _, err = ExecCommandWithEnv(nil, "sh", "-c", "echo $"+key)
	}

	require.NoError(t, err)
	assert.Contains(t, stdout, "parent_value")
}

func TestExecCommandWithEnv_EmptyEnv(t *testing.T) {
	var stdout string
	var err error

	if runtime.GOOS == "windows" {
		stdout, _, err = ExecCommandWithEnv(map[string]string{}, "cmd", "/c", "echo", "test")
	} else {
		stdout, _, err = ExecCommandWithEnv(map[string]string{}, "echo", "test")
	}

	require.NoError(t, err)
	assert.Contains(t, stdout, "test")
}

func TestExecCommand_CapturesStderr(t *testing.T) {
	var stderr string
	var err error

	if runtime.GOOS == "windows" {
		// On Windows, redirect to stderr using cmd
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
