package project

import (
	"fmt"
	"os"
	"strings"

	"github.com/YewFence/YewSeal/internal/config"
)

const (
	decryptedFilesHeader = "# YewSeal - Decrypted configuration files"
	privateKeysHeader    = "# YewSeal - Age private keys"
	privateKeyPath       = ".age/keys.txt"
)

// UpdateGitignore creates or updates .gitignore with plaintext file entries.
func UpdateGitignore(filePairs []config.FilePair) error {
	plaintextFiles := uniquePlaintextFiles(filePairs)
	if len(plaintextFiles) == 0 {
		return nil
	}

	if existingData, err := os.ReadFile(".gitignore"); err == nil {
		updatedContent, changed := mergeGitignoreEntries(string(existingData), plaintextFiles)
		if !changed {
			fmt.Println("⏭️  .gitignore already contains YewSeal entries")
			return nil
		}

		if err := os.WriteFile(".gitignore", []byte(updatedContent), 0644); err != nil {
			return fmt.Errorf("failed to update .gitignore: %w", err)
		}
		fmt.Println("✅ Updated .gitignore")
		return nil
	}

	content := renderGitignoreSection(plaintextFiles)
	if err := os.WriteFile(".gitignore", []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to create .gitignore: %w", err)
	}
	fmt.Println("✅ Created .gitignore")
	return nil
}

func mergeGitignoreEntries(content string, plaintextFiles []string) (string, bool) {
	trimmedContent := strings.TrimRight(content, "\n")
	changed := false

	if !strings.Contains(trimmedContent, decryptedFilesHeader) {
		section := renderGitignoreSection(plaintextFiles)
		if trimmedContent == "" {
			return section, true
		}
		return trimmedContent + "\n\n" + section, true
	}

	for _, plaintextFile := range plaintextFiles {
		if containsExactLine(trimmedContent, plaintextFile) {
			continue
		}
		insertBefore := "\n" + privateKeysHeader
		if strings.Contains(trimmedContent, insertBefore) {
			trimmedContent = strings.Replace(trimmedContent, insertBefore, "\n"+plaintextFile+insertBefore, 1)
		} else {
			trimmedContent += "\n" + plaintextFile
		}
		changed = true
	}

	if !containsExactLine(trimmedContent, privateKeysHeader) {
		trimmedContent += "\n\n" + privateKeysHeader
		changed = true
	}
	if !containsExactLine(trimmedContent, privateKeyPath) {
		if strings.Contains(trimmedContent, privateKeysHeader) {
			trimmedContent = strings.Replace(trimmedContent, privateKeysHeader, privateKeysHeader+"\n"+privateKeyPath, 1)
		} else {
			trimmedContent += "\n" + privateKeyPath
		}
		changed = true
	}

	return trimmedContent + "\n", changed
}

func renderGitignoreSection(plaintextFiles []string) string {
	var lines []string
	lines = append(lines, decryptedFilesHeader)
	lines = append(lines, plaintextFiles...)
	lines = append(lines, "")
	lines = append(lines, privateKeysHeader, privateKeyPath)
	return strings.Join(lines, "\n") + "\n"
}

func uniquePlaintextFiles(filePairs []config.FilePair) []string {
	seen := make(map[string]struct{}, len(filePairs))
	files := make([]string, 0, len(filePairs))

	for _, filePair := range filePairs {
		plaintextFile := strings.TrimSpace(filePair.PlaintextPath)
		if plaintextFile == "" {
			continue
		}
		if _, ok := seen[plaintextFile]; ok {
			continue
		}
		seen[plaintextFile] = struct{}{}
		files = append(files, plaintextFile)
	}

	return files
}

func containsExactLine(content, target string) bool {
	for _, line := range strings.Split(content, "\n") {
		if strings.TrimSpace(line) == target {
			return true
		}
	}
	return false
}
