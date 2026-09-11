package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/YewFence/YewSeal/internal/fileformat"
	"github.com/YewFence/YewSeal/internal/seal"
	"github.com/YewFence/YewSeal/internal/task"
)

type SelectionOptions struct {
	Command             string
	Targets             []string
	Output              string
	OutputSet           bool
	RequireSingleTarget bool
	AllowEmptyTarget    bool
	StrictRecipients    bool
}

type SelectionResult struct {
	FilePairs       []FilePair
	AllConfigPairs  []FilePair
	ConfigMode      bool
	TargetKind      string
	CurrentDirScope string
	// SelectedBy 记录每个被选中的 FilePair 的来源标签，键为明文路径。
	SelectedBy map[string]string
}

func SelectFilePairs(cfg *Config, opts SelectionOptions) (SelectionResult, error) {
	allConfigPairs, err := configuredFilePairs(cfg, opts.Command)
	if err != nil {
		return SelectionResult{}, err
	}
	result, err := selectConfiguredFilePairs(cfg, allConfigPairs, opts)
	if err != nil {
		return SelectionResult{}, err
	}
	if opts.OutputSet {
		pair := &result.FilePairs[0]
		format, _, err := resolveFinalFormat(*pair)
		if err != nil {
			return SelectionResult{}, err
		}
		pair.Format = format
		if opts.Command == task.ModeEncrypt {
			pair.EncryptedPath = resolveCommandPath(cwdFromConfig(cfg), opts.Output)
		} else {
			pair.PlaintextPath = resolveCommandPath(cwdFromConfig(cfg), opts.Output)
		}
	}
	return result, nil
}

func selectConfiguredFilePairs(cfg *Config, allConfigPairs []FilePair, opts SelectionOptions) (SelectionResult, error) {
	if !policyForCommand(opts.Command).writes && opts.OutputSet {
		return SelectionResult{}, fmt.Errorf("%s does not support output overrides", opts.Command)
	}
	if hasTargets(opts.Targets) {
		result, err := selectTargetFilePairs(cfg, allConfigPairs, opts)
		if err != nil {
			return SelectionResult{}, err
		}
		if opts.RequireSingleTarget && len(result.FilePairs) != 1 {
			return SelectionResult{}, fmt.Errorf("%s requires exactly one target", opts.Command)
		}
		return result, nil
	}

	if opts.RequireSingleTarget && !opts.AllowEmptyTarget {
		return SelectionResult{}, fmt.Errorf("%s requires exactly one target", opts.Command)
	}
	if opts.OutputSet {
		return SelectionResult{}, fmt.Errorf("--output is only supported when the path target is a file")
	}

	selected, err := filterCurrentDirectoryScope(allConfigPairs, opts.Command, cwdFromConfig(cfg))
	if err != nil {
		return SelectionResult{}, err
	}
	if opts.RequireSingleTarget && len(selected) != 1 {
		return SelectionResult{}, fmt.Errorf("%s requires exactly one target", opts.Command)
	}
	if len(selected) == 0 {
		return SelectionResult{}, fmt.Errorf("no configured file pairs selected for current directory scope %s", DisplayPath(cwdFromConfig(cfg), cwdFromConfig(cfg)))
	}

	return SelectionResult{
		FilePairs:       selected,
		AllConfigPairs:  allConfigPairs,
		ConfigMode:      true,
		TargetKind:      "none",
		CurrentDirScope: cwdFromConfig(cfg),
	}, nil
}

func hasTargets(targets []string) bool {
	for _, target := range targets {
		if strings.TrimSpace(target) != "" {
			return true
		}
	}
	return false
}

func ValidateFormatOverride(format string) (string, error) {
	if strings.TrimSpace(format) == "" {
		return "", nil
	}
	parsed, ok := seal.NormalizeFormatOverride(format)
	if !ok {
		return "", fmt.Errorf("unsupported format %q (supported: toml, yaml, json, env, ini, binary)", format)
	}
	return parsed, nil
}

func ValidateFilePairs(filePairs []FilePair) ([]FilePair, error) {
	validated := make([]FilePair, 0, len(filePairs))
	for _, filePair := range filePairs {
		next, err := validateFilePair(filePair)
		if err != nil {
			return nil, err
		}
		validated = append(validated, next)
	}
	return validated, nil
}

func DisplayFilePairs(filePairs []FilePair, cwd string) []FilePair {
	display := make([]FilePair, 0, len(filePairs))
	for _, filePair := range filePairs {
		filePair.PlaintextPath = DisplayPath(cwd, filePair.PlaintextPath)
		filePair.EncryptedPath = DisplayPath(cwd, filePair.EncryptedPath)
		display = append(display, filePair)
	}
	return display
}

