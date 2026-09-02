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

const (
	ValueSourceArgument     = "argument"
	ValueSourceExact        = "exact"
	ValueSourceScan         = "scan"
	ValueSourceProtocol     = "protocol"
	ValueSourceFilename     = "filename"
	ValueSourceConfigFormat = "config-format"

	PairSourceExact           = "exact"
	PairSourceScan            = "scan"
	PairSourceFileTarget      = "file-target"
	PairSourceDirectoryTarget = "directory-target"

	SelectedByCurrentDirectory = "current-directory"
	SelectedByPathTarget       = "path-target"
	SelectedByDirectoryScan    = "directory-scan"
)

type ValueSource struct {
	Kind       string
	ConfigPath string
	Detail     string
}

type ResolvedFilePair struct {
	PlaintextPath string
	EncryptedPath string
	Format        string
	ConfigPath    string

	Source          string
	SelectedBy      string
	PlaintextSource ValueSource
	EncryptedSource ValueSource
	FormatSource    ValueSource

	RecipientAliases []string
	Recipients       []string
	RecipientInfo    RecipientProvenance
}

type ResolvedSelection struct {
	Command         string
	FilePairs       []ResolvedFilePair
	AllConfigPairs  []ResolvedFilePair
	ConfigFiles     []LoadedFile
	ConfigMode      bool
	Unconfigured    bool
	CurrentDirScope string
	TargetKind      string
}

func ResolvePlanSelection(cfg *Config, opts SelectionOptions) (ResolvedSelection, error) {
	if strings.TrimSpace(opts.Target) == "" {
		if opts.OutputSet {
			return ResolvedSelection{}, fmt.Errorf("--output is only supported when the path target is a file")
		}
		if strings.TrimSpace(opts.Format) != "" {
			return ResolvedSelection{}, fmt.Errorf("--format is only supported in single-file mode")
		}
		allConfigPairs, err := configuredFilePairs(cfg, task.ModeEncrypt, groupRequestOptions{
			FormatRules:        opts.FormatRules,
			UnknownAsBinary:    opts.UnknownAsBinary,
			UnknownAsBinarySet: opts.UnknownAsBinarySet,
		})
		if err != nil {
			return ResolvedSelection{}, err
		}
		allResolved, err := resolveFilePairs(cfg, allConfigPairs, opts, SelectionResult{}, true)
		if err != nil {
			return ResolvedSelection{}, err
		}
		selected := make([]ResolvedFilePair, 0, len(allResolved))
		for _, filePair := range allResolved {
			plainInside, err := pathWithin(cwdFromConfig(cfg), filePair.PlaintextPath)
			if err != nil {
				return ResolvedSelection{}, err
			}
			encInside, err := pathWithin(cwdFromConfig(cfg), filePair.EncryptedPath)
			if err != nil {
				return ResolvedSelection{}, err
			}
			if plainInside || encInside {
				filePair.SelectedBy = SelectedByCurrentDirectory
				selected = append(selected, filePair)
			}
		}
		selected, err = filterResolvedPairsByPatterns(selected, opts.Patterns, cwdFromConfig(cfg))
		if err != nil {
			return ResolvedSelection{}, err
		}
		if len(selected) == 0 {
			return ResolvedSelection{}, fmt.Errorf("no configured file pairs selected for current directory scope %s", DisplayPath(cwdFromConfig(cfg), cwdFromConfig(cfg)))
		}
		return ResolvedSelection{
			Command:         "plan",
			FilePairs:       selected,
			AllConfigPairs:  allResolved,
			ConfigFiles:     append([]LoadedFile(nil), cfg.LoadedFiles...),
			ConfigMode:      true,
			CurrentDirScope: cwdFromConfig(cfg),
			TargetKind:      "none",
		}, nil
	}

	opts.Command = inferPlanCommand(cfg, opts)
	opts.AllowEmptyTarget = true
	opts.UseConfiguredDefault = true
	selection, err := ResolveSelection(cfg, opts)
	if err != nil {
		return ResolvedSelection{}, err
	}
	selection.Command = "plan"
	return selection, nil
}

