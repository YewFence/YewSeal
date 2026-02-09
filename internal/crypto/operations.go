package crypto

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/YewFence/YewSeal/internal/errx"
	"github.com/YewFence/YewSeal/internal/tools"
)

func hasSopsYaml() bool {
	_, err := os.Stat(".sops.yaml")
	return err == nil
}

func buildSopsEncryptArgs(inputFile, outputFile, inputType, outputType, publicKey string, verbose bool) ([]string, error) {
	if !hasSopsYaml() {
		if strings.TrimSpace(publicKey) == "" {
			return nil, fmt.Errorf("public key is required when .sops.yaml is not present")
		}
		if verbose {
			fmt.Println("📋 No .sops.yaml found, using command-line parameters only")
		}
		return []string{"--config", os.DevNull, "encrypt", "--age", publicKey, "--input-type", inputType, "--output-type", outputType, inputFile, "--output", outputFile}, nil
	}

	if verbose {
		fmt.Println("📋 Using .sops.yaml configuration")
	}
	return []string{"encrypt", "--filename-override", outputFile, "--input-type", inputType, "--output-type", outputType, inputFile, "--output", outputFile}, nil
}

// Encrypt encrypts a configuration file using SOPS
// For TOML files: converts to YAML first, then encrypts
// For native SOPS formats (YAML, JSON, ENV, INI): encrypts directly
func Encrypt(inputFile, outputFile, keyFile, publicKeyParam string, verbose bool) error {
	// Check if input file exists
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		return &errx.NotFoundError{What: "input file", Path: inputFile}
	}

	inputFormat := DetectFormat(inputFile)

	if inputFormat == FormatTOML {
		// TOML needs conversion: check remarshal is available
		if err := tools.CheckRemarshal(); err != nil {
			return err
		}
		return encryptTOML(inputFile, outputFile, keyFile, publicKeyParam, verbose)
	}

	if inputFormat == FormatUnknown {
		return &errx.UnsupportedFormatError{Path: inputFile, Supported: []string{".toml", ".yaml", ".yml", ".json", ".env", ".ini"}}
	}

	// Native SOPS format: encrypt directly
	return encryptNative(inputFile, outputFile, keyFile, publicKeyParam, inputFormat, verbose)
}

// encryptTOML encrypts a TOML file by converting it to YAML first
func encryptTOML(inputFile, outputFile, keyFile, publicKeyParam string, verbose bool) error {
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

	if err := os.WriteFile(tempFile, yamlContent, 0600); err != nil {
		return fmt.Errorf("failed to write temporary YAML file: %w", err)
	}
	defer os.Remove(tempFile)

	if verbose {
		fmt.Println("🔐 Encrypting with SOPS...")
	}

	// Step 2: Encrypt with SOPS
	// If .sops.yaml exists, rely on creation_rules; otherwise fall back to command-line recipients.
	publicKey := ""
	if !hasSopsYaml() {
		// Get Age public key with priority: CLI param > env var > config file > extract from keys.txt
		publicKey, err = GetPublicKey(publicKeyParam, keyFile, verbose)
		if err != nil {
			return err
		}
	}

	args, err := buildSopsEncryptArgs(tempFile, outputFile, "yaml", "yaml", publicKey, verbose)
	if err != nil {
		return err
	}

	_, stderr, err := tools.ExecCommand("sops", args...)
	if err != nil {
		return &errx.ExternalCommandError{Op: "failed to encrypt with SOPS", Cmd: "sops", Args: args, Stderr: stderr, Err: err}
	}

	fmt.Printf("✅ Encrypted %s → %s\n", inputFile, outputFile)
	return nil
}