func PrintSelection(verbose bool, cfg *Config, result SelectionResult) {
	if !verbose {
		return
	}
	if len(cfg.LoadedFiles) > 0 {
		fmt.Printf("Loaded %d config files\n", len(cfg.LoadedFiles))
	}
	fmt.Printf("Selected %d file pairs\n", len(result.FilePairs))
	if result.CurrentDirScope != "" {
		fmt.Printf("Using current directory scope: %s\n", DisplayPath(cwdFromConfig(cfg), result.CurrentDirScope))
	}
	for _, filePair := range result.FilePairs {
		fmt.Printf("  %s -> %s\n", DisplayPath(cwdFromConfig(cfg), filePair.PlaintextPath), DisplayPath(cwdFromConfig(cfg), filePair.EncryptedPath))
	}
}

func CurrentDir(cfg *Config) string {
	return cwdFromConfig(cfg)
}

func DisplayPath(cwd, path string) string {
	if strings.TrimSpace(path) == "" {
		return path
	}
	rel, err := filepath.Rel(cwd, path)
	if err == nil && rel != "." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".." {
		return filepath.Clean(rel)
	}
	if err == nil && rel == "." {
		return "."
	}
	return filepath.Clean(path)
}

// selectTargetFilePairs 把每个位置参数解析为一组 FilePair 并取并集：
// 精确路径命中任一已登记映射（明文或密文路径均可），目录按命令主侧收缩
// 范围，含 glob 元字符的参数按模式与已登记映射求交集。任一参数零命中
// 都会报错。
func selectTargetFilePairs(cfg *Config, allConfigPairs []FilePair, opts SelectionOptions) (SelectionResult, error) {
	cwd := cwdFromConfig(cfg)
	result := SelectionResult{
		AllConfigPairs: allConfigPairs,
		SelectedBy:     make(map[string]string),
	}
	seen := make(map[string]struct{})
	kinds := make(map[string]struct{})
	dirScopes := make([]string, 0, 1)

	addPairs := func(pairs []FilePair, kind, label string) {
		for _, pair := range pairs {
			key := cleanAbsPath(pair.PlaintextPath)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			result.FilePairs = append(result.FilePairs, pair)
			result.SelectedBy[key] = label
			kinds[kind] = struct{}{}
		}
	}

	for _, rawTarget := range opts.Targets {
		target := strings.TrimSpace(rawTarget)
		if target == "" {
			continue
		}

		if task.HasPatternMeta(target) {
			pairs, err := matchPairsByPattern(allConfigPairs, target, opts.Command, cwd)
			if err != nil {
				return SelectionResult{}, err
			}
			if len(pairs) == 0 {
				return SelectionResult{}, fmt.Errorf("target pattern %s matches no configured file pairs", target)
			}
			addPairs(pairs, "pattern", fmt.Sprintf("pattern %q", target))
			continue
		}

		targetAbs := resolveCommandPath(cwd, target)
		if filePair, matched := findConfiguredPair(allConfigPairs, targetAbs); matched {
			addPairs([]FilePair{filePair}, "file", SelectedByPathTarget)
			continue
		}

		info, statErr := os.Stat(targetAbs)
		switch {
		case statErr == nil && info.IsDir():
			if opts.OutputSet {
				return SelectionResult{}, fmt.Errorf("--output is only supported when the path target is a file")
			}
			pairs, err := filterCurrentDirectoryScope(allConfigPairs, opts.Command, targetAbs)
			if err != nil {
				return SelectionResult{}, err
			}
			if len(pairs) == 0 {
				return SelectionResult{}, fmt.Errorf("no configured file pairs selected for target directory %s", target)
			}
			addPairs(pairs, "directory", SelectedByDirectoryTarget)
			dirScopes = append(dirScopes, targetAbs)
		case statErr == nil:
			return SelectionResult{}, fmt.Errorf("target %s is not configured", target)
		case os.IsNotExist(statErr):
			return SelectionResult{}, fmt.Errorf("target file %s does not exist", target)
		default:
			return SelectionResult{}, fmt.Errorf("failed to stat %s: %w", target, statErr)
		}
	}

	if len(kinds) == 1 {
		for kind := range kinds {
			result.TargetKind = kind
		}
	} else if len(kinds) > 1 {
		result.TargetKind = "multiple"
	}
	if len(dirScopes) == 1 && len(kinds) == 1 {
		result.CurrentDirScope = dirScopes[0]
	}
	return result, nil
}

func matchPairsByPattern(filePairs []FilePair, pattern, command, cwd string) ([]FilePair, error) {
	matcher, err := task.NewPatternMatcher([]string{pattern})
	if err != nil {
		return nil, err
	}
	selected := make([]FilePair, 0, len(filePairs))
	for _, filePair := range filePairs {
		if pairMatchesPattern(matcher, filePair, command, cwd) {
			selected = append(selected, filePair)
		}
	}
	return selected, nil
}

