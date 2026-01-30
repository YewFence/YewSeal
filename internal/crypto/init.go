package crypto

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/YewFence/YewSeal/internal/tools"
	"gopkg.in/yaml.v3"
)

// InitProject initializes the project with Age keys and SOPS configuration
func InitProject(force bool, inputFile, outputFile string, createExampleFlag, skipSopsConfigFlag bool) error {

	// Interactive mode: Determine file names
	// If parameters are empty, use interactive prompts
	if inputFile == "" {
		inputFile = tools.PromptWithDefault("Enter original config file name", "wrangler.toml")
	}

	// Extract extension from input file
	inputExt := filepath.Ext(inputFile)
	inputBase := strings.TrimSuffix(filepath.Base(inputFile), inputExt)

	// Determine output file
	if outputFile == "" {
		// Generate default output file name: {base}.enc.{ext}.yaml
		// For example: wrangler.toml -> wrangler.enc.toml.yaml
		defaultOutput := inputBase + ".enc" + inputExt + ".yaml"
		outputFile = tools.PromptWithDefault("Enter encrypted output file name", defaultOutput)
	}

	// Interactive mode: Ask about creating example file
	shouldCreateExample := tools.PromptYesNoConditional(
		createExampleFlag,
		createExampleFlag,
		"Create example file? (Recommended, as encryption loses comments)",
	)

	// Interactive mode: Ask about creating .sops.yaml
	shouldCreateSopsConfig := tools.PromptYesNoConditional(
		skipSopsConfigFlag,
		!skipSopsConfigFlag,
		"Create .sops.yaml? (Optional, but convenient for direct sops commands)",
	)

	var publicKey string

	// Check if Age key already exists
	keyFilePath := ".age/keys.txt"
	pubKeyPath := ".age.pub"
	keyExists := false

	if _, err := os.Stat(keyFilePath); err == nil {
		keyExists = true
	}

	if keyExists && !force {
		// Use existing key
		fmt.Println("🔑 Found existing Age key, using it...")

		// Try to read public key from .age.pub file
		if pubContent, err := os.ReadFile(pubKeyPath); err == nil {
			publicKey = strings.TrimSpace(string(pubContent))
			fmt.Printf("✅ Using existing public key: %s\n", publicKey)
		} else {
			// If .age.pub doesn't exist, try to extract from keys.txt
			keyContent, err := os.ReadFile(keyFilePath)
			if err != nil {
				return fmt.Errorf("failed to read existing key file: %w", err)
			}
			publicKey = extractPublicKey(string(keyContent))
			if publicKey == "" {
				return fmt.Errorf("failed to extract public key from existing key file")
			}
			// Save the public key for future use
			if err := os.WriteFile(pubKeyPath, []byte(publicKey+"\n"), 0644); err != nil {
				return fmt.Errorf("failed to save public key: %w", err)
			}
			fmt.Printf("✅ Extracted and saved public key: %s\n", publicKey)
		}
	} else {
		// Generate new key (either no key exists or force mode)
		if force && keyExists {
			fmt.Println("🔑 Force mode: Regenerating Age key pair...")
			os.Remove(keyFilePath)
			os.Remove(pubKeyPath)
		} else {
			fmt.Println("🔑 Generating Age key pair...")
		}

		// Create .age directory
		if err := os.MkdirAll(".age", 0700); err != nil {
			return fmt.Errorf("failed to create .age directory: %w", err)
		}

		// Generate Age key
		stdout, stderr, err := tools.ExecCommand("age-keygen", "-o", keyFilePath)
		if err != nil {
			return fmt.Errorf("failed to generate Age key: %w\n%s", err, stderr)
		}

		fmt.Println("✅ Age key generated at .age/keys.txt")

		// Extract public key from output (age-keygen prints it to stderr)
		combinedOutput := stderr + stdout
		publicKey = extractPublicKey(combinedOutput)
		if publicKey == "" {
			// If extraction from output failed, try reading from the key file
			keyContent, err := os.ReadFile(keyFilePath)
			if err == nil {
				publicKey = extractPublicKey(string(keyContent))
			}
		}
		if publicKey == "" {
			return fmt.Errorf("failed to extract public key from age-keygen output")
		}

		// Save public key
		if err := os.WriteFile(pubKeyPath, []byte(publicKey+"\n"), 0644); err != nil {
			return fmt.Errorf("failed to save public key: %w", err)
		}

		fmt.Printf("✅ Public key saved to .age.pub: %s\n", publicKey)
	}

	// Create or update .sops.yaml if requested
	if shouldCreateSopsConfig {
		if err := updateSopsYaml(outputFile, publicKey, force); err != nil {
			return fmt.Errorf("failed to update .sops.yaml: %w", err)
		}
	} else {
		fmt.Println("⏭️  Skipped creating .sops.yaml")
	}

	// Create or update .gitignore
	gitignoreAdditions := fmt.Sprintf(`
# YewSeal - Decrypted configuration files
%s

# YewSeal - Age private keys
.age/keys.txt
`, inputFile)

	// Read existing .gitignore if it exists
	var existingContent []byte
	if existingData, err := os.ReadFile(".gitignore"); err == nil {
		existingContent = existingData
		// Check if YewSeal section already exists
		if !strings.Contains(string(existingContent), "# YewSeal") {
			// Append to existing content
			gitignoreContent := string(existingContent) + gitignoreAdditions
			if err := os.WriteFile(".gitignore", []byte(gitignoreContent), 0644); err != nil {
				return fmt.Errorf("failed to update .gitignore: %w", err)
			}
			fmt.Println("✅ Updated .gitignore")
		} else {
			fmt.Println("⏭️  .gitignore already contains YewSeal entries")
		}
	} else {
		// Create new .gitignore
		if err := os.WriteFile(".gitignore", []byte(strings.TrimPrefix(gitignoreAdditions, "\n")), 0644); err != nil {
			return fmt.Errorf("failed to create .gitignore: %w", err)
		}
		fmt.Println("✅ Created .gitignore")
	}

	// Create example file if requested and input file exists
	if shouldCreateExample {
		if _, err := os.Stat(inputFile); err == nil {
			exampleContent, err := os.ReadFile(inputFile)
			if err == nil {
				exampleFile := strings.TrimSuffix(inputFile, filepath.Ext(inputFile)) + ".example" + filepath.Ext(inputFile)
				if err := os.WriteFile(exampleFile, exampleContent, 0644); err != nil {
					fmt.Printf("⚠️  Warning: Failed to create %s: %v\n", exampleFile, err)
				} else {
					fmt.Printf("✅ Created %s (remember to remove sensitive values)\n", exampleFile)
				}
			}
		} else {
			fmt.Printf("⚠️  Warning: Input file %s does not exist yet, skipping example creation\n", inputFile)
		}
	}

	fmt.Println("\n🎉 Initialization complete!")
	fmt.Println("\nNext steps:")
	if shouldCreateExample {
		fmt.Printf("  1. Review %s.example%s and remove any sensitive values\n", inputBase, inputExt)
	}
	fmt.Printf("  2. Run 'yews encrypt -i %s -o %s' to encrypt your configuration\n", inputFile, outputFile)
	if shouldCreateSopsConfig {
		fmt.Printf("  3. Commit .sops.yaml, .gitignore, %s", outputFile)
		if shouldCreateExample {
			fmt.Printf(", and %s.example%s", inputBase, inputExt)
		}
		fmt.Println(" to git")
	} else {
		fmt.Printf("  3. Commit .gitignore, %s", outputFile)
		if shouldCreateExample {
			fmt.Printf(", and %s.example%s", inputBase, inputExt)
		}
		fmt.Println(" to git")
	}
	fmt.Printf("  4. NEVER commit %s or .age/keys.txt!\n", inputFile)
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

// SopsConfig represents the structure of .sops.yaml
type SopsConfig struct {
	CreationRules []CreationRule `yaml:"creation_rules"`
}

// CreationRule represents a single rule in .sops.yaml
type CreationRule struct {
	PathRegex string `yaml:"path_regex"`
	Age       string `yaml:"age"`
}

// updateSopsYaml creates or updates .sops.yaml with proper YAML handling
func updateSopsYaml(outputFile, publicKey string, force bool) error {
	const sopsYamlPath = ".sops.yaml"

	// Generate regex pattern that precisely matches the output file
	escapedOutput := regexp.QuoteMeta(outputFile)
	pathRegex := "^" + escapedOutput + "$"

	var config SopsConfig

	// Try to read existing file
	if existingData, err := os.ReadFile(sopsYamlPath); err == nil {
		// Parse existing YAML
		if err := yaml.Unmarshal(existingData, &config); err != nil {
			return fmt.Errorf("failed to parse existing .sops.yaml: %w", err)
		}

		// Check if rule already exists (idempotency)
		for _, rule := range config.CreationRules {
			if rule.PathRegex == pathRegex {
				fmt.Printf("⏭️  .sops.yaml already contains rule for %s\n", outputFile)
				return nil
			}
		}

		if force {
			// Force mode: replace all rules
			config.CreationRules = []CreationRule{
				{PathRegex: pathRegex, Age: publicKey},
			}
			fmt.Println("✅ Replaced .sops.yaml (force mode)")
		} else {
			// Append new rule
			config.CreationRules = append(config.CreationRules, CreationRule{
				PathRegex: pathRegex,
				Age:       publicKey,
			})
			fmt.Printf("✅ Added rule to .sops.yaml for %s\n", outputFile)
		}
	} else {
		// File doesn't exist, create new
		config.CreationRules = []CreationRule{
			{PathRegex: pathRegex, Age: publicKey},
		}
		fmt.Println("✅ Created .sops.yaml")
	}

	// Marshal to YAML
	data, err := yaml.Marshal(&config)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	// Write to file
	if err := os.WriteFile(sopsYamlPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write .sops.yaml: %w", err)
	}

	return nil
}
