package tools

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

// ExecCommand runs a command and returns stdout, stderr, and error
func ExecCommand(name string, args ...string) (string, string, error) {
	cmd := exec.Command(name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// ExecCommandWithEnv runs a command with custom environment variables
func ExecCommandWithEnv(env map[string]string, name string, args ...string) (string, string, error) {
	cmd := exec.Command(name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Copy current environment and add custom vars
	cmd.Env = os.Environ()
	for key, value := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// ExecCommandWithStdin runs a command with data piped to stdin and returns stdout, stderr, and error
func ExecCommandWithStdin(stdin []byte, name string, args ...string) (string, string, error) {
	cmd := exec.Command(name, args...)
	cmd.Stdin = bytes.NewReader(stdin)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// ExecCommandInteractive runs a command with interactive I/O (for editors)
func ExecCommandInteractive(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// ExecCommandInteractiveWithEnv runs a command with interactive I/O and custom environment variables
func ExecCommandInteractiveWithEnv(env map[string]string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Env = os.Environ()
	for key, value := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	return cmd.Run()
}
