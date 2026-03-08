package crypto

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"filippo.io/age"
	"github.com/YewFence/YewSeal/internal/tools"
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

	// Generate or use existing Age key
	publicKey, err := setupAgeKey(force)
	if err != nil {
		return err
	}

	// Create or update .sops.yaml if requested
	if shouldCreateSopsConfig {
		if err := UpdateSopsYaml(outputFile, publicKey, force); err != nil {
			return fmt.Errorf("failed to update .sops.yaml: %w", err)
		}
	} else {
		fmt.Println("⏭️  Skipped creating .sops.yaml")
	}

	// Create or update .yewseal.toml with public key
	if err := SavePublicKeyToConfig(publicKey, inputFile, outputFile); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	// Create or update .gitignore
	if err := updateGitignore(inputFile); err != nil {
		return err
	}

	// Create example file if requested and input file exists
	if shouldCreateExample {
		createExampleFile(inputFile)
	}

	// Print completion message
	printCompletionMessage(inputBase, inputExt, inputFile, outputFile, shouldCreateExample, shouldCreateSopsConfig)

	return nil
}

// setupAgeKey generates or retrieves the Age key pair
func setupAgeKey(force bool) (string, error) {
	keyFilePath := ".age/keys.txt"
	keyExists := false

	if _, err := os.Stat(keyFilePath); err == nil {
		keyExists = true
	}

	if keyExists && !force {
		// Use existing key
		fmt.Println("🔑 Found existing Age key, using it...")

		keyContent, err := os.ReadFile(keyFilePath)
		if err != nil {
			return "", fmt.Errorf("failed to read existing key file: %w", err)
		}
		publicKey := ExtractPublicKey(string(keyContent))
		if publicKey == "" {
			return "", fmt.Errorf("failed to extract public key from existing key file")
		}
		fmt.Printf("✅ Using existing public key: %s\n", publicKey)
		return publicKey, nil
	}

	// Generate new key (either no key exists or force mode)
	if force && keyExists {
		fmt.Println("🔑 Force mode: Regenerating Age key pair...")
		os.Remove(keyFilePath)
	} else {
		fmt.Println("🔑 Generating Age key pair...")
	}

	// Create .age directory
	if err := os.MkdirAll(".age", 0700); err != nil {
		return "", fmt.Errorf("failed to create .age directory: %w", err)
	}

	// Generate Age key using filippo.io/age library
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		return "", fmt.Errorf("failed to generate Age key: %w", err)
	}

	keyContent := fmt.Sprintf("# created: %s\n# public key: %s\n%s\n",
		time.Now().UTC().Format(time.RFC3339),
		identity.Recipient().String(),
		identity.String())

	if err := os.WriteFile(keyFilePath, []byte(keyContent), 0600); err != nil {
		return "", fmt.Errorf("failed to write key file: %w", err)
	}

	publicKey := identity.Recipient().String()

	fmt.Println("✅ Age key generated at .age/keys.txt")
	fmt.Printf("✅ Public key generated: %s\n", publicKey)
	return publicKey, nil
}

// updateGitignore creates or updates .gitignore with YewSeal entries
func updateGitignore(inputFile string) error {
	gitignoreAdditions := fmt.Sprintf(`
# YewSeal - Decrypted configuration files
%s

# YewSeal - Age private keys
.age/keys.txt
`, inputFile)

	// Read existing .gitignore if it exists
	if existingData, err := os.ReadFile(".gitignore"); err == nil {
		// Check if YewSeal section already exists
		if !strings.Contains(string(existingData), "# YewSeal") {
			// Append to existing content
			gitignoreContent := string(existingData) + gitignoreAdditions
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

	return nil
}

// createExampleFile creates an example file from the input file
func createExampleFile(inputFile string) {
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

// printCompletionMessage prints the initialization completion message
func printCompletionMessage(inputBase, inputExt, inputFile, outputFile string, shouldCreateExample, shouldCreateSopsConfig bool) {
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
}
