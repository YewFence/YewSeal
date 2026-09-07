package project

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"filippo.io/age"
	"github.com/YewFence/YewSeal/internal/agekey"
	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/presentation"
	tools "github.com/YewFence/YewSeal/internal/prompt"
)

const defaultInitPlaintextFile = "wrangler.toml"

type initSelections struct {
	FilePairs    []config.FilePair
	ExampleFiles []string
}

type initializer struct {
	output  *presentation.Output
	prompts *tools.Session
}

// InitProject initializes the project with Age keys and SOPS configuration.
func InitProject(force bool, inputFile, outputFile, formatOverride string, createExampleFlag, skipSopsConfigFlag bool, out *presentation.Output, prompts *tools.Session) (err error) {
	i := initializer{output: presentation.OrDiscard(out), prompts: prompts}
	defer func() { err = i.output.Finish(i.prompts.Check(err)) }()
	interactive := inputFile == "" && outputFile == ""

	shouldContinue, err := i.confirmInitOverwrite(force, interactive)
	if err != nil {
		return err
	}
	if !shouldContinue {
		i.output.InitKept()
		return nil
	}

	selections, err := i.collectInitSelections(inputFile, outputFile, formatOverride, createExampleFlag)
	if err != nil {
		return err
	}
	filePairs := selections.FilePairs

	shouldCreateSopsConfig := i.prompts.PromptYesNoConditional(
		!interactive || skipSopsConfigFlag,
		!skipSopsConfigFlag,
		"Create .sops.yaml? (optional, but convenient for direct sops commands)",
	)
	if err := i.prompts.Err(); err != nil {
		return err
	}

	if force {
		i.output.Warning("Force rebuild: the new owner identity may not decrypt existing ciphertext")
	}

	publicKey, err := setupAgeKey(force)
	if err != nil {
		return err
	}

	resolved := make([]config.ResolvedFilePair, 0, len(filePairs))
	for _, filePair := range filePairs {
		resolved = append(resolved, config.ResolvedFilePair{PlaintextPath: filePair.PlaintextPath, EncryptedPath: filePair.EncryptedPath, Format: filePair.Format, Recipients: []string{publicKey}})
	}
	if shouldCreateSopsConfig {
		if err := SyncResolvedSopsYaml(resolved); err != nil {
			return fmt.Errorf("failed to update .sops.yaml: %w", err)
		}
	} else {
		if force {
			if err := os.Remove(sopsYamlPath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to remove managed .sops.yaml: %w", err)
			}
		}
	}

	for i := range filePairs {
		aliases := []string{"owner"}
		filePairs[i].Recipients = &aliases
	}
	if err := SaveBootstrapConfig(publicKey, filePairs); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	if err := UpdateGitignore(filePairs); err != nil {
		return err
	}

	for _, exampleFile := range selections.ExampleFiles {
		i.createExampleFile(exampleFile)
	}
	i.output.Initialized(len(filePairs), shouldCreateSopsConfig)
	return nil
}

func (i *initializer) confirmInitOverwrite(force, interactive bool) (bool, error) {
	if force {
		return true, nil
	}

	if _, err := os.Stat(".yewseal.toml"); err != nil {
		return true, nil
	}

	if !interactive {
		return false, fmt.Errorf(".yewseal.toml already exists, use --force to overwrite")
	}

	answer := i.prompts.PromptYesNo(".yewseal.toml already exists, overwrite it?", false)
	return answer, i.prompts.Err()
}

func (i *initializer) collectInitFilePairs(inputFile, outputFile, formatOverride string) ([]config.FilePair, error) {
	if inputFile != "" || outputFile != "" {
		filePair, err := i.newInitFilePair(inputFile, outputFile, formatOverride, false)
		if err != nil {
			return nil, err
		}
		return []config.FilePair{filePair}, nil
	}

	firstFilePair, err := i.promptInitFilePair(true)
	if err != nil {
		return nil, err
	}
	filePairs := []config.FilePair{firstFilePair}
	for i.prompts.PromptYesNo("Add another file to encrypt?", false) {
		filePair, err := i.promptInitFilePair(false)
		if err != nil {
			return nil, err
		}
		filePairs = append(filePairs, filePair)
	}

	return filePairs, i.prompts.Err()
}

