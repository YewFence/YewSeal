package execx

import (
	"bytes"
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
