package config

import (
	"fmt"
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

	PairSourceExact = "exact"
	PairSourceScan  = "scan"

	SelectedByCurrentDirectory = "current-directory"
	SelectedByPathTarget       = "path-target"
	SelectedByDirectoryTarget  = "directory-target"
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
	RecipientWarning string
}

type ResolvedSelection struct {
	Command         string
	FilePairs       []ResolvedFilePair
	AllConfigPairs  []ResolvedFilePair
	ConfigFiles     []LoadedFile
	ConfigMode      bool
	CurrentDirScope string
	TargetKind      string
}

func ResolveSelection(cfg *Config, opts SelectionOptions) (ResolvedSelection, error) {
	filePairs, err := configuredFilePairs(cfg, opts.Command)
	if err != nil {
		return ResolvedSelection{}, err
	}

	// Resolve the entire configured scope before applying selectors or output overrides.
	allConfigPairs, err := resolveFilePairs(cfg, filePairs, opts)
	if err != nil {
		return ResolvedSelection{}, err
	}
	result, err := selectConfiguredFilePairs(cfg, filePairs, opts)
	if err != nil {
		return ResolvedSelection{}, err
	}
	byPlaintext := make(map[string]ResolvedFilePair, len(allConfigPairs))
	for _, pair := range allConfigPairs {
		byPlaintext[pair.PlaintextPath] = pair
	}
	selected := make([]ResolvedFilePair, 0, len(result.FilePairs))
	for _, pair := range result.FilePairs {
		resolved := byPlaintext[cleanAbsPath(pair.PlaintextPath)]
		resolved.SelectedBy = selectedBy(result, pair)
		if opts.OutputSet {
			output := resolveCommandPath(cwdFromConfig(cfg), opts.Output)
			source := ValueSource{Kind: ValueSourceArgument, Detail: "--output"}
			if opts.Command == task.ModeEncrypt {
				resolved.EncryptedPath, resolved.EncryptedSource = output, source
			} else {
				resolved.PlaintextPath, resolved.PlaintextSource = output, source
			}
		}
		selected = append(selected, resolved)
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
		CurrentDirScope: result.CurrentDirScope,
		TargetKind:      result.TargetKind,
	}, nil
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

func resolveFilePairs(cfg *Config, filePairs []FilePair, opts SelectionOptions) ([]ResolvedFilePair, error) {
	resolved := make([]ResolvedFilePair, 0, len(filePairs))
	for _, filePair := range filePairs {
		next, err := resolveFilePair(cfg, filePair, opts)
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, next)
	}
	return resolved, nil
}

func resolveFilePair(cfg *Config, filePair FilePair, opts SelectionOptions) (ResolvedFilePair, error) {
	plainAbs := cleanAbsPath(filePair.PlaintextPath)
	encAbs := cleanAbsPath(filePair.EncryptedPath)
	filePair.PlaintextPath = plainAbs
	filePair.EncryptedPath = encAbs

	format, formatSource, err := resolveFinalFormat(filePair)
	if err != nil {
		return ResolvedFilePair{}, err
	}

	source := PairSourceExact
	plainSource := ValueSource{Kind: ValueSourceExact, ConfigPath: filePair.ConfigPath, Detail: "exact"}
	encSource := plainSource
	if filePair.Source == PairSourceScan {
		source = PairSourceScan
		plainSource = ValueSource{Kind: ValueSourceScan}
		encSource = ValueSource{Kind: ValueSourceProtocol}
	}
	var resolvedRecipients ResolvedRecipients
	recipientWarning := ""
	if !policyForCommand(opts.Command).historicalRecipients || opts.StrictRecipients {
		resolvedRecipients, err = cfg.ResolveFileRecipients(filePair)
		if err != nil {
			return ResolvedFilePair{}, err
		}
	} else if hasRecipientPolicy(cfg) || filePair.Recipients != nil {
		resolvedRecipients, err = cfg.ResolveFileRecipients(filePair)
		if err != nil {
			recipientWarning = formatRecipientWarning(filePair, err)
			resolvedRecipients = ResolvedRecipients{}
		}
	}

	return ResolvedFilePair{
		PlaintextPath:    plainAbs,
		EncryptedPath:    encAbs,
		Format:           format,
		ConfigPath:       filePair.ConfigPath,
		Source:           source,
		SelectedBy:       "metadata",
		PlaintextSource:  plainSource,
		EncryptedSource:  encSource,
		FormatSource:     formatSource,
		RecipientAliases: resolvedRecipients.Aliases,
		Recipients:       resolvedRecipients.Recipients,
		RecipientInfo:    resolvedRecipients.Provenance,
		RecipientWarning: recipientWarning,
	}, nil
}

func formatRecipientWarning(pair FilePair, err error) string {
	source := pair.RecipientSource.ConfigPath
	if source == "" {
		source = pair.ConfigPath
	}
	if source == "" {
		source = "configuration"
	}
	return fmt.Sprintf("warning: could not resolve recipients for %s (%s): %v; continuing with encrypted metadata", pair.PlaintextPath, source, err)
}

func resolveFinalFormat(filePair FilePair) (string, ValueSource, error) {
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

func selectedBy(result SelectionResult, pair FilePair) string {
	if label, ok := result.SelectedBy[cleanAbsPath(pair.PlaintextPath)]; ok && label != "" {
		return label
	}
	if result.ConfigMode {
		return SelectedByCurrentDirectory
	}
	return "default"
}

func checkWriteConflicts(command string, filePairs []ResolvedFilePair) error {
	if !policyForCommand(command).writes {
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