// pairMatchesPattern 按命令主侧匹配：decrypt 匹配密文路径，plan 和 diff
// 匹配任一侧，其余命令匹配明文路径。
func pairMatchesPattern(matcher task.PatternMatcher, filePair FilePair, command, cwd string) bool {
	matches := func(path string) bool {
		decided, included := matcher.Decision(DisplayPath(cwd, path), false)
		return decided && included
	}
	switch command {
	case task.ModeDecrypt, task.ModeView:
		return matches(filePair.EncryptedPath)
	case task.ModePlan, task.ModeDiff:
		return matches(filePair.PlaintextPath) || matches(filePair.EncryptedPath)
	default:
		return matches(filePair.PlaintextPath)
	}
}

func configuredFilePairs(cfg *Config, mode string) ([]FilePair, error) {
	groupPairs, err := scopedConfigGroupPairs(cfg, mode)
	if err != nil {
		return nil, err
	}
	pairs := append(groupPairs, cfg.Encryption.Files...)
	deduped, err := dedupeFilePairs(pairs, mode)
	if err != nil {
		return nil, err
	}
	return deduped, nil
}

func configuredEncryptedPaths(cfg *Config) []string {
	if cfg == nil {
		return nil
	}

	paths := make([]string, 0, len(cfg.Encryption.Files))
	for _, filePair := range cfg.Encryption.Files {
		if strings.TrimSpace(filePair.EncryptedPath) == "" {
			continue
		}
		paths = append(paths, cleanAbsPath(filePair.EncryptedPath))
	}
	return paths
}

func scopedConfigGroupPairs(cfg *Config, mode string) ([]FilePair, error) {
	policy := policyForCommand(mode)
	groups := cfg.GetGroups()
	if len(groups) == 0 {
		return nil, nil
	}

	pairs := make([]FilePair, 0)
	excludedPaths := configuredEncryptedPaths(cfg)
	seenRecipients := make(map[string][]string)
	for _, group := range groups {
		root := group.ConfigDir
		if strings.TrimSpace(root) == "" {
			root = cwdFromConfig(cfg)
		}
		groupAliases, groupRecipientSource := effectiveGroupAuthorization(cfg, group)
		canonical := []string(nil)
		if !policy.historicalRecipients && groupAliases != nil {
			var resolveErr error
			canonical, resolveErr = cfg.resolveAliases(*groupAliases)
			if resolveErr != nil {
				return nil, fmt.Errorf("group %s: %w", group.ConfigPath, resolveErr)
			}
		}
		taskPairs, err := task.BuildProjectGroupFilePairs(task.GroupOptions{
			Root:            root,
			Patterns:        group.Patterns,
			FormatRules:     group.FormatRules,
			ExcludedPaths:   excludedPaths,
			UnknownAsBinary: group.UnknownAsBinary,
			Mode:            policy.discoveryMode,
		})
		if err != nil {
			return nil, err
		}
		for _, taskPair := range taskPairs {
			if _, explicit := findConfiguredPair(cfg.Encryption.Files, cleanAbsPath(taskPair.PlaintextPath)); explicit {
				continue
			}
			if mode == task.ModeDiff || mode == task.ModePlan {
				if _, explicit := findConfiguredPair(cfg.Encryption.Files, cleanAbsPath(taskPair.EncryptedPath)); explicit {
					continue
				}
			}
			key := cleanAbsPath(taskPair.PlaintextPath)
			if previous, ok := seenRecipients[key]; ok && !equalStrings(previous, canonical) {
				return nil, fmt.Errorf("conflicting recipient sets for %s", taskPair.PlaintextPath)
			}
			seenRecipients[key] = append([]string(nil), canonical...)
			pairs = append(pairs, FilePair{
				PlaintextPath:   taskPair.PlaintextPath,
				EncryptedPath:   taskPair.EncryptedPath,
				Format:          taskPair.Format,
				ConfigPath:      group.ConfigPath,
				ConfigDir:       root,
				Recipients:      cloneOptionalStrings(groupAliases),
				RecipientSource: groupRecipientSource,
				Source:          "scan",
			})
		}
	}
	return pairs, nil
}

