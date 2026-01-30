package crypto

import (
	"fmt"
	"os"
	"strings"

	"github.com/yourusername/sops-config-tool/internal/tools"
)

// InitProject initializes the project with Age keys and SOPS configuration
func InitProject(force bool) error {
	// Check if files already exist
	if !force {
		if _, err := os.Stat(".age"); err == nil {
			return fmt.Errorf(".age directory already exists. Use --force to overwrite")
		}
		if _, err := os.Stat(".sops.yaml"); err == nil {
			return fmt.Errorf(".sops.yaml already exists. Use --force to overwrite")
		}
	}

	fmt.Println("🔑 Generating Age key pair...")

	// Create .age directory
	if err := os.MkdirAll(".age", 0700); err != nil {
		return fmt.Errorf("failed to create .age directory: %w", err)
	}

	// Remove existing key file if force mode
	if force {
		os.Remove(".age/keys.txt")
		os.Remove(".age.pub")
	}

	// Generate Age key
	stdout, stderr, err := tools.ExecCommand("age-keygen", "-o", ".age/keys.txt")
	if err != nil {
		return fmt.Errorf("failed to generate Age key: %w\n%s", err, stderr)
	}

	fmt.Println("✅ Age key generated at .age/keys.txt")

	// Extract public key from output (age-keygen prints it to stderr)
	combinedOutput := stderr + stdout
	publicKey := extractPublicKey(combinedOutput)
	if publicKey == "" {
		// If extraction from output failed, try reading from the key file
		keyContent, err := os.ReadFile(".age/keys.txt")
		if err == nil {
			publicKey = extractPublicKey(string(keyContent))
		}
	}
	if publicKey == "" {
		return fmt.Errorf("failed to extract public key from age-keygen output")
	}

	// Save public key
	if err := os.WriteFile(".age.pub", []byte(publicKey+"\n"), 0644); err != nil {
		return fmt.Errorf("failed to save public key: %w", err)
	}

	fmt.Printf("✅ Public key saved to .age.pub: %s\n", publicKey)

	// Create .sops.yaml
	sopsConfig := fmt.Sprintf(`creation_rules:
  - path_regex: .*\.enc\.yaml$
    age: %s
`, publicKey)

	if err := os.WriteFile(".sops.yaml", []byte(sopsConfig), 0644); err != nil {
		return fmt.Errorf("failed to create .sops.yaml: %w", err)
	}

	fmt.Println("✅ Created .sops.yaml")

	// Create .gitignore
	gitignoreContent := `# Decrypted configuration files
wrangler.toml

# Age private keys
.age/keys.txt

# Temporary files
*.tmp
*.bak
*~
`

	if err := os.WriteFile(".gitignore", []byte(gitignoreContent), 0644); err != nil {
		return fmt.Errorf("failed to create .gitignore: %w", err)
	}

	fmt.Println("✅ Created .gitignore")

	// Create example file if wrangler.toml exists
	if _, err := os.Stat("wrangler.toml"); err == nil {
		exampleContent, err := os.ReadFile("wrangler.toml")
		if err == nil {
			if err := os.WriteFile("wrangler.example.toml", exampleContent, 0644); err != nil {
				fmt.Printf("⚠️  Warning: Failed to create wrangler.example.toml: %v\n", err)
			} else {
				fmt.Println("✅ Created wrangler.example.toml (remember to remove sensitive values)")
			}
		}
	}

	fmt.Println("\n🎉 Initialization complete!")
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Review wrangler.example.toml and remove any sensitive values")
	fmt.Println("  2. Run 'sops-config-tool encrypt' to encrypt your configuration")
	fmt.Println("  3. Commit .sops.yaml, .gitignore, wrangler.enc.yaml, and wrangler.example.toml to git")
	fmt.Println("  4. NEVER commit wrangler.toml or .age/keys.txt!")
	fmt.Println("\n⚠️  IMPORTANT: Back up your .age/keys.txt file securely!")

	return nil
}

// extractPublicKey extracts the public key from age-keygen output
func extractPublicKey(output string) string {
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

func splitLines(s string) []string {
	return strings.Split(s, "\n")
}

// GetAgeKey returns the Age private key from environment or file
func GetAgeKey(keyFile string) (string, error) {
	// Priority 1: Environment variable
	if key := os.Getenv("SOPS_AGE_KEY"); key != "" {
		return key, nil
	}

	// Priority 2: Specified key file
	if keyFile != "" {
		return readKeyFile(keyFile)
	}

	// Priority 3: Default key file
	defaultPath := ".age/keys.txt"
	if _, err := os.Stat(defaultPath); err == nil {
		return readKeyFile(defaultPath)
	}

	return "", fmt.Errorf("no Age key found. Set SOPS_AGE_KEY environment variable or provide --key-file")
}

func readKeyFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read key file %s: %w", path, err)
	}

	// Extract the private key line (starts with AGE-SECRET-KEY-)
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 0 && strings.HasPrefix(line, "AGE-SECRET-KEY-") {
			return line, nil
		}
	}

	return "", fmt.Errorf("no valid Age secret key found in %s", path)
}
