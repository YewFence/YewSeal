package crypto

import (
	"fmt"
	"os"
	"strings"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/errx"
	"github.com/YewFence/YewSeal/internal/tools"
)

// ExtractPublicKey extracts the public key from age-keygen output or key file
func ExtractPublicKey(output string) string {
	// age-keygen outputs: "# public key: age1..."
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# public key: ") {
			return strings.TrimPrefix(line, "# public key: ")
		}
	}
	return ""
}

// GetAgeKey returns the Age private key with the following priority:
// 1. Command-line parameter --key-file (highest)
// 2. Environment variable SOPS_AGE_KEY (direct key value)
// 3. Environment variable SOPS_AGE_KEY_FILE (path to key file)
// 4. Environment variable SOPS_AGE_KEY_CMD (command to output key)
// 5. Default key file .age/keys.txt (lowest)
func GetAgeKey(keyFile string) (string, error) {
	// Priority 1: Command-line parameter
	if keyFile != "" {
		return readKeyFile(keyFile)
	}

	// Priority 2: SOPS_AGE_KEY environment variable (direct key value)
	if key := os.Getenv("SOPS_AGE_KEY"); key != "" {
		return key, nil
	}

	// Priority 3: SOPS_AGE_KEY_FILE environment variable (path to key file)
	if keyFilePath := os.Getenv("SOPS_AGE_KEY_FILE"); keyFilePath != "" {
		return readKeyFile(keyFilePath)
	}

	// Priority 4: SOPS_AGE_KEY_CMD environment variable (command to output key)
	if keyCmd := os.Getenv("SOPS_AGE_KEY_CMD"); keyCmd != "" {
		stdout, stderr, err := tools.ExecCommand("sh", "-c", keyCmd)
		if err != nil {
			return "", &errx.ExternalCommandError{Op: "failed to execute SOPS_AGE_KEY_CMD", Cmd: "sh", Args: []string{"-c", keyCmd}, Stderr: stderr, Err: err}
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

// GetPublicKey returns the Age public key with priority:
// 1. Command-line parameter (highest)
// 2. Environment variable SOPS_AGE_RECIPIENTS
// 3. Config file .yewseal.toml
// 4. Extract from private key file .age/keys.txt (fallback)
func GetPublicKey(providedKey, keyFile string, verbose bool) (string, error) {
	// Priority 1: Command-line parameter
	if providedKey != "" {
		if verbose {
			fmt.Println("🔑 Using public key from command-line parameter")
		}
		return providedKey, nil
	}

	// Priority 2: Environment variable
	if envKey := os.Getenv("SOPS_AGE_RECIPIENTS"); envKey != "" {
		if verbose {
			fmt.Println("🔑 Using public key from SOPS_AGE_RECIPIENTS environment variable")
		}
		return envKey, nil
	}

	// Priority 3: Config file
	cfg, err := config.LoadConfig()
	if err == nil && cfg.GetPublicKey() != "" {
		if verbose {
			fmt.Println("🔑 Using public key from .yewseal.toml")
		}
		return cfg.GetPublicKey(), nil
	}

	// Priority 4: Extract from private key file (fallback)
	if verbose {
		fmt.Println("🔑 Attempting to extract public key from private key file...")
	}

	// Try to get the key file path
	if keyFile == "" {
		if cfg != nil {
			keyFile = cfg.GetKeyFile("")
		} else {
			keyFile = ".age/keys.txt"
		}
	}

	// Read the private key file
	keyContent, err := os.ReadFile(keyFile)
	if err != nil {
		return "", &errx.PublicKeyNotFoundError{KeyFile: keyFile, Cause: err, Tried: []string{"CLI parameter", "SOPS_AGE_RECIPIENTS env", ".yewseal.toml config"}}
	}

	// Extract public key from the key file content
	publicKey := ExtractPublicKey(string(keyContent))
	if publicKey == "" {
		return "", &errx.PublicKeyNotFoundError{KeyFile: keyFile, Tried: []string{"CLI parameter", "SOPS_AGE_RECIPIENTS env", ".yewseal.toml config"}}
	}

	if verbose {
		fmt.Printf("✅ Extracted public key from %s: %s\n", keyFile, publicKey)
	}

	return publicKey, nil
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