func ResolveSelection(cfg *Config, opts SelectionOptions) (ResolvedSelection, error) {
	result, err := SelectFilePairs(cfg, opts)
	if err != nil {
		return ResolvedSelection{}, err
	}

	allConfigPairs, err := resolveFilePairs(cfg, result.AllConfigPairs, opts, result, true)
	if err != nil {
		return ResolvedSelection{}, err
	}
	selected, err := resolveFilePairs(cfg, result.FilePairs, opts, result, false)
	if err != nil {
		return ResolvedSelection{}, err
	}
	if err := checkWriteConflicts(opts.Command, selected); err != nil {
		return ResolvedSelection{}, err
	}

	return ResolvedSelection{
		Command:         opts.Command,
		FilePairs:       selected,
		AllConfigPairs:  allConfigPairs,
		ConfigFiles:     append([]LoadedFile(nil), cfg.LoadedFiles...),
		ConfigMode:      result.ConfigMode,
		Unconfigured:    result.Unconfigured,
		CurrentDirScope: result.CurrentDirScope,
		TargetKind:      targetKind(cfg, opts),
	}, nil
}

func inferPlanCommand(cfg *Config, opts SelectionOptions) string {
	target := strings.TrimSpace(opts.Target)
	if target == "" {
		return task.ModeEncrypt
	}
	targetPath := resolveCommandPath(cwdFromConfig(cfg), target)
	if _, _, err := fileformat.PlaintextPathForEncrypted(targetPath, ""); err == nil {
		return task.ModeDecrypt
	}
	return task.ModeEncrypt
}

func ResolvedFilePairsToFilePairs(filePairs []ResolvedFilePair) []FilePair {
	pairs := make([]FilePair, 0, len(filePairs))
	for _, filePair := range filePairs {
		pairs = append(pairs, FilePair{
			PlaintextPath: filePair.PlaintextPath,
			EncryptedPath: filePair.EncryptedPath,
			Format:        filePair.Format,
			Recipients:    cloneStringSlicePtr(filePair.RecipientAliases),
			ConfigPath:    filePair.ConfigPath,
		})
	}
	return pairs
}

func ResolvedFilePairsToTaskPairs(filePairs []ResolvedFilePair) []task.FilePair {
	pairs := make([]task.FilePair, 0, len(filePairs))
	for _, filePair := range filePairs {
		pairs = append(pairs, task.FilePair{
			PlaintextPath: filePair.PlaintextPath,
			EncryptedPath: filePair.EncryptedPath,
			Format:        filePair.Format,
			Recipients:    append([]string(nil), filePair.Recipients...),
		})
	}
	return pairs
}

func DisplayResolvedFilePairs(filePairs []ResolvedFilePair, cwd string) []ResolvedFilePair {
	display := make([]ResolvedFilePair, 0, len(filePairs))
	for _, filePair := range filePairs {
		filePair.PlaintextPath = DisplayPath(cwd, filePair.PlaintextPath)
		filePair.EncryptedPath = DisplayPath(cwd, filePair.EncryptedPath)
		filePair.ConfigPath = DisplayPath(cwd, filePair.ConfigPath)
		filePair.PlaintextSource.ConfigPath = DisplayPath(cwd, filePair.PlaintextSource.ConfigPath)
		filePair.EncryptedSource.ConfigPath = DisplayPath(cwd, filePair.EncryptedSource.ConfigPath)
		filePair.FormatSource.ConfigPath = DisplayPath(cwd, filePair.FormatSource.ConfigPath)
		display = append(display, filePair)
	}
	return display
}

func FormatValueSource(source ValueSource, cwd string) string {
	kind := strings.TrimSpace(source.Kind)
	if kind == "" {
		kind = "unknown"
	}
	if source.ConfigPath != "" {
		configPath := DisplayPath(cwd, source.ConfigPath)
		if source.Detail != "" {
			return strings.TrimSpace(configPath + " " + source.Detail)
		}
		return strings.TrimSpace(configPath + " " + kind)
	}
	if source.Detail != "" {
		return strings.TrimSpace(kind + " " + source.Detail)
	}
	return kind
}

