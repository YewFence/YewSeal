package agekey

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/YewFence/YewSeal/internal/errx"
	"github.com/YewFence/YewSeal/internal/execx"
)

// GetAgeKey returns the Age private key with the following priority:
// 1. Command-line parameter --key-file (highest)
// 2. Environment variable SOPS_AGE_KEY (direct key value)
// 3. Environment variable SOPS_AGE_KEY_FILE (path to key file)
// 4. Environment variable SOPS_AGE_KEY_CMD (command to output key)
// 5. Default key file .age/keys.txt (lowest)
func GetAgeKey(keyFile string) (string, error) {
	// Priority 1: Command-line parameter
	if keyFile != "" {
		key, err := readKeyFile(keyFile)
		if err == nil {
			return key, nil
		}
		if !isKeyFileNotExist(err) {
			return "", err
		}
	}

	// Priority 2: SOPS_AGE_KEY environment variable (direct key value)
	if key := os.Getenv("SOPS_AGE_KEY"); key != "" {
		return key, nil
	}

	// Priority 3: SOPS_AGE_KEY_FILE environment variable (path to key file)
	if keyFilePath := os.Getenv("SOPS_AGE_KEY_FILE"); keyFilePath != "" {
		key, err := readKeyFile(keyFilePath)
		if err == nil {
			return key, nil
		}
		if !isKeyFileNotExist(err) {
			return "", err
		}
	}

	// Priority 4: SOPS_AGE_KEY_CMD environment variable (command to output key)
	if keyCmd := os.Getenv("SOPS_AGE_KEY_CMD"); keyCmd != "" {
		shell := "sh"
		args := []string{"-c", keyCmd}
		if runtime.GOOS == "windows" {
			shell = "cmd"
			args = []string{"/c", keyCmd}
		}

		stdout, stderr, err := execx.ExecCommand(shell, args...)
		if err != nil {
			return "", &errx.ExternalCommandError{Op: "failed to execute SOPS_AGE_KEY_CMD", Cmd: shell, Args: args, Stderr: stderr, Err: err}
		}
		key := strings.TrimSpace(stdout)
		if key == "" {
			return "", fmt.Errorf("SOPS_AGE_KEY_CMD returned empty output")
		}
		return key, nil
	}

	// Priority 5: Default key file
	defaultPath := ".age/keys.txt"
	if _, err := os.Stat(defaultPath); err == nil {
		return readKeyFile(defaultPath)
	}

	return "", &errx.AgeKeyNotFoundError{Options: []string{"--key-file", "SOPS_AGE_KEY", "SOPS_AGE_KEY_FILE", "SOPS_AGE_KEY_CMD", "or .age/keys.txt"}}
}

func isKeyFileNotExist(err error) bool {
	return errors.Is(err, os.ErrNotExist)
}

// readKeyFile reads an Age private key from a file
func readKeyFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", &errx.KeyFileReadError{Path: path, Err: err}
	}

	// Extract the private key line (starts with AGE-SECRET-KEY-)
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 0 && strings.HasPrefix(line, "AGE-SECRET-KEY-") {
			return line, nil
		}
	}

	return "", &errx.AgeSecretKeyNotFoundError{Path: path}
}