func (i *initializer) collectInitSelections(inputFile, outputFile, formatOverride string, createExampleFlag bool) (initSelections, error) {
	if inputFile != "" || outputFile != "" {
		filePairs, err := i.collectInitFilePairs(inputFile, outputFile, formatOverride)
		if err != nil {
			return initSelections{}, err
		}
		selections := initSelections{FilePairs: filePairs}
		if createExampleFlag {
			selections.ExampleFiles = append(selections.ExampleFiles, filePairs[0].PlaintextPath)
		}
		return selections, nil
	}

	selections := initSelections{}
	filePair, shouldCreateExample, err := i.promptInteractiveInitFilePair(true, createExampleFlag)
	if err != nil {
		return initSelections{}, err
	}
	selections.FilePairs = append(selections.FilePairs, filePair)
	if shouldCreateExample {
		selections.ExampleFiles = append(selections.ExampleFiles, filePair.PlaintextPath)
	}

	for i.prompts.PromptYesNo("Add another file to encrypt?", false) {
		filePair, shouldCreateExample, err = i.promptInteractiveInitFilePair(false, createExampleFlag)
		if err != nil {
			return initSelections{}, err
		}
		selections.FilePairs = append(selections.FilePairs, filePair)
		if shouldCreateExample {
			selections.ExampleFiles = append(selections.ExampleFiles, filePair.PlaintextPath)
		}
	}

	return selections, i.prompts.Err()
}

func (i *initializer) promptInitFilePair(first bool) (config.FilePair, error) {
	var plaintextFile string
	if first {
		plaintextFile = i.prompts.PromptWithDefault("Enter plaintext config file name", defaultInitPlaintextFile)
	} else {
		var err error
		plaintextFile, err = i.prompts.PromptRequired("Enter plaintext config file name")
		if err != nil {
			return config.FilePair{}, fmt.Errorf("failed to read plaintext config file name: %w", err)
		}
	}

	encryptedFile := i.prompts.PromptWithDefault("Enter encrypted file name", defaultEncryptedOutputNameForFile(plaintextFile))
	if err := i.prompts.Err(); err != nil {
		return config.FilePair{}, err
	}
	formatOverride, err := i.resolveInitFormatOverride(plaintextFile, "", true)
	if err != nil {
		return config.FilePair{}, err
	}

	return config.FilePair{
		PlaintextPath: plaintextFile,
		EncryptedPath: encryptedFile,
		Format:        formatOverride,
	}, nil
}

func (i *initializer) promptInteractiveInitFilePair(first bool, createExampleFlag bool) (config.FilePair, bool, error) {
	filePair, err := i.promptInitFilePair(first)
	if err != nil {
		return config.FilePair{}, false, err
	}
	if createExampleFlag {
		return filePair, true, nil
	}

	shouldCreateExample := i.prompts.PromptYesNo(
		fmt.Sprintf("Create example file for %s?", filePair.PlaintextPath),
		false,
	)
	return filePair, shouldCreateExample, i.prompts.Err()
}

func defaultEncryptedOutputNameForFile(inputFile string) string {
	inputExt := filepath.Ext(inputFile)
	inputBase := strings.TrimSuffix(filepath.Base(inputFile), inputExt)
	return defaultEncryptedOutputName(inputBase, inputExt)
}

func defaultEncryptedOutputName(inputBase, inputExt string) string {
	return inputBase + ".enc" + inputExt
}

func (i *initializer) newInitFilePair(inputFile, outputFile, formatOverride string, interactive bool) (config.FilePair, error) {
	filePair := config.FilePair{PlaintextPath: defaultInitPlaintextFile}
	if inputFile != "" {
		filePair.PlaintextPath = inputFile
	}
	if outputFile != "" {
		filePair.EncryptedPath = outputFile
	} else {
		filePair.EncryptedPath = defaultEncryptedOutputNameForFile(filePair.PlaintextPath)
	}

	resolvedFormat, err := i.resolveInitFormatOverride(filePair.PlaintextPath, formatOverride, interactive)
	if err != nil {
		return config.FilePair{}, err
	}
	filePair.Format = resolvedFormat
	return filePair, nil
}