func resolveFilePairs(cfg *Config, filePairs []FilePair, opts SelectionOptions, result SelectionResult, allConfig bool) ([]ResolvedFilePair, error) {
	resolved := make([]ResolvedFilePair, 0, len(filePairs))
	for _, filePair := range filePairs {
		next, err := resolveFilePair(cfg, filePair, opts, result, allConfig)
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, next)
	}
	return resolved, nil
}

func resolveFilePair(cfg *Config, filePair FilePair, opts SelectionOptions, result SelectionResult, allConfig bool) (ResolvedFilePair, error) {
	plainAbs := cleanAbsPath(filePair.PlaintextPath)
	encAbs := cleanAbsPath(filePair.EncryptedPath)
	filePair.PlaintextPath = plainAbs
	filePair.EncryptedPath = encAbs

	format, formatSource, err := resolveFinalFormat(filePair, opts)
	if err != nil {
		return ResolvedFilePair{}, err
	}

	source := pairSource(filePair, opts, result)
	selectedBy := selectedBy(opts, result, allConfig, source)
	plainSource, encSource := pathSources(filePair, opts, result, source)
	var resolvedRecipients ResolvedRecipients
	if opts.Command != task.ModeDecrypt && (hasRecipientPolicy(cfg) || filePair.Recipients != nil) {
		resolvedRecipients, err = cfg.ResolveFileRecipients(filePair)
		if err != nil {
			return ResolvedFilePair{}, err
		}
	}

	return ResolvedFilePair{
		PlaintextPath:    plainAbs,
		EncryptedPath:    encAbs,
		Format:           format,
		ConfigPath:       filePair.ConfigPath,
		Source:           source,
		SelectedBy:       selectedBy,
		PlaintextSource:  plainSource,
		EncryptedSource:  encSource,
		FormatSource:     formatSource,
		RecipientAliases: resolvedRecipients.Aliases,
		Recipients:       resolvedRecipients.Recipients,
		RecipientInfo:    resolvedRecipients.Provenance,
	}, nil
}

func resolveFinalFormat(filePair FilePair, opts SelectionOptions) (string, ValueSource, error) {
	cliFormat, err := ValidateFormatOverride(opts.Format)
	if err != nil {
		return "", ValueSource{}, err
	}
	if cliFormat != "" {
		return cliFormat, ValueSource{Kind: ValueSourceArgument, Detail: "--format"}, nil
	}

	configFormat := strings.TrimSpace(filePair.Format)
	if configFormat != "" {
		normalized, err := ValidateFormatOverride(configFormat)
		if err != nil {
			return "", ValueSource{}, err
		}
		if filePair.ConfigPath != "" {
			return normalized, ValueSource{Kind: ValueSourceConfigFormat, ConfigPath: filePair.ConfigPath, Detail: "format"}, nil
		}
		return normalized, ValueSource{Kind: ValueSourceScan}, nil
	}

	if format, ok := seal.NormalizeFormatForPath(filePair.PlaintextPath); ok {
		return format, ValueSource{Kind: ValueSourceFilename}, nil
	}
	if _, format, err := fileformat.PlaintextPathForEncrypted(filePair.EncryptedPath, ""); err == nil {
		return format, ValueSource{Kind: ValueSourceProtocol}, nil
	}
	return "", ValueSource{}, fmt.Errorf("could not detect format for %s (supported: toml, yaml, json, env, ini, binary)", filePair.PlaintextPath)
}

func pairSource(filePair FilePair, opts SelectionOptions, result SelectionResult) string {
	if filePair.Source == PairSourceScan {
		if strings.TrimSpace(opts.Target) != "" && result.Unconfigured {
			return PairSourceDirectoryTarget
		}
		return PairSourceScan
	}
	if filePair.Source == PairSourceFileTarget {
		return PairSourceFileTarget
	}
	if strings.TrimSpace(opts.Target) != "" {
		if result.Unconfigured {
			if targetKindFromPath(opts.Target) == "directory" {
				return PairSourceDirectoryTarget
			}
			return PairSourceFileTarget
		}
		return PairSourceExact
	}
	if filePair.ConfigPath != "" && filePair.Format == "" {
		return PairSourceExact
	}
	if filePair.ConfigPath != "" {
		return PairSourceExact
	}
	return PairSourceScan
}

