package project

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/YewFence/YewSeal/internal/config"
)

const (
	managedGitignoreBegin = "# BEGIN YewSeal managed entries"
	managedGitignoreEnd   = "# END YewSeal managed entries"
	privateKeyPath        = ".age/keys.txt"
)

// UpdateGitignore rebuilds the managed block from the complete source
// configuration policy, preserving all content outside the block.
func UpdateGitignore(dir string, filePairs []config.FilePair) error {
	plaintextFiles, err := uniquePlaintextFiles(filePairs, dir)
	if err != nil {
		return err
	}
	path := filepath.Join(dir, ".gitignore")
	existingData, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read %s: %w", path, err)
	}
	updated, err := mergeGitignoreEntries(string(existingData), plaintextFiles)
	if err != nil {
		return fmt.Errorf("failed to update %s: %w", path, err)
	}
	if updated == string(existingData) {
		return nil
	}
	if err := os.WriteFile(path, []byte(updated), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}
	return nil
}

func mergeGitignoreEntries(content string, plaintextFiles []string) (string, error) {
	start, end := -1, -1
	offset := 0
	for line := range strings.SplitAfterSeq(content, "\n") {
		marker := strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
		switch marker {
		case managedGitignoreBegin:
			if start != -1 || end != -1 {
				return "", fmt.Errorf("invalid YewSeal managed block: expected exactly one BEGIN line followed by one END line")
			}
			start = offset
		case managedGitignoreEnd:
			if start == -1 || end != -1 {
				return "", fmt.Errorf("invalid YewSeal managed block: expected exactly one BEGIN line followed by one END line")
			}
			end = offset + len(line)
		}
		offset += len(line)
	}
	if start != -1 && end == -1 {
		return "", fmt.Errorf("invalid YewSeal managed block: missing END line")
	}
	block := renderGitignoreSection(plaintextFiles)
	if start != -1 {
		return content[:start] + block + content[end:], nil
	}
	if content == "" {
		return block, nil
	}
	separator := "\n"
	if !strings.HasSuffix(content, "\n") {
		separator += "\n"
	}
	return content + separator + block, nil
}

func renderGitignoreSection(plaintextFiles []string) string {
	lines := []string{managedGitignoreBegin}
	for _, path := range plaintextFiles {
		lines = append(lines, literalGitignorePath(path))
	}
	lines = append(lines, literalGitignorePath(privateKeyPath), managedGitignoreEnd)
	return strings.Join(lines, "\n") + "\n"
}

func literalGitignorePath(path string) string {
	return "/" + strings.NewReplacer(
		`\`, `\\`,
		"*", `\*`,
		"?", `\?`,
		"[", `\[`,
		"]", `\]`,
		" ", `\ `,
	).Replace(path)
}

func uniquePlaintextFiles(filePairs []config.FilePair, dir string) ([]string, error) {
	root, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{}, len(filePairs))
	files := make([]string, 0, len(filePairs))
	for _, pair := range filePairs {
		path := pair.PlaintextPath
		if strings.ContainsAny(path, "\r\n") {
			return nil, fmt.Errorf("cannot represent plaintext path %q in .gitignore: path contains a line break", path)
		}
		if strings.TrimSpace(path) == "" {
			continue
		}
		if !filepath.IsAbs(path) {
			path = filepath.Join(root, path)
		}
		relative, err := filepath.Rel(root, path)
		if err != nil || !filepath.IsLocal(relative) {
			continue
		}
		relative = filepath.ToSlash(relative)
		if _, exists := seen[relative]; exists {
			continue
		}
		seen[relative] = struct{}{}
		files = append(files, relative)
	}
	sort.Strings(files)
	return files, nil
}
