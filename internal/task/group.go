package task

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/YewFence/YewSeal/internal/fileformat"
	"github.com/YewFence/YewSeal/internal/seal"
)

const (
	ModeEncrypt = "encrypt"
	ModeDecrypt = "decrypt"
)

var defaultEncryptPatterns = []string{
	"*.toml",
	"*.yaml",
	"*.yml",
	"*.json",
	"*.env",
	"*.ini",
	"*.bin",
	"*.binary",
	"!*.enc.toml",
	"!*.enc.yaml",
	"!*.enc.json",
	"!*.enc.env",
	"!*.enc.ini",
	"!*.enc.bin",
}

var defaultDecryptPatterns = []string{
	"*.enc.toml",
	"*.enc.yaml",
	"*.enc.json",
	"*.enc.env",
	"*.enc.ini",
	"*.enc.bin",
}

type GroupOptions struct {
	Root            string
	Patterns        []string
	FormatRules     []string
	UnknownAsBinary bool
	Mode            string
}

type FormatRule struct {
	Raw     string
	Format  string
	Matcher PatternMatcher
}

func BuildGroupFilePairs(opts GroupOptions) ([]FilePair, error) {
	mode := strings.TrimSpace(opts.Mode)
	if mode != ModeEncrypt && mode != ModeDecrypt {
		return nil, fmt.Errorf("invalid group mode %q", opts.Mode)
	}

	root := strings.TrimSpace(opts.Root)
	if root == "" {
		root = "."
	}
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("failed to stat group path %s: %w", root, err)
	}
	if !info.IsDir() {
		return singleFilePair(root, mode, opts)
	}

	patterns := opts.Patterns
	if len(patterns) == 0 {
		patterns = defaultPatterns(mode)
	}
	matcher, err := NewPatternMatcher(patterns)
	if err != nil {
		return nil, err
	}
	formatRules, err := parseFormatRules(opts.FormatRules)
	if err != nil {
		return nil, err
	}

	files := make([]string, 0)
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		decided, included := matcher.Decision(rel, entry.IsDir())
		if entry.IsDir() {
			if decided && !included {
				return filepath.SkipDir
			}
			return nil
		}
		if !decided || !included {
			return nil
		}
		files = append(files, path)
		return nil
	}); err != nil {
		return nil, fmt.Errorf("failed to scan %s: %w", root, err)
	}

	sort.Strings(files)
	if len(files) == 0 {
		return nil, fmt.Errorf("no files found in %s matching group patterns", root)
	}

	pairs := make([]FilePair, 0, len(files))
	for _, file := range files {
		rel, err := filepath.Rel(root, file)
		if err != nil {
			return nil, err
		}
		pair, err := groupFilePair(file, filepath.ToSlash(rel), mode, opts, formatRules)
		if err != nil {
			return nil, err
		}
		pairs = append(pairs, pair)
	}
	return pairs, nil
}

func BuildProjectGroupFilePairs(opts GroupOptions) ([]FilePair, error) {
	if opts.Mode != ModeDecrypt {
		return BuildGroupFilePairs(opts)
	}

	root := strings.TrimSpace(opts.Root)
	if root == "" {
		root = "."
	}
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("failed to stat group path %s: %w", root, err)
	}
	if !info.IsDir() {
		return BuildGroupFilePairs(opts)
	}

	patterns := opts.Patterns
	if len(patterns) == 0 {
		patterns = defaultPatterns(ModeEncrypt)
	}
	logicalMatcher, err := NewPatternMatcher(patterns)
	if err != nil {
		return nil, err
	}
	encryptedMatcher, err := NewPatternMatcher(defaultPatterns(ModeDecrypt))
	if err != nil {
		return nil, err
	}
	formatRules, err := parseFormatRules(opts.FormatRules)
	if err != nil {
		return nil, err
	}

	pairs := make([]FilePair, 0)
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		if entry.IsDir() {
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			decided, included := logicalMatcher.Decision(filepath.ToSlash(rel), true)
			if decided && !included {
				return filepath.SkipDir
			}
			return nil
		}
		encryptedRel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		decided, included := encryptedMatcher.Decision(filepath.ToSlash(encryptedRel), false)
		if !decided || !included {
			return nil
		}

		pair, ok, err := projectDecryptPairForEncrypted(root, path, logicalMatcher, formatRules)
		if err != nil {
			return err
		}
		if ok {
			pairs = append(pairs, pair)
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("failed to scan %s: %w", root, err)
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].EncryptedPath < pairs[j].EncryptedPath
	})
	if len(pairs) == 0 {
		return nil, fmt.Errorf("no encrypted files found in %s matching group patterns", root)
	}
	return pairs, nil
}