func selectedBy(opts SelectionOptions, result SelectionResult, allConfig bool, source string) string {
	if allConfig {
		return "metadata"
	}
	if len(opts.Patterns) > 0 {
		return fmt.Sprintf("pattern %q", opts.Patterns[len(opts.Patterns)-1])
	}
	if strings.TrimSpace(opts.Target) != "" {
		if source == PairSourceDirectoryTarget {
			return SelectedByDirectoryScan
		}
		return SelectedByPathTarget
	}
	if result.ConfigMode {
		return SelectedByCurrentDirectory
	}
	return "default"
}

func pathSources(filePair FilePair, opts SelectionOptions, result SelectionResult, source string) (ValueSource, ValueSource) {
	if strings.TrimSpace(opts.Target) != "" && result.Unconfigured {
		if source == PairSourceDirectoryTarget {
			return ValueSource{Kind: ValueSourceScan}, ValueSource{Kind: ValueSourceProtocol}
		}
		if opts.Command == task.ModeEncrypt {
			encSource := ValueSource{Kind: ValueSourceProtocol}
			if opts.OutputSet {
				encSource = ValueSource{Kind: ValueSourceArgument, Detail: "--output"}
			}
			return ValueSource{Kind: ValueSourceArgument, Detail: "path"}, encSource
		}
		plainSource := ValueSource{Kind: ValueSourceProtocol}
		if opts.OutputSet {
			plainSource = ValueSource{Kind: ValueSourceArgument, Detail: "--output"}
		}
		return plainSource, ValueSource{Kind: ValueSourceArgument, Detail: "path"}
	}
	if source == PairSourceScan || source == PairSourceDirectoryTarget {
		return ValueSource{Kind: ValueSourceScan}, ValueSource{Kind: ValueSourceProtocol}
	}
	configSource := ValueSource{Kind: ValueSourceExact, ConfigPath: filePair.ConfigPath, Detail: "exact"}
	return configSource, configSource
}

func checkWriteConflicts(command string, filePairs []ResolvedFilePair) error {
	if command == "plan" {
		return nil
	}
	seen := make(map[string]ResolvedFilePair, len(filePairs))
	for _, filePair := range filePairs {
		target := filePair.EncryptedPath
		if command == task.ModeDecrypt {
			target = filePair.PlaintextPath
		}
		target = cleanAbsPath(target)
		if existing, ok := seen[target]; ok {
			if command == task.ModeDecrypt {
				return fmt.Errorf("multiple file pairs write to %s: decrypted from %s and %s", target, existing.EncryptedPath, filePair.EncryptedPath)
			}
			return fmt.Errorf("multiple file pairs write to %s: encrypted from %s and %s", target, existing.PlaintextPath, filePair.PlaintextPath)
		}
		seen[target] = filePair
	}
	return nil
}

func filterResolvedPairsByPatterns(filePairs []ResolvedFilePair, patterns []string, cwd string) ([]ResolvedFilePair, error) {
	if len(patterns) == 0 {
		return filePairs, nil
	}
	matcher, err := task.NewPatternMatcher(patterns)
	if err != nil {
		return nil, err
	}
	selected := make([]ResolvedFilePair, 0, len(filePairs))
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
			filePair.SelectedBy = fmt.Sprintf("pattern %q", patterns[len(patterns)-1])
			selected = append(selected, filePair)
		}
	}
	return selected, nil
}

func targetKind(cfg *Config, opts SelectionOptions) string {
	target := strings.TrimSpace(opts.Target)
	if target == "" {
		return "none"
	}
	path := resolveCommandPath(cwdFromConfig(cfg), target)
	return targetKindFromPath(path)
}

func targetKindFromPath(path string) string {
	info, err := filepath.Abs(path)
	if err != nil {
		return "file"
	}
	stat, statErr := os.Stat(info)
	if statErr == nil && stat.IsDir() {
		return "directory"
	}
	return "file"
}
