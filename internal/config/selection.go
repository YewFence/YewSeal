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
	Command              string
	Target               string
	Output               string
	OutputSet            bool
	Format               string
	Patterns             []string
	FormatRules          []string
	UnknownAsBinary      bool
	UnknownAsBinarySet   bool
	RequireSingleTarget  bool
	AllowEmptyTarget     bool
	UseConfiguredDefault bool
}

type SelectionResult struct {
	FilePairs       []FilePair
	AllConfigPairs  []FilePair
	ConfigMode      bool
	Unconfigured    bool
	CurrentDirScope string
}

func SelectFilePairs(cfg *Config, opts SelectionOptions) (SelectionResult, error) {
	cliFormat, err := ValidateFormatOverride(opts.Format)
	if err != nil {
		return SelectionResult{}, err
	}

	allConfigPairs, err := configuredFilePairs(cfg, opts.Command, groupRequestOptions{
		FormatRules:        opts.FormatRules,
		UnknownAsBinary:    opts.UnknownAsBinary,
		UnknownAsBinarySet: opts.UnknownAsBinarySet,
	})
	if err != nil {
		return SelectionResult{}, err
	}

	target := strings.TrimSpace(opts.Target)
	if target != "" {
		selected, unconfigured, err := selectTargetFilePairs(cfg, allConfigPairs, opts, cliFormat)
		if err != nil {
			return SelectionResult{}, err
		}
		return SelectionResult{
			FilePairs:      selected,
			AllConfigPairs: allConfigPairs,
			Unconfigured:   unconfigured,
		}, nil
	}

	if opts.RequireSingleTarget && !opts.AllowEmptyTarget {
		return SelectionResult{}, fmt.Errorf("%s requires exactly one target", opts.Command)
	}
	if opts.OutputSet {
		return SelectionResult{}, fmt.Errorf("--output is only supported when the path target is a file")
	}
	if cliFormat != "" {
		return SelectionResult{}, fmt.Errorf("--format is only supported in single-file mode")
	}
	if len(allConfigPairs) == 0 && opts.UseConfiguredDefault {
		allConfigPairs = cfg.GetFiles()
	}

	selected, err := filterCurrentDirectoryScope(allConfigPairs, opts.Command, cwdFromConfig(cfg))
	if err != nil {
		return SelectionResult{}, err
	}
	selected, err = filterPairsByPatterns(selected, opts.Patterns, cwdFromConfig(cfg))
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
		CurrentDirScope: cwdFromConfig(cfg),
	}, nil
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

