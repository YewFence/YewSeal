package crypto

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/YewFence/YewSeal/internal/tools"
)

// Encrypt converts TOML to YAML and encrypts it with SOPS
func Encrypt(inputFile, outputFile, keyFile string, verbose bool) error {
	// Check if input file exists
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		return fmt.Errorf("input file %s does not exist", inputFile)
	}

	if verbose {
		fmt.Printf("📖 Reading %s...\n", inputFile)
	}

	// Read input TOML file
	tomlContent, err := os.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	// Step 1: Convert TOML to YAML
	if verbose {
		fmt.Println("🔄 Converting TOML to YAML...")
	}

	yamlContent, err := TOMLToYAML(tomlContent)
	if err != nil {
		return fmt.Errorf("failed to convert TOML to YAML: %w", err)
	}

	// Write to temporary file (must match .sops.yaml pattern for encryption to work)
	// Generate temp file name based on output file to maintain the infix pattern
	outputExt := filepath.Ext(outputFile)
	outputBase := strings.TrimSuffix(filepath.Base(outputFile), outputExt)
	tempFile := "." + outputBase + ".tmp" + outputExt

	if err := os.WriteFile(tempFile, yamlContent, 0644); err != nil {
		return fmt.Errorf("failed to write temporary YAML file: %w", err)
	}
	defer os.Remove(tempFile)

	if verbose {
		fmt.Println("🔐 Encrypting with SOPS...")
	}

	// Step 2: Encrypt with SOPS
	// Read Age public key from .age.pub file
	pubKeyPath := ".age.pub"
	pubKeyContent, err := os.ReadFile(pubKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read Age public key from %s: %w", pubKeyPath, err)
	}
	publicKey := strings.TrimSpace(string(pubKeyContent))

	// Use --age parameter to specify the public key for encryption
	_, stderr, err := tools.ExecCommand("sops", "--encrypt", "--age", publicKey, "--input-type", "yaml", "--output-type", "yaml", "--output", outputFile, tempFile)
	if err != nil {
		return fmt.Errorf("failed to encrypt with SOPS: %w\n%s", err, stderr)
	}

	fmt.Printf("✅ Encrypted %s → %s\n", inputFile, outputFile)
	return nil
}

// Decrypt decrypts SOPS file and converts it back to TOML
func Decrypt(inputFile, outputFile, keyFile string, verbose bool) error {
	// Check if input file exists
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		return fmt.Errorf("input file %s does not exist", inputFile)
	}

	if verbose {
		fmt.Printf("📖 Reading %s...\n", inputFile)
		fmt.Println("🔓 Decrypting with SOPS...")
	}

	// Step 1: Decrypt with SOPS
	key, err := GetAgeKey(keyFile)
	if err != nil {
		return err
	}

	env := map[string]string{
		"SOPS_AGE_KEY": key,
	}

	// Create temporary file for decrypted YAML
	tempFile := outputFile + ".tmp.yaml"
	_, stderr, err := tools.ExecCommandWithEnv(env, "sops", "--decrypt", "--output", tempFile, inputFile)
	if err != nil {
		return fmt.Errorf("failed to decrypt with SOPS: %w\n%s", err, stderr)
	}
	defer os.Remove(tempFile)

	if verbose {
		fmt.Println("🔄 Converting YAML to TOML...")
	}

	// Step 2: Convert YAML to TOML using Go libraries
	yamlContent, err := os.ReadFile(tempFile)
	if err != nil {
		return fmt.Errorf("failed to read decrypted YAML: %w", err)
	}

	tomlContent, err := YAMLToTOML(yamlContent)
	if err != nil {
		return fmt.Errorf("failed to convert YAML to TOML: %w", err)
	}

	// Write output
	if err := os.WriteFile(outputFile, tomlContent, 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	fmt.Printf("✅ Decrypted %s → %s\n", inputFile, outputFile)
	return nil
}

// Edit opens the encrypted file in an editor using SOPS
func Edit(file, editor, keyFile string) error {
	// Check if file exists
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return fmt.Errorf("file %s does not exist", file)
	}

	fmt.Printf("✏️  Opening %s in editor...\n", file)

	// Get Age key
	key, err := GetAgeKey(keyFile)
	if err != nil {
		return err
	}

	// Prepare environment - set SOPS_AGE_KEY
	os.Setenv("SOPS_AGE_KEY", key)
	defer os.Unsetenv("SOPS_AGE_KEY")

	// Set EDITOR environment variable if provided
	if editor != "" {
		os.Setenv("EDITOR", editor)
		defer os.Unsetenv("EDITOR")
	}

	// Build command
	args := []string{file}

	// Run SOPS interactively
	if err := tools.ExecCommandInteractive("sops", args...); err != nil {
		return fmt.Errorf("failed to edit file: %w", err)
	}

	fmt.Println("✅ File edited successfully")
	return nil
}
