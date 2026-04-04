package project

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"filippo.io/age"
	"github.com/YewFence/YewSeal/internal/config"
	tools "github.com/YewFence/YewSeal/internal/prompt"
)

// InitProject initializes the project with Age keys and SOPS configuration.
func InitProject(force bool, inputFile, outputFile string, createExampleFlag, skipSopsConfigFlag bool) error {
	interactive := inputFile == "" && outputFile == ""

	shouldContinue, err := confirmInitOverwrite(force, interactive)
	if err != nil {
		return err
	}
	if !shouldContinue {
		fmt.Println("⏭️  Skipped init because existing config was kept")
		return nil
	}

	filePairs := collectInitFilePairs(inputFile, outputFile)

	shouldCreateExample := tools.PromptYesNoConditional(
		createExampleFlag,
		createExampleFlag,
		"Create example files? (recommended if comments matter)",
	)

	shouldCreateSopsConfig := tools.PromptYesNoConditional(
		skipSopsConfigFlag,
		!skipSopsConfigFlag,
		"Create .sops.yaml? (optional, but convenient for direct sops commands)",
	)

	publicKey, err := setupAgeKey(force)
	if err != nil {
		return err
	}

	if shouldCreateSopsConfig {
		if err := SyncSopsYaml(filePairs, publicKey); err != nil {
			return fmt.Errorf("failed to update .sops.yaml: %w", err)
		}
	} else {
		fmt.Println("⏭️  Skipped creating .sops.yaml")
	}

	if err := SavePublicKeyToConfig(publicKey, filePairs); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	if err := UpdateGitignore(filePairs); err != nil {
		return err
	}

	if shouldCreateExample {
		for _, filePair := range filePairs {
			createExampleFile(filePair.PlaintextPath)
		}
	}

	printCompletionMessage(filePairs, shouldCreateExample, shouldCreateSopsConfig)
	return nil
}

func confirmInitOverwrite(force, interactive bool) (bool, error) {
	if force {
		return true, nil
	}

	if _, err := os.Stat(".yewseal.toml"); err != nil {
		return true, nil
	}

	if !interactive {
		return false, fmt.Errorf(".yewseal.toml already exists, use --force to overwrite")
	}

	return tools.PromptYesNo(".yewseal.toml already exists, overwrite it?", false), nil
}

func collectInitFilePairs(inputFile, outputFile string) []config.FilePair {
	if inputFile != "" || outputFile != "" {
		filePair := config.DefaultFilePair()
		if inputFile != "" {
			filePair.PlaintextPath = inputFile
		}
		if outputFile != "" {
			filePair.EncryptedPath = outputFile
		} else {
			filePair.EncryptedPath = defaultEncryptedOutputNameForFile(filePair.PlaintextPath)
		}
		return []config.FilePair{filePair}
	}

	fmt.Println("ℹ️  Init 会把所有文件统一写进 [[encryption.files]]。")
	fmt.Println("ℹ️  先录入第一组文件，后面可以继续追加。")

	filePairs := []config.FilePair{promptInitFilePair(true)}
	for tools.PromptYesNo("Add another file to encrypt?", false) {
		filePairs = append(filePairs, promptInitFilePair(false))
	}

	return filePairs
}

func promptInitFilePair(first bool) config.FilePair {
	var plaintextFile string
	if first {
		plaintextFile = tools.PromptWithDefault("Enter plaintext config file name", config.DefaultFilePair().PlaintextPath)
	} else {
		plaintextFile = tools.PromptRequired("Enter plaintext config file name")
	}

	encryptedFile := tools.PromptWithDefault("Enter encrypted file name", defaultEncryptedOutputNameForFile(plaintextFile))
	return config.FilePair{
		PlaintextPath: plaintextFile,
		EncryptedPath: encryptedFile,
	}
}

func defaultEncryptedOutputNameForFile(inputFile string) string {
	inputExt := filepath.Ext(inputFile)
	inputBase := strings.TrimSuffix(filepath.Base(inputFile), inputExt)
	return defaultEncryptedOutputName(inputBase, inputExt)
}

func defaultEncryptedOutputName(inputBase, inputExt string) string {
	return inputBase + ".enc" + inputExt + ".yaml"
}

func extractPublicKey(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# public key: ") {
			return strings.TrimPrefix(line, "# public key: ")
		}
	}
	return ""
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
		publicKey := extractPublicKey(string(keyContent))
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
func printCompletionMessage(filePairs []config.FilePair, shouldCreateExample, shouldCreateSopsConfig bool) {
	fmt.Println("\n🎉 Initialization complete!")
	fmt.Println("\nNext steps:")
	step := 1
	if shouldCreateExample {
		fmt.Printf("  %d. Review the generated .example files and remove any sensitive values\n", step)
		step++
	}
	fmt.Printf("  %d. Run 'yews encrypt' to encrypt the %d configured file(s)\n", step, len(filePairs))
	step++
	fmt.Printf("  %d. Run 'yews decrypt' whenever you need the plaintext back\n", step)
	step++
	fmt.Printf("  %d. After encrypting, commit .yewseal.toml, .gitignore", step)
	if shouldCreateSopsConfig {
		fmt.Print(", .sops.yaml")
	}
	if len(filePairs) == 1 {
		fmt.Printf(", and %s", filePairs[0].EncryptedPath)
	} else {
		fmt.Print(", and the encrypted files")
	}
	fmt.Println(" to git")
	step++
	fmt.Printf("  %d. NEVER commit the plaintext files listed in .gitignore or .age/keys.txt!\n", step)
	fmt.Println("\n⚠️  IMPORTANT: Back up your .age/keys.txt file securely!")
}
