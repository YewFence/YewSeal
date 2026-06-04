package batch

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

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
	"!*.enc.toml.yaml",
	"!*.enc.yaml",
	"!*.enc.json",
	"!*.enc.env",
	"!*.enc.ini",
	"!*.enc.bin",
}

var defaultDecryptPatterns = []string{
	"*.enc.toml.yaml",
	"*.enc.yaml",
	"*.enc.json",
	"*.enc.env",
	"*.enc.ini",
	"*.enc.bin",
}

type ScanOptions struct {
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

func BuildScanFilePairs(opts ScanOptions) ([]FilePair, error) {
	mode := strings.TrimSpace(opts.Mode)
	if mode != ModeEncrypt && mode != ModeDecrypt {
		return nil, fmt.Errorf("invalid scan mode %q", opts.Mode)
	}

	root := strings.TrimSpace(opts.Root)
	if root == "" {
		root = "."
	}
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("failed to stat scan path %s: %w", root, err)
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
		return nil, fmt.Errorf("no files found in %s matching scan patterns", root)
	}

	pairs := make([]FilePair, 0, len(files))
	for _, file := range files {
		rel, err := filepath.Rel(root, file)
		if err != nil {
			return nil, err
		}
		pair, err := scanFilePair(file, filepath.ToSlash(rel), mode, opts, formatRules)
		if err != nil {
			return nil, err
		}
		pairs = append(pairs, pair)
	}
	return pairs, nil
}

func BuildProjectScanFilePairs(opts ScanOptions) ([]FilePair, error) {
	if opts.Mode != ModeDecrypt {
		return BuildScanFilePairs(opts)
	}

	root := strings.TrimSpace(opts.Root)
	if root == "" {
		root = "."
	}
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("failed to stat scan path %s: %w", root, err)
	}
	if !info.IsDir() {
		return BuildScanFilePairs(opts)
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
		if path == root || entry.IsDir() {
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
		return nil, fmt.Errorf("no encrypted files found in %s matching scan patterns", root)
	}
	return pairs, nil
}