func (i *initializer) resolveInitFormatOverride(plaintextFile, providedFormat string, interactive bool) (string, error) {
	if normalizedFormat, ok := normalizeInitFormat(providedFormat); ok {
		return normalizedFormat, nil
	}
	if strings.TrimSpace(providedFormat) != "" {
		return "", fmt.Errorf("unsupported format override %q (supported: toml, yaml, json, env, ini, binary)", providedFormat)
	}

	if detectInitFormat(plaintextFile) != "" {
		return "", nil
	}

	if !interactive {
		return "", fmt.Errorf("could not detect format for %s, please pass --format (toml, yaml, json, env, ini, binary). Hint: pass --format binary if this should be encrypted as a binary file", plaintextFile)
	}

	format := i.promptInitFormatOverride(plaintextFile)
	return format, i.prompts.Err()
}

func (i *initializer) promptInitFormatOverride(plaintextFile string) string {
	for {
		input := i.prompts.PromptOptional("Format for " + plaintextFile + " (toml/yaml/json/env/ini/binary, optional)")
		if input == "" {
			return ""
		}

		if normalizedFormat, ok := normalizeInitFormat(input); ok {
			return normalizedFormat
		}

		i.output.Warning("Unsupported format. Use one of: toml, yaml, json, env, ini, binary")
	}
}

func detectInitFormat(filename string) string {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".toml":
		return "toml"
	case ".yaml", ".yml":
		return "yaml"
	case ".json":
		return "json"
	case ".env":
		return "env"
	case ".ini":
		return "ini"
	case ".bin", ".binary":
		return "binary"
	default:
		return ""
	}
}

func normalizeInitFormat(format string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "toml":
		return "toml", true
	case "yaml", "yml":
		return "yaml", true
	case "json":
		return "json", true
	case "env", "dotenv":
		return "env", true
	case "ini":
		return "ini", true
	case "binary", "bin":
		return "binary", true
	default:
		return "", false
	}
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

		bundle, err := agekey.GetIdentityBundle(keyFilePath)
		if err != nil {
			return "", fmt.Errorf("failed to parse existing key file: %w", err)
		}
		identity, err := age.ParseX25519Identity(bundle.Identities()[0])
		if err != nil {
			return "", fmt.Errorf("failed to parse existing owner identity: %w", err)
		}
		publicKey := identity.Recipient().String()
		return publicKey, nil
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

	tempFile, err := os.CreateTemp(".age", "keys.txt.*")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary key file: %w", err)
	}
	tempPath := tempFile.Name()
	defer func() { _ = os.Remove(tempPath) }()
	if _, err := tempFile.Write([]byte(keyContent)); err != nil {
		_ = tempFile.Close()
		return "", fmt.Errorf("failed to write key file: %w", err)
	}
	if err := tempFile.Chmod(0600); err != nil {
		_ = tempFile.Close()
		return "", fmt.Errorf("failed to set key file permissions: %w", err)
	}
	if err := tempFile.Sync(); err != nil {
		_ = tempFile.Close()
		return "", fmt.Errorf("failed to sync key file: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return "", fmt.Errorf("failed to close key file: %w", err)
	}
	if err := os.Rename(tempPath, keyFilePath); err != nil {
		return "", fmt.Errorf("failed to replace key file: %w", err)
	}

	publicKey := identity.Recipient().String()

	return publicKey, nil
}

// createExampleFile creates an example file from the input file
func (i *initializer) createExampleFile(inputFile string) {
	if _, err := os.Stat(inputFile); err == nil {
		exampleContent, err := os.ReadFile(inputFile)
		if err == nil {
			exampleFile := strings.TrimSuffix(inputFile, filepath.Ext(inputFile)) + ".example" + filepath.Ext(inputFile)
			if err := os.WriteFile(exampleFile, exampleContent, 0644); err != nil {
				i.output.Warning(fmt.Sprintf("Failed to create %s: %v", exampleFile, err))
			} else {
				i.output.Warning("Review " + exampleFile + " and remove sensitive values")
			}
		}
	} else {
		i.output.Warning(fmt.Sprintf("Input file %s does not exist yet, skipping example creation", inputFile))
	}
}
