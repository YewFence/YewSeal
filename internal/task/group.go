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
	ModeDiff    = "diff"
	ModePlan    = "plan"
	ModeView    = "view"
	ModeClean   = "clean"
	ModeVerify  = "verify"
)

// protocolFilePatterns is how decryption discovers ciphertext: YewSeal
// protocol files are recognized by their .enc.* suffix. This is the discovery
// mechanism, not a user-facing default — group patterns are always required
// and filter the logical plaintext paths.
var protocolFilePatterns = []string{
	"*.enc.toml",
	"*.enc.yaml",
	"*.enc.json",
	"*.enc.env",
	"*.enc.ini",
	"*.enc.bin",
}

// ScanStats counts successful WalkDir visits and directories expanded across scans.
type ScanStats struct {
	Entries     int
	Directories int
}

type GroupOptions struct {
	Exclude         []string
	Stats           *ScanStats
	Root            string
	Patterns        []string
	FormatRules     []string
	ExcludedPaths   []string
	UnknownAsBinary bool
	AllowEmpty      bool
	Mode            string
}

type FormatRule struct {
	Raw     string
	Format  string
	Matcher PatternMatcher
}

func BuildGroupFilePairs(opts GroupOptions) ([]FilePair, error) {
	if opts.Mode == ModePlan || opts.Mode == ModeClean || opts.Mode == ModeDiff {
		return BuildProjectGroupFilePairs(opts)
	}
	mode := strings.TrimSpace(opts.Mode)
	if mode != ModeEncrypt && mode != ModeDecrypt {
		return nil, fmt.Errorf("invalid group mode %q", opts.Mode)
	}
	discovery := NewGroupDiscovery(opts)
	if err := discovery.scan(); err != nil {
		return nil, err
	}
	pairs, err := discovery.pathPairs(mode)
	if err != nil {
		return nil, err
	}
	if len(pairs) == 0 && !opts.AllowEmpty {
		return nil, fmt.Errorf("no files found in %s matching group patterns", discovery.root)
	}
	return pairs, nil
}

// GroupDiscovery scans and prunes a Group once per selection preparation.
// It interprets each side only when requested. Its options and returned pairs
// must not be mutated during that preparation.
type GroupDiscovery struct {
	opts           GroupOptions
	root           string
	scanned        bool
	singleFile     bool
	files          []string
	matcher        PatternMatcher
	excludeMatcher PatternMatcher
	formatRules    []FormatRule
	plaintextPairs []FilePair
	encryptedPairs []FilePair
	plaintextRead  bool
	encryptedRead  bool
}

// NewGroupDiscovery creates discovery state scoped to one selection preparation.
func NewGroupDiscovery(opts GroupOptions) *GroupDiscovery {
	return &GroupDiscovery{opts: opts}
}

// FilePairs interprets the requested sides from one directory traversal,
// allowing empty results. Bidirectional modes prefer real plaintext paths.
func (d *GroupDiscovery) FilePairs(mode string) ([]FilePair, error) {
	switch mode {
	case ModeEncrypt, ModeDecrypt, ModeDiff, ModePlan, ModeClean:
	default:
		return nil, fmt.Errorf("invalid group mode %q", mode)
	}
	if err := d.scan(); err != nil {
		return nil, err
	}
	switch mode {
	case ModePlan, ModeClean:
		plainPairs, err := d.FilePairs(ModeEncrypt)
		if err != nil {
			return nil, err
		}
		encryptedPairs, err := d.FilePairs(ModeDecrypt)
		if err != nil {
			return nil, err
		}
		return mergeGroupFilePairs(plainPairs, encryptedPairs), nil
	case ModeDiff:
		return d.FilePairs(ModeEncrypt)
	case ModeEncrypt:
		if !d.plaintextRead {
			pairs, err := d.pathPairs(ModeEncrypt)
			if err != nil {
				return nil, err
			}
			d.plaintextPairs, d.plaintextRead = pairs, true
		}
		return d.plaintextPairs, nil
	default:
		if !d.encryptedRead {
			pairs, err := d.ciphertextPairs()
			if err != nil {
				return nil, err
			}
			d.encryptedPairs, d.encryptedRead = pairs, true
		}
		return d.encryptedPairs, nil
	}
}