func ResolveFormatOverride(cliFormat string, filePair FilePair) (string, error) {
	validatedFormat, err := ValidateFormatOverride(cliFormat)
	if err != nil {
		return "", err
	}
	if validatedFormat != "" {
		return validatedFormat, nil
	}
	return filePair.Format, nil
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

type groupRequestOptions struct {
	Patterns           []string
	FormatRules        []string
	UnknownAsBinary    bool
	UnknownAsBinarySet bool
}

func selectTargetFilePairs(cfg *Config, allConfigPairs []FilePair, opts SelectionOptions, cliFormat string) ([]FilePair, bool, error) {
	targetAbs, err := cleanAbsTarget(opts.Target)
	if err != nil {
		return nil, false, err
	}
	info, statErr := os.Stat(targetAbs)
	if statErr != nil && !os.IsNotExist(statErr) {
		return nil, false, fmt.Errorf("failed to stat %s: %w", opts.Target, statErr)
	}

	if filePair, matched := findConfiguredPair(allConfigPairs, targetAbs); matched {
		formatOverride, err := ResolveFormatOverride(cliFormat, filePair)
		if err != nil {
			return nil, false, err
		}
		if opts.OutputSet {
			if opts.Command == task.ModeEncrypt {
				filePair.EncryptedPath = resolveCommandPath(cwdFromConfig(cfg), opts.Output)
			} else {
				filePair.PlaintextPath = resolveCommandPath(cwdFromConfig(cfg), opts.Output)
			}
		}
		filePair.Format = formatOverride
		return []FilePair{filePair}, false, nil
	}

	if statErr == nil && info.IsDir() {
		if cliFormat != "" {
			return nil, false, fmt.Errorf("--format is only supported in single-file mode")
		}
		if opts.OutputSet {
			return nil, false, fmt.Errorf("--output is only supported when the path target is a file")
		}
		pairs, err := directoryTargetPairs(cfg, targetAbs, opts)
		return pairs, true, err
	}

	if statErr != nil && os.IsNotExist(statErr) {
		return nil, false, fmt.Errorf("target file %s does not exist", opts.Target)
	}

	filePair, err := unconfiguredFileTargetPair(cfg, opts, cliFormat, targetAbs)
	if err != nil {
		return nil, true, err
	}
	return []FilePair{filePair}, true, nil
}

func configuredFilePairs(cfg *Config, mode string, req groupRequestOptions) ([]FilePair, error) {
	pairs := cfg.Encryption.Files
	if !cfg.UserConfig {
		pairs = cfg.GetFiles()
	}
	groupPairs, err := scopedConfigGroupPairs(cfg, mode, req)
	if err != nil {
		return nil, err
	}
	pairs = append(pairs, groupPairs...)
	return dedupeFilePairs(pairs), nil
}

func scopedConfigGroupPairs(cfg *Config, mode string, req groupRequestOptions) ([]FilePair, error) {
	groups := cfg.GetGroups()
	if len(groups) == 0 {
		if !hasGroupRequestOptions(req) {
			return nil, nil
		}
		groups = []GroupConfig{{ConfigDir: cwdFromConfig(cfg)}}
	}

	pairs := make([]FilePair, 0)
	for _, group := range groups {
		root := group.ConfigDir
		if strings.TrimSpace(root) == "" {
			root = cwdFromConfig(cfg)
		}
		groupReq := req
		if len(groupReq.Patterns) == 0 {
			groupReq.Patterns = group.Patterns
		}
		formatRules := append([]string(nil), group.FormatRules...)
		formatRules = append(formatRules, req.FormatRules...)
		groupReq.FormatRules = formatRules
		if !groupReq.UnknownAsBinarySet {
			groupReq.UnknownAsBinary = group.UnknownAsBinary
		}
		taskPairs, err := task.BuildProjectGroupFilePairs(task.GroupOptions{
			Root:            root,
			Patterns:        groupReq.Patterns,
			FormatRules:     groupReq.FormatRules,
			UnknownAsBinary: groupReq.UnknownAsBinary,
			Mode:            mode,
		})
		if err != nil {
			return nil, err
		}
		for _, taskPair := range taskPairs {
			pairs = append(pairs, FilePair{
				PlaintextPath: taskPair.PlaintextPath,
				EncryptedPath: taskPair.EncryptedPath,
				Format:        taskPair.Format,
				ConfigPath:    group.ConfigPath,
				ConfigDir:     root,
				Source:        "scan",
			})
		}
	}
	return pairs, nil
}

func directoryTargetPairs(cfg *Config, root string, opts SelectionOptions) ([]FilePair, error) {
	taskPairs, err := groupFilePairsFromRequest(cfg, root, opts.Command, groupRequestOptions{
		Patterns:           opts.Patterns,
		FormatRules:        opts.FormatRules,
		UnknownAsBinary:    opts.UnknownAsBinary,
		UnknownAsBinarySet: opts.UnknownAsBinarySet,
	})
	if err != nil {
		return nil, err
	}
	pairs := make([]FilePair, 0, len(taskPairs))
	for _, taskPair := range taskPairs {
		pairs = append(pairs, FilePair{
			PlaintextPath: taskPair.PlaintextPath,
			EncryptedPath: taskPair.EncryptedPath,
			Format:        taskPair.Format,
			Source:        "scan",
		})
	}

	allConfigPairs, err := configuredFilePairs(cfg, opts.Command, groupRequestOptions{})
	if err != nil {
		return nil, err
	}
	for i, pair := range pairs {
		if configured, matched := findConfiguredPair(allConfigPairs, cleanAbsPath(pair.PlaintextPath)); matched {
			pairs[i] = configured
			continue
		}
		if configured, matched := findConfiguredPair(allConfigPairs, cleanAbsPath(pair.EncryptedPath)); matched {
			pairs[i] = configured
		}
	}
	return dedupeFilePairs(pairs), nil
}

func groupFilePairsFromRequest(cfg *Config, root, mode string, req groupRequestOptions) ([]task.FilePair, error) {
	groups := cfg.GetGroups()
	if len(groups) == 0 {
		groups = []GroupConfig{{}}
	}

	pairs := make([]task.FilePair, 0)
	for _, group := range groups {
		patterns := group.Patterns
		if len(req.Patterns) > 0 {
			patterns = append([]string(nil), req.Patterns...)
		}
		formatRules := append([]string(nil), group.FormatRules...)
		formatRules = append(formatRules, req.FormatRules...)
		unknownAsBinary := group.UnknownAsBinary
		if req.UnknownAsBinarySet {
			unknownAsBinary = req.UnknownAsBinary
		}
		groupPairs, err := task.BuildGroupFilePairs(task.GroupOptions{
			Root:            root,
			Patterns:        patterns,
			FormatRules:     formatRules,
			UnknownAsBinary: unknownAsBinary,
			Mode:            mode,
		})
		if err != nil {
			return nil, err
		}
		pairs = append(pairs, groupPairs...)
	}
	return pairs, nil
}

func unconfiguredFileTargetPair(cfg *Config, opts SelectionOptions, cliFormat, targetAbs string) (FilePair, error) {
	if opts.Command == task.ModeEncrypt {
		encryptedPath := resolveCommandPath(cwdFromConfig(cfg), opts.Output)
		var err error
		if encryptedPath == "" {
			encryptedPath, err = fileformat.EncryptPathForPlaintext(targetAbs, cliFormat)
			if err != nil {
				return FilePair{}, err
			}
		}
		return FilePair{
			PlaintextPath: targetAbs,
			EncryptedPath: encryptedPath,
			Format:        cliFormat,
			Source:        "file-target",
		}, nil
	}

	plaintextPath := resolveCommandPath(cwdFromConfig(cfg), opts.Output)
	inferredPlaintextPath, pathFormat, err := fileformat.PlaintextPathForEncrypted(targetAbs, cliFormat)
	if err != nil {
		return FilePair{}, err
	}
	if plaintextPath == "" {
		plaintextPath = inferredPlaintextPath
	}
	formatOverride := cliFormat
	if formatOverride == "" {
		formatOverride = pathFormat
	}
	return FilePair{
		PlaintextPath: plaintextPath,
		EncryptedPath: targetAbs,
		Format:        formatOverride,
		Source:        "file-target",
	}, nil
}

func filterCurrentDirectoryScope(filePairs []FilePair, command, cwd string) ([]FilePair, error) {
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
		if inside {
			selected = append(selected, filePair)
		}
	}
	return selected, nil
}

