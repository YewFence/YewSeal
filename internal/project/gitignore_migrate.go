package project

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/YewFence/YewSeal/internal/config"
)

const (
	legacyPlaintextHeader = "# YewSeal - Decrypted configuration files"
	legacyPrivateHeader   = "# YewSeal - Age private keys"
)

// MigrateGitignore removes a structurally complete legacy section and rebuilds
// the managed block for configuration owners. Other directories only lose the
// legacy section. Ambiguous files are left untouched and return an error.
func MigrateGitignore(dir string, pairs []config.FilePair, ownsConfig bool) (bool, error) {
	path := filepath.Join(dir, ".gitignore")
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}
	content := string(data)
	updated, err := removeLegacyGitignoreSection(content)
	if err != nil {
		return false, err
	}
	// Validate existing managed boundaries even when only removing a legacy section.
	if _, err := mergeGitignoreEntries(updated, nil); err != nil {
		return false, err
	}
	if ownsConfig {
		paths, err := uniquePlaintextFiles(pairs, dir)
		if err != nil {
			return false, err
		}
		updated, err = mergeGitignoreEntries(updated, paths)
		if err != nil {
			return false, err
		}
	}
	if updated == content {
		return false, nil
	}
	if err := os.WriteFile(path, []byte(updated), 0644); err != nil {
		return false, err
	}
	return true, nil
}

func removeLegacyGitignoreSection(content string) (string, error) {
	lines := strings.SplitAfter(content, "\n")
	start, private := -1, -1
	for index, line := range lines {
		marker := strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
		switch marker {
		case legacyPlaintextHeader:
			if start != -1 {
				return "", fmt.Errorf("ambiguous legacy YewSeal section: duplicate plaintext header")
			}
			start = index
		case legacyPrivateHeader:
			if private != -1 {
				return "", fmt.Errorf("ambiguous legacy YewSeal section: duplicate private key header")
			}
			private = index
		}
	}
	if start == -1 && private == -1 {
		return content, nil
	}
	if start == -1 || private <= start || private+1 >= len(lines) {
		return "", fmt.Errorf("ambiguous legacy YewSeal section: expected plaintext header, private key header, and key path in order")
	}
	key := strings.TrimSuffix(strings.TrimSuffix(lines[private+1], "\n"), "\r")
	if key != privateKeyPath {
		return "", fmt.Errorf("ambiguous legacy YewSeal section: private key header must be followed by %s", privateKeyPath)
	}
	for _, line := range lines[start+1 : private] {
		entry := strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
		if strings.ContainsAny(entry, "*?[]\\\r") || strings.HasPrefix(strings.TrimSpace(entry), "#") || strings.HasPrefix(strings.TrimSpace(entry), "!") {
			return "", fmt.Errorf("ambiguous legacy YewSeal section: possible custom rule %q", entry)
		}
	}
	return strings.Join(lines[:start], "") + strings.Join(lines[private+2:], ""), nil
}