func projectDecryptPairForEncrypted(root, encryptedPath string, logicalMatcher PatternMatcher, formatRules []FormatRule) (FilePair, bool, error) {
	protocolPlaintext, protocolFormat, err := PlaintextPathForEncrypted(encryptedPath, "")
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

	stemPlaintext, stemFormat, ok := plaintextStemPathForEncrypted(encryptedPath)
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

func singleFilePair(path, mode string, opts ScanOptions) ([]FilePair, error) {
	formatRules, err := parseFormatRules(opts.FormatRules)
	if err != nil {
		return nil, err
	}
	pair, err := scanFilePair(path, filepath.ToSlash(path), mode, opts, formatRules)
	if err != nil {
		return nil, err
	}
	return []FilePair{pair}, nil
}

func scanFilePair(path, matchPath, mode string, opts ScanOptions, formatRules []FormatRule) (FilePair, error) {
	switch mode {
	case ModeEncrypt:
		formatOverride, err := resolveEncryptScanFormat(path, matchPath, formatRules, opts.UnknownAsBinary)
		if err != nil {
			return FilePair{}, err
		}
		encryptedPath, err := EncryptPathForPlaintext(path, formatOverride)
		if err != nil {
			return FilePair{}, err
		}
		return FilePair{PlaintextPath: path, EncryptedPath: encryptedPath, Format: formatOverride}, nil
	case ModeDecrypt:
		formatOverride := resolveFormatRule(matchPath, formatRules)
		plaintextPath, pathFormat, err := PlaintextPathForEncrypted(path, formatOverride)
		if err != nil {
			return FilePair{}, err
		}
		return FilePair{PlaintextPath: plaintextPath, EncryptedPath: path, Format: pathFormat}, nil
	default:
		return FilePair{}, fmt.Errorf("invalid scan mode %q", mode)
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

func resolveEncryptScanFormat(path, matchPath string, rules []FormatRule, unknownAsBinary bool) (string, error) {
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

func EncryptPathForPlaintext(path, formatOverride string) (string, error) {
	format := formatOverride
	detected := ""
	if format == "" {
		pathFormat, ok := seal.NormalizeFormatForPath(path)
		if !ok {
			return "", fmt.Errorf("could not detect format for %s", path)
		}
		format = pathFormat
		detected = pathFormat
	} else {
		detected, _ = seal.NormalizeFormatForPath(path)
	}

	dir := filepath.Dir(path)
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	if detected != format {
		name = base
	}

	switch format {
	case "toml":
		if ext == "" {
			name = base
		}
		return filepath.Join(dir, name+".enc.toml.yaml"), nil
	case "yaml":
		if strings.EqualFold(ext, ".yml") {
			name = strings.TrimSuffix(base, ext)
		}
		return filepath.Join(dir, name+".enc.yaml"), nil
	case "json":
		return filepath.Join(dir, name+".enc.json"), nil
	case "env":
		return filepath.Join(dir, name+".enc.env"), nil
	case "ini":
		return filepath.Join(dir, name+".enc.ini"), nil
	case "binary":
		if ext == "" {
			name = base
		}
		return filepath.Join(dir, name+".enc.bin"), nil
	default:
		return "", fmt.Errorf("unsupported format %q", format)
	}
}

func PlaintextPathForEncrypted(path, formatOverride string) (string, string, error) {
	if formatOverride != "" {
		plaintextPath, err := plaintextPathForEncryptedWithFormat(path, formatOverride)
		return plaintextPath, formatOverride, err
	}

	dir := filepath.Dir(path)
	base := filepath.Base(path)
	suffixes := []struct {
		encryptedSuffix string
		plaintextSuffix string
		format          string
	}{
		{".enc.toml.yaml", ".toml", "toml"},
		{".enc.yaml", ".yaml", "yaml"},
		{".enc.json", ".json", "json"},
		{".enc.env", ".env", "env"},
		{".enc.ini", ".ini", "ini"},
		{".enc.bin", ".bin", "binary"},
	}
	for _, suffix := range suffixes {
		if strings.HasSuffix(strings.ToLower(base), suffix.encryptedSuffix) {
			name := base[:len(base)-len(suffix.encryptedSuffix)]
			return filepath.Join(dir, name+suffix.plaintextSuffix), suffix.format, nil
		}
	}
	return "", "", fmt.Errorf("encrypted filename %s does not follow the yewseal scan protocol; add a format rule to make the plaintext format unambiguous", path)
}

func plaintextStemPathForEncrypted(path string) (string, string, bool) {
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	stem, format, ok := encryptedStemAndFormat(base)
	if !ok {
		return "", "", false
	}
	return filepath.Join(dir, stem), format, true
}

func encryptedStemAndFormat(base string) (string, string, bool) {
	suffixes := []struct {
		encryptedSuffix string
		format          string
	}{
		{".enc.toml.yaml", "toml"},
		{".enc.yaml", "yaml"},
		{".enc.json", "json"},
		{".enc.env", "env"},
		{".enc.ini", "ini"},
		{".enc.bin", "binary"},
	}
	lowerBase := strings.ToLower(base)
	for _, suffix := range suffixes {
		if strings.HasSuffix(lowerBase, suffix.encryptedSuffix) {
			return base[:len(base)-len(suffix.encryptedSuffix)], suffix.format, true
		}
	}
	return "", "", false
}

func plaintextPathForEncryptedWithFormat(path, format string) (string, error) {
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	lowerBase := strings.ToLower(base)
	suffix := ".enc." + encryptedFormatSuffix(format)
	if strings.HasSuffix(lowerBase, suffix) {
		name := base[:len(base)-len(suffix)]
		if filepath.Ext(name) != "" {
			return filepath.Join(dir, name), nil
		}
		return filepath.Join(dir, name+"."+plaintextFormatSuffix(format)), nil
	}
	if strings.HasSuffix(lowerBase, ".enc.toml.yaml") {
		name := base[:len(base)-len(".enc.toml.yaml")]
		if filepath.Ext(name) != "" {
			return filepath.Join(dir, name), nil
		}
		return filepath.Join(dir, name+"."+plaintextFormatSuffix(format)), nil
	}
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	if strings.HasSuffix(strings.ToLower(name), ".enc") {
		name = name[:len(name)-len(".enc")]
	}
	return filepath.Join(dir, name+"."+plaintextFormatSuffix(format)), nil
}

func defaultPatterns(mode string) []string {
	if mode == ModeDecrypt {
		return append([]string(nil), defaultDecryptPatterns...)
	}
	return append([]string(nil), defaultEncryptPatterns...)
}

func encryptedFormatSuffix(format string) string {
	if format == "toml" || format == "yaml" {
		return "yaml"
	}
	if format == "binary" {
		return "bin"
	}
	return format
}

func plaintextFormatSuffix(format string) string {
	if format == "binary" {
		return "bin"
	}
	return format
}