func filterPairsByPatterns(filePairs []FilePair, patterns []string, cwd string) ([]FilePair, error) {
	if len(patterns) == 0 {
		return filePairs, nil
	}
	matcher, err := task.NewPatternMatcher(patterns)
	if err != nil {
		return nil, err
	}
	selected := make([]FilePair, 0, len(filePairs))
	for _, filePair := range filePairs {
		plain := DisplayPath(cwd, filePair.PlaintextPath)
		enc := DisplayPath(cwd, filePair.EncryptedPath)
		plainDecided, plainIncluded := matcher.Decision(filepath.ToSlash(plain), false)
		encDecided, encIncluded := matcher.Decision(filepath.ToSlash(enc), false)
		included := false
		if plainDecided {
			included = plainIncluded
		}
		if encDecided {
			included = encIncluded
		}
		if included {
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

func dedupeFilePairs(filePairs []FilePair) []FilePair {
	result := make([]FilePair, 0, len(filePairs))
	for _, filePair := range filePairs {
		nextPlaintext := cleanAbsPath(filePair.PlaintextPath)
		nextEncrypted := cleanAbsPath(filePair.EncryptedPath)
		filtered := result[:0]
		for _, existing := range result {
			if cleanAbsPath(existing.PlaintextPath) == nextPlaintext || cleanAbsPath(existing.EncryptedPath) == nextEncrypted {
				continue
			}
			filtered = append(filtered, existing)
		}
		result = append(filtered, filePair)
	}
	return result
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

func hasGroupRequestOptions(req groupRequestOptions) bool {
	return len(req.Patterns) > 0 || len(req.FormatRules) > 0 || req.UnknownAsBinarySet
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

func cleanAbsTarget(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path %s: %w", path, err)
	}
	return filepath.Clean(abs), nil
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
	return rel == "." || (!strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".."), nil
}
