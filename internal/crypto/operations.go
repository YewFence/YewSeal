package crypto

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	tools "github.com/YewFence/YewSeal/internal/doctor"
	"github.com/YewFence/YewSeal/internal/errx"
)

// Encrypt encrypts a configuration file using SOPS
// For TOML files: converts to YAML first, then encrypts
// For native SOPS formats (YAML, JSON, ENV, INI): encrypts directly
// formatOverride: optional format string (e.g. "env") to bypass extension detection
func Encrypt(inputFile, outputFile, keyFile, publicKeyParam, formatOverride string, verbose bool) error {
	// Check if input file exists
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		return &errx.NotFoundError{What: "input file", Path: inputFile}
	}

	inputFormat := DetectFormat(inputFile)
	if override := ParseFormat(formatOverride); override != FormatUnknown {
		inputFormat = override
	}

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

	tomlContent, err := os.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	// Step 1: Convert TOML to YAML (via remarshal)
	if verbose {
		fmt.Println("🔄 Converting TOML to YAML...")
	}

	yamlContent, err := TOMLToYAML(tomlContent)
	if err != nil {
		return fmt.Errorf("failed to convert TOML to YAML: %w", err)
	}

	if verbose {
		fmt.Println("🔐 Encrypting with SOPS...")
	}

	// Step 2: Get public key and encrypt
	publicKey, err := GetPublicKey(publicKeyParam, keyFile, verbose)
	if err != nil {
		return err
	}

	encData, err := sopsEncryptData(yamlContent, "yaml", publicKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt: %w", err)
	}

	if err := os.WriteFile(outputFile, encData, 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
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

	publicKey, err := GetPublicKey(publicKeyParam, keyFile, verbose)
	if err != nil {
		return err
	}

	plainData, err := os.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	sopsType := GetSopsType(inputFormat)
	encData, err := sopsEncryptData(plainData, sopsType, publicKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt: %w", err)
	}

	if err := os.WriteFile(outputFile, encData, 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	fmt.Printf("✅ Encrypted %s → %s\n", inputFile, outputFile)
	return nil
}

// Decrypt decrypts a SOPS encrypted file
// For TOML output: decrypts then converts from YAML
// For native SOPS formats: decrypts directly
// formatOverride: optional format string (e.g. "env") to bypass extension detection
func Decrypt(inputFile, outputFile, keyFile, formatOverride string, verbose bool) error {
	// Check if input file exists
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		return &errx.NotFoundError{What: "input file", Path: inputFile}
	}

	outputFormat := DetectFormat(outputFile)
	if override := ParseFormat(formatOverride); override != FormatUnknown {
		outputFormat = override
	}

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

	key, err := GetAgeKey(keyFile)
	if err != nil {
		return err
	}

	encData, err := os.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	yamlContent, err := sopsDecryptData(encData, "yaml", key)
	if err != nil {
		return fmt.Errorf("failed to decrypt: %w", err)
	}

	if verbose {
		fmt.Println("🔄 Converting YAML to TOML...")
	}

	// Convert YAML to TOML (via remarshal)
	tomlContent, err := YAMLToTOML(yamlContent)
	if err != nil {
		return fmt.Errorf("failed to convert YAML to TOML: %w", err)
	}

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

	key, err := GetAgeKey(keyFile)
	if err != nil {
		return err
	}

	encData, err := os.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	sopsType := GetSopsType(outputFormat)
	plainData, err := sopsDecryptData(encData, sopsType, key)
	if err != nil {
		return fmt.Errorf("failed to decrypt: %w", err)
	}

	if err := os.WriteFile(outputFile, plainData, 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	fmt.Printf("✅ Decrypted %s → %s\n", inputFile, outputFile)
	return nil
}

// Edit decrypts the file to a temp file, opens an editor, and re-encrypts if changed.
func Edit(file, editor, keyFile string) error {
	// Check if file exists
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return &errx.NotFoundError{What: "file", Path: file}
	}

	editFormat := DetectFormat(file)
	if editFormat == FormatUnknown {
		editFormat = FormatYAML
	}
	sopsType := GetSopsType(editFormat)

	// Get Age private key
	key, err := GetAgeKey(keyFile)
	if err != nil {
		return err
	}

	// Read and decrypt
	encData, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("failed to read encrypted file: %w", err)
	}

	plainData, err := sopsDecryptData(encData, sopsType, key)
	if err != nil {
		return fmt.Errorf("failed to decrypt: %w", err)
	}

	// Write to temp file
	tmpFile, err := os.CreateTemp("", "yews-edit-*."+string(editFormat))
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.Write(plainData); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Record original hash for change detection
	originalHash := sha256.Sum256(plainData)

	// Resolve editor
	editorCmd := resolveEditor(editor)

	fmt.Printf("✏️  Opening %s in %s...\n", file, editorCmd)

	// Open editor
	parts := strings.Fields(editorCmd)
	args := append(parts[1:], tmpPath)
	cmd := exec.Command(parts[0], args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("editor exited with error: %w", err)
	}

	// Read edited content
	editedData, err := os.ReadFile(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to read edited file: %w", err)
	}

	// Check for changes
	editedHash := sha256.Sum256(editedData)
	if originalHash == editedHash {
		fmt.Println("⏭️  No changes detected, skipping re-encryption")
		return nil
	}

	// Extract public key from original encrypted file
	store := storeForFormat(sopsType)
	tree, err := store.LoadEncryptedFile(encData)
	if err != nil {
		return fmt.Errorf("failed to parse encrypted file metadata: %w", err)
	}

	publicKey, err := extractAgeRecipientFromTree(tree)
	if err != nil {
		return fmt.Errorf("failed to extract public key from encrypted file: %w", err)
	}

	// Re-encrypt with the same public key
	newEncData, err := sopsEncryptData(editedData, sopsType, publicKey)
	if err != nil {
		return fmt.Errorf("failed to re-encrypt: %w", err)
	}

	if err := os.WriteFile(file, newEncData, 0644); err != nil {
		return fmt.Errorf("failed to write encrypted file: %w", err)
	}

	fmt.Println("✅ File edited and re-encrypted successfully")
	return nil
}

// resolveEditor determines the editor command to use.
// Priority: parameter > $EDITOR > $VISUAL > platform default
func resolveEditor(editor string) string {
	if editor != "" {
		return editor
	}
	if e := os.Getenv("EDITOR"); e != "" {
		return e
	}
	if e := os.Getenv("VISUAL"); e != "" {
		return e
	}
	if runtime.GOOS == "windows" {
		// Check if common editors exist
		for _, candidate := range []string{"code -w", "notepad"} {
			name := strings.Fields(candidate)[0]
			if p, _ := exec.LookPath(name); p != "" {
				return candidate
			}
		}
		return "notepad"
	}
	return "vi"
}