func effectiveGroupAuthorization(cfg *Config, group GroupConfig) (*[]string, ValueSource) {
	if group.Recipients != nil {
		return cloneOptionalStrings(group.Recipients), group.RecipientSource
	}
	if cfg.Recipients.Defaults != nil {
		return cloneOptionalStrings(cfg.Recipients.Defaults), ValueSource{
			Kind:       "defaults",
			ConfigPath: cfg.Recipients.DefaultsConfigPath,
			Detail:     "recipients.defaults",
		}
	}
	return nil, ValueSource{}
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func filterCurrentDirectoryScope(filePairs []FilePair, command, cwd string) ([]FilePair, error) {
	command = policyForCommand(command).scopeMode
	selected := make([]FilePair, 0, len(filePairs))
	for _, filePair := range filePairs {
		path := filePair.PlaintextPath
		if command == task.ModeDecrypt {
			path = filePair.EncryptedPath
		}
		inside, err := pathWithin(cwd, path)
		if err != nil {
			return nil, err
		}
		if command == task.ModePlan && !inside {
			inside, err = pathWithin(cwd, filePair.EncryptedPath)
			if err != nil {
				return nil, err
			}
		}
		if inside {
			selected = append(selected, filePair)
		}
	}
	return selected, nil
}

func findConfiguredPair(filePairs []FilePair, targetAbs string) (FilePair, bool) {
	targetAbs = cleanAbsPath(targetAbs)
	for _, filePair := range filePairs {
		if cleanAbsPath(filePair.PlaintextPath) == targetAbs || cleanAbsPath(filePair.EncryptedPath) == targetAbs {
			return filePair, true
		}
	}
	return FilePair{}, false
}

func dedupeFilePairs(filePairs []FilePair, mode string) ([]FilePair, error) {
	result := make([]FilePair, 0, len(filePairs))
	for _, filePair := range filePairs {
		nextPlaintext := cleanAbsPath(filePair.PlaintextPath)
		nextEncrypted := cleanAbsPath(filePair.EncryptedPath)
		filtered := result[:0]
		for _, existing := range result {
			if cleanAbsPath(existing.PlaintextPath) == nextPlaintext || cleanAbsPath(existing.EncryptedPath) == nextEncrypted {
				// An explicitly configured file may intentionally override a scanned group pair.
				if existing.Source == PairSourceScan && filePair.Source != PairSourceScan {
					continue
				}
				if (mode == task.ModeDiff || mode == task.ModePlan) && existing.Source == PairSourceScan && filePair.Source == PairSourceScan &&
					(cleanAbsPath(existing.PlaintextPath) != nextPlaintext || cleanAbsPath(existing.EncryptedPath) != nextEncrypted || existing.Format != filePair.Format) {
					operation := "comparison"
					if mode == task.ModePlan {
						operation = "plan"
					}
					return nil, fmt.Errorf("conflicting group file pairs for %s: %s -> %s and %s -> %s", operation, existing.PlaintextPath, existing.EncryptedPath, filePair.PlaintextPath, filePair.EncryptedPath)
				}
				if existing.Source != PairSourceScan && filePair.Source != PairSourceScan {
					return nil, fmt.Errorf("conflicting file pairs for plaintext %q or encrypted %q", filePair.PlaintextPath, filePair.EncryptedPath)
				}
				continue
			}
			filtered = append(filtered, existing)
		}
		result = append(filtered, filePair)
	}
	return result, nil
}

func validateFilePair(filePair FilePair) (FilePair, error) {
	format, err := effectiveFormat(filePair.PlaintextPath, filePair.Format)
	if err != nil {
		if filePair.Format == "" {
			if _, pathFormat, pathErr := fileformat.PlaintextPathForEncrypted(filePair.EncryptedPath, ""); pathErr == nil {
				format = pathFormat
			} else {
				return FilePair{}, err
			}
		} else {
			return FilePair{}, err
		}
	}
	filePair.Format = format
	return filePair, nil
}

func effectiveFormat(path, formatOverride string) (string, error) {
	if formatOverride != "" {
		return formatOverride, nil
	}
	format, ok := seal.NormalizeFormatForPath(path)
	if !ok {
		return "", fmt.Errorf("could not detect format for %s (supported: toml, yaml, json, env, ini, binary)", path)
	}
	return format, nil
}

func cwdFromConfig(cfg *Config) string {
	if cfg != nil && strings.TrimSpace(cfg.CurrentDir) != "" {
		return cfg.CurrentDir
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return cwd
}
func hasRecipientPolicy(cfg *Config) bool {
	return cfg != nil && (cfg.Recipients.Defaults != nil || len(cfg.Recipients.Registry) > 0)
}

func resolveCommandPath(cwd, path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Clean(filepath.Join(cwd, path))
}

func pathWithin(root, path string) (bool, error) {
	rootAbs := cleanAbsPath(root)
	pathAbs := cleanAbsPath(path)
	rel, err := filepath.Rel(rootAbs, pathAbs)
	if err != nil {
		return false, err
	}
	rel = filepath.ToSlash(rel)
	return rel == "." || (!strings.HasPrefix(rel, "../") && rel != ".."), nil
}