// encryptNative encrypts a native SOPS format file directly
func encryptNative(inputFile, outputFile, keyFile, publicKeyParam string, inputFormat FileFormat, verbose bool) error {
	if verbose {
		fmt.Printf("📖 Reading %s...\n", inputFile)
		fmt.Println("🔐 Encrypting with SOPS...")
	}

	sopsType := GetSopsType(inputFormat)

	publicKey := ""
	var err error
	if !hasSopsYaml() {
		// Get Age public key (only needed when we don't have .sops.yaml)
		publicKey, err = GetPublicKey(publicKeyParam, keyFile, verbose)
		if err != nil {
			return err
		}
	}

	args, err := buildSopsEncryptArgs(inputFile, outputFile, sopsType, sopsType, publicKey, verbose)
	if err != nil {
		return err
	}

	_, stderr, err := tools.ExecCommand("sops", args...)
	if err != nil {
		return &errx.ExternalCommandError{Op: "failed to encrypt with SOPS", Cmd: "sops", Args: args, Stderr: stderr, Err: err}
	}

	fmt.Printf("✅ Encrypted %s → %s\n", inputFile, outputFile)
	return nil
}

// Decrypt decrypts a SOPS encrypted file
// For TOML output: decrypts then converts from YAML
// For native SOPS formats: decrypts directly
func Decrypt(inputFile, outputFile, keyFile string, verbose bool) error {
	// Check if input file exists
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		return &errx.NotFoundError{What: "input file", Path: inputFile}
	}

	outputFormat := DetectFormat(outputFile)

	if outputFormat == FormatTOML {
		// TOML needs conversion: check remarshal is available
		if err := tools.CheckRemarshal(); err != nil {
			return err
		}
		return decryptTOML(inputFile, outputFile, keyFile, verbose)
	}

	if outputFormat == FormatUnknown {
		return &errx.UnsupportedFormatError{Path: outputFile, Supported: []string{".toml", ".yaml", ".yml", ".json", ".env", ".ini"}}
	}

	// Native SOPS format: decrypt directly
	return decryptNative(inputFile, outputFile, keyFile, outputFormat, verbose)
}

// decryptTOML decrypts a SOPS file and converts it to TOML
func decryptTOML(inputFile, outputFile, keyFile string, verbose bool) error {
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
		return &errx.ExternalCommandError{Op: "failed to decrypt with SOPS", Cmd: "sops", Args: []string{"--decrypt", "--output", tempFile, inputFile}, Stderr: stderr, Err: err}
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

// decryptNative decrypts a native SOPS format file directly
func decryptNative(inputFile, outputFile, keyFile string, outputFormat FileFormat, verbose bool) error {
	if verbose {
		fmt.Printf("📖 Reading %s...\n", inputFile)
		fmt.Println("🔓 Decrypting with SOPS...")
	}

	// Get Age key
	key, err := GetAgeKey(keyFile)
	if err != nil {
		return err
	}

	env := map[string]string{
		"SOPS_AGE_KEY": key,
	}

	// Decrypt directly to output file
	_, stderr, err := tools.ExecCommandWithEnv(env, "sops", "--decrypt", "--output", outputFile, inputFile)
	if err != nil {
		return &errx.ExternalCommandError{Op: "failed to decrypt with SOPS", Cmd: "sops", Args: []string{"--decrypt", "--output", outputFile, inputFile}, Stderr: stderr, Err: err}
	}

	fmt.Printf("✅ Decrypted %s → %s\n", inputFile, outputFile)
	return nil
}

// Edit opens the encrypted file in an editor using SOPS
func Edit(file, editor, keyFile string) error {
	// Check if file exists
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return &errx.NotFoundError{What: "file", Path: file}
	}

	fmt.Printf("✏️  Opening %s in editor...\n", file)

	// Get Age key
	key, err := GetAgeKey(keyFile)
	if err != nil {
		return err
	}

	// Prepare environment for subprocess only
	env := map[string]string{
		"SOPS_AGE_KEY": key,
	}
	if editor != "" {
		env["EDITOR"] = editor
	}

	// Build command
	args := []string{file}

	// Run SOPS interactively
	if err := tools.ExecCommandInteractiveWithEnv(env, "sops", args...); err != nil {
		return fmt.Errorf("failed to edit file: %w", err)
	}

	fmt.Println("✅ File edited successfully")
	return nil
}