func (d *GroupDiscovery) scan() error {
	if d.scanned {
		return nil
	}
	if len(d.opts.Patterns) == 0 {
		return fmt.Errorf("group patterns must not be empty")
	}
	d.root = strings.TrimSpace(d.opts.Root)
	if d.root == "" {
		d.root = "."
	}
	info, err := os.Stat(d.root)
	if err != nil {
		return fmt.Errorf("failed to stat group path %s: %w", d.root, err)
	}
	if !info.IsDir() {
		d.singleFile, d.scanned = true, true
		return nil
	}
	d.matcher, err = NewPatternMatcher(d.opts.Patterns)
	if err != nil {
		return err
	}
	d.formatRules, err = parseFormatRules(d.opts.FormatRules)
	if err != nil {
		return err
	}
	d.excludeMatcher, err = NewPatternMatcher(d.opts.Exclude)
	if err != nil {
		return fmt.Errorf("invalid exclude: %w", err)
	}
	if err := filepath.WalkDir(d.root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.opts.Stats != nil {
			d.opts.Stats.Entries++
		}
		if path == d.root {
			if d.opts.Stats != nil {
				d.opts.Stats.Directories++
			}
			return nil
		}
		if entry.IsDir() && (entry.Name() == ".git" || entry.Name() == ".jj") {
			return filepath.SkipDir
		}
		rel, err := filepath.Rel(d.root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if entry.IsDir() {
			decided, included := d.matcher.Decision(rel, true)
			_, excluded := d.excludeMatcher.Decision(rel, true)
			if excluded || (decided && !included) {
				return filepath.SkipDir
			}
			if d.opts.Stats != nil {
				d.opts.Stats.Directories++
			}
			return nil
		}
		d.files = append(d.files, rel)
		return nil
	}); err != nil {
		return fmt.Errorf("failed to scan %s: %w", d.root, err)
	}
	d.scanned = true
	return nil
}

func (d *GroupDiscovery) pathPairs(mode string) ([]FilePair, error) {
	if d.singleFile {
		return singleFilePair(d.root, mode, d.opts)
	}
	excludedPaths := make(map[string]struct{}, len(d.opts.ExcludedPaths))
	for _, path := range d.opts.ExcludedPaths {
		excludedPaths[filepath.Clean(path)] = struct{}{}
	}
	pairs := make([]FilePair, 0)
	for _, rel := range d.files {
		path := filepath.Join(d.root, filepath.FromSlash(rel))
		if mode == ModeEncrypt {
			if _, _, ok := fileformat.EncryptedStemAndFormat(filepath.Base(path)); ok {
				continue
			}
			if _, excluded := excludedPaths[filepath.Clean(path)]; excluded {
				continue
			}
		}
		decided, included := d.matcher.Decision(rel, false)
		_, excluded := d.excludeMatcher.Decision(rel, false)
		if excluded || !decided || !included {
			continue
		}
		pair, err := groupFilePair(path, rel, mode, d.opts, d.formatRules)
		if err != nil {
			return nil, err
		}
		pairs = append(pairs, pair)
	}
	sort.Slice(pairs, func(i, j int) bool {
		if mode == ModeDecrypt {
			return pairs[i].EncryptedPath < pairs[j].EncryptedPath
		}
		return pairs[i].PlaintextPath < pairs[j].PlaintextPath
	})
	return pairs, nil
}

func (d *GroupDiscovery) ciphertextPairs() ([]FilePair, error) {
	if d.singleFile {
		return singleFilePair(d.root, ModeDecrypt, d.opts)
	}
	encryptedMatcher, err := NewPatternMatcher(protocolFilePatterns)
	if err != nil {
		return nil, err
	}
	pairs := make([]FilePair, 0)
	for _, rel := range d.files {
		decided, included := encryptedMatcher.Decision(rel, false)
		if !decided || !included {
			continue
		}
		path := filepath.Join(d.root, filepath.FromSlash(rel))
		pair, ok, err := projectDecryptPairForEncrypted(d.root, path, d.matcher, d.formatRules)
		if err != nil {
			return nil, fmt.Errorf("failed to scan %s: %w", d.root, err)
		}
		if !ok {
			continue
		}
		logicalRel, err := filepath.Rel(d.root, pair.PlaintextPath)
		if err != nil {
			return nil, fmt.Errorf("failed to scan %s: %w", d.root, err)
		}
		_, excluded := d.excludeMatcher.Decision(filepath.ToSlash(logicalRel), false)
		if !excluded {
			pairs = append(pairs, pair)
		}
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].EncryptedPath < pairs[j].EncryptedPath })
	return pairs, nil
}

func BuildProjectGroupFilePairs(opts GroupOptions) ([]FilePair, error) {
	discovery := NewGroupDiscovery(opts)
	pairs, err := discovery.FilePairs(opts.Mode)
	if err != nil {
		return nil, err
	}
	if len(pairs) == 0 && !opts.AllowEmpty && opts.Mode != ModeClean && opts.Mode != ModeDiff {
		if opts.Mode == ModeDecrypt {
			return nil, fmt.Errorf("no encrypted files found in %s matching group patterns", discovery.root)
		}
		return nil, fmt.Errorf("no files found in %s matching group patterns", discovery.root)
	}
	return pairs, nil
}

func mergeGroupFilePairs(plainPairs, encryptedPairs []FilePair) []FilePair {
	// A real plaintext path is more precise than reversing a normalized ciphertext suffix.
	byEncrypted := make(map[string]FilePair)
	pairs := make([]FilePair, 0, len(plainPairs)+len(encryptedPairs))
	for _, pair := range plainPairs {
		key := filepath.Clean(pair.EncryptedPath)
		byEncrypted[key] = pair
		pairs = append(pairs, pair)
	}
	for _, pair := range encryptedPairs {
		key := filepath.Clean(pair.EncryptedPath)
		if _, exists := byEncrypted[key]; exists {
			continue
		}
		byEncrypted[key] = pair
		pairs = append(pairs, pair)
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].PlaintextPath < pairs[j].PlaintextPath })
	return pairs
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