func projectDecryptPairForEncrypted(root, encryptedPath string, logicalMatcher PatternMatcher, formatRules []FormatRule) (FilePair, bool, error) {
	protocolPlaintext, protocolFormat, err := fileformat.PlaintextPathForEncrypted(encryptedPath, "")
	if err == nil {
		rel, relErr := filepath.Rel(root, protocolPlaintext)
		if relErr != nil {
			return FilePair{}, false, relErr
		}
		decided, included := logicalMatcher.Decision(filepath.ToSlash(rel), false)
		if decided && included {
			formatOverride := resolveFormatRule(filepath.ToSlash(rel), formatRules)
			if formatOverride == "" {
				formatOverride = protocolFormat
			}
			return FilePair{PlaintextPath: protocolPlaintext, EncryptedPath: encryptedPath, Format: formatOverride}, true, nil
		}
	}

	stemPlaintext, stemFormat, ok := fileformat.PlaintextStemPathForEncrypted(encryptedPath)
	if !ok {
		if err != nil {
			return FilePair{}, false, err
		}
		return FilePair{}, false, nil
	}
	rel, relErr := filepath.Rel(root, stemPlaintext)
	if relErr != nil {
		return FilePair{}, false, relErr
	}
	rel = filepath.ToSlash(rel)
	decided, included := logicalMatcher.Decision(rel, false)
	if !decided || !included {
		return FilePair{}, false, nil
	}
	formatOverride := resolveFormatRule(rel, formatRules)
	if formatOverride == "" {
		formatOverride = stemFormat
	}
	return FilePair{PlaintextPath: stemPlaintext, EncryptedPath: encryptedPath, Format: formatOverride}, true, nil
}

func singleFilePair(path, mode string, opts GroupOptions) ([]FilePair, error) {
	formatRules, err := parseFormatRules(opts.FormatRules)
	if err != nil {
		return nil, err
	}
	pair, err := groupFilePair(path, filepath.ToSlash(path), mode, opts, formatRules)
	if err != nil {
		return nil, err
	}
	return []FilePair{pair}, nil
}

func groupFilePair(path, matchPath, mode string, opts GroupOptions, formatRules []FormatRule) (FilePair, error) {
	switch mode {
	case ModeEncrypt:
		formatOverride, err := resolveEncryptGroupFormat(path, matchPath, formatRules, opts.UnknownAsBinary)
		if err != nil {
			return FilePair{}, err
		}
		encryptedPath, err := fileformat.EncryptPathForPlaintext(path, formatOverride)
		if err != nil {
			return FilePair{}, err
		}
		return FilePair{PlaintextPath: path, EncryptedPath: encryptedPath, Format: formatOverride}, nil
	case ModeDecrypt:
		formatOverride := resolveFormatRule(matchPath, formatRules)
		plaintextPath, pathFormat, err := fileformat.PlaintextPathForEncrypted(path, formatOverride)
		if err != nil {
			return FilePair{}, err
		}
		return FilePair{PlaintextPath: plaintextPath, EncryptedPath: path, Format: pathFormat}, nil
	default:
		return FilePair{}, fmt.Errorf("invalid group mode %q", mode)
	}
}

func parseFormatRules(values []string) ([]FormatRule, error) {
	rules := make([]FormatRule, 0, len(values))
	for _, raw := range values {
		parts := strings.SplitN(raw, "=", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
			return nil, fmt.Errorf("invalid format rule %q: expected <pattern>=<format>", raw)
		}
		format, ok := seal.NormalizeFormatOverride(parts[1])
		if !ok {
			return nil, fmt.Errorf("invalid format rule %q: unsupported format %q", raw, parts[1])
		}
		matcher, err := NewPatternMatcher([]string{strings.TrimSpace(parts[0])})
		if err != nil {
			return nil, fmt.Errorf("invalid format rule %q: %w", raw, err)
		}
		rules = append(rules, FormatRule{
			Raw:     raw,
			Format:  format,
			Matcher: matcher,
		})
	}
	return rules, nil
}

func resolveEncryptGroupFormat(path, matchPath string, rules []FormatRule, unknownAsBinary bool) (string, error) {
	format := resolveFormatRule(matchPath, rules)
	if format != "" {
		return format, nil
	}
	if detected, ok := seal.NormalizeFormatForPath(path); ok {
		return detected, nil
	}
	if unknownAsBinary {
		return "binary", nil
	}
	return "", fmt.Errorf("could not detect format for %s; add --format-rule <pattern>=<format> or use --unknown-as-binary", path)
}

func resolveFormatRule(matchPath string, rules []FormatRule) string {
	format := ""
	for _, rule := range rules {
		decided, included := rule.Matcher.Decision(matchPath, false)
		if decided && included {
			format = rule.Format
		}
	}
	return format
}

func defaultPatterns(mode string) []string {
	if mode == ModeDecrypt {
		return append([]string(nil), defaultDecryptPatterns...)
	}
	return append([]string(nil), defaultEncryptPatterns...)
}
