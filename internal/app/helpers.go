package app

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/fileformat"
	"github.com/YewFence/YewSeal/internal/seal"
	"github.com/YewFence/YewSeal/internal/task"
)

func ValidateCLIFormatOverride(format string) (string, error) {
	return config.ValidateFormatOverride(format)
}

func ResolveFormatOverride(cliFormat string, filePair config.FilePair) (string, error) {
	return config.ResolveFormatOverride(cliFormat, filePair)
}

func ResolvePlaintextTarget(cfg *config.Config, target, cliFormat, output string) (config.FilePair, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return config.FilePair{}, fmt.Errorf("encrypt requires exactly one target")
	}

	for _, filePair := range cfg.GetFiles() {
		if !sameCleanPath(target, filePair.PlaintextPath) && !sameCleanPath(target, filePair.EncryptedPath) {
			continue
		}
		formatOverride, err := ResolveFormatOverride(cliFormat, filePair)
		if err != nil {
			return config.FilePair{}, err
		}
		if output != "" {
			filePair.EncryptedPath = output
		}
		filePair.Format = formatOverride
		return filePair, nil
	}

	formatOverride, err := ValidateCLIFormatOverride(cliFormat)
	if err != nil {
		return config.FilePair{}, err
	}
	encryptedPath := output
	if encryptedPath == "" {
		encryptedPath, err = fileformat.EncryptPathForPlaintext(target, formatOverride)
		if err != nil {
			return config.FilePair{}, err
		}
	}
	return config.FilePair{
		PlaintextPath: target,
		EncryptedPath: encryptedPath,
		Format:        formatOverride,
	}, nil
}

type groupRequestOptions struct {
	Patterns           []string
	FormatRules        []string
	UnknownAsBinary    bool
	UnknownAsBinarySet bool
}

func groupTaskOptionsFromRequest(
	group config.GroupConfig,
	root,
	mode,
	keyFile,
	publicKey string,
	parallel int,
	verbose,
	force bool,
	req groupRequestOptions,
) task.Options {
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

	return task.Options{
		InputDir:        root,
		Patterns:        patterns,
		FormatRules:     formatRules,
		UnknownAsBinary: unknownAsBinary,
		KeyFile:         keyFile,
		PublicKey:       publicKey,
		Parallel:        parallel,
		Verbose:         verbose,
		Force:           force,
	}
}

func configGroupPairs(cfg *config.Config, mode string, req groupRequestOptions) ([]task.FilePair, error) {
	groups := cfg.GetGroups()
	if len(groups) == 0 {
		if !hasGroupRequestOptions(req) {
			return nil, nil
		}
		groups = []config.GroupConfig{{}}
	}

	pairs := make([]task.FilePair, 0)
	for _, group := range groups {
		opts := groupTaskOptionsFromRequest(group, ".", mode, "", "", 1, false, false, req)
		groupPairs, err := task.BuildProjectGroupFilePairs(task.GroupOptions{
			Root:            opts.InputDir,
			Patterns:        opts.Patterns,
			FormatRules:     opts.FormatRules,
			UnknownAsBinary: opts.UnknownAsBinary,
			Mode:            mode,
		})
		if err != nil {
			return nil, err
		}
		pairs = append(pairs, groupPairs...)
	}
	return pairs, nil
}

func hasGroupRequestOptions(req groupRequestOptions) bool {
	return len(req.Patterns) > 0 || len(req.FormatRules) > 0 || req.UnknownAsBinarySet
}

func ResolveTargetFilePairs(cfg *config.Config, target string) ([]config.FilePair, error) {
	target = strings.TrimSpace(target)
	files := cfg.GetFiles()
	if target == "" {
		return files, nil
	}

	for _, filePair := range files {
		if target == filePair.PlaintextPath || target == filePair.EncryptedPath {
			return []config.FilePair{filePair}, nil
		}
	}

	return nil, fmt.Errorf("target %s is not configured as a plaintext or encrypted file", target)
}

func ResolveSingleTargetFilePair(cfg *config.Config, target, commandName string) (config.FilePair, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return config.FilePair{}, fmt.Errorf("%s requires exactly one target", commandName)
	}

	filePairs, err := ResolveTargetFilePairs(cfg, target)
	if err != nil {
		return config.FilePair{}, err
	}
	if len(filePairs) != 1 {
		return config.FilePair{}, fmt.Errorf("%s requires exactly one target", commandName)
	}
	return filePairs[0], nil
}

type ResolvedEncryptedTarget struct {
	PlaintextPath  string
	EncryptedPath  string
	FormatOverride string
	Format         string
}

func ResolveEncryptedTarget(cfg *config.Config, target, commandName string) (ResolvedEncryptedTarget, error) {
	return ResolveEncryptedTargetWithOverrides(cfg, target, commandName, "", "")
}

func ResolveEncryptedTargetWithOverrides(cfg *config.Config, target, commandName, cliFormat, output string) (ResolvedEncryptedTarget, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return ResolvedEncryptedTarget{}, fmt.Errorf("%s requires exactly one target", commandName)
	}

	for _, filePair := range cfg.GetFiles() {
		if !sameCleanPath(target, filePair.PlaintextPath) && !sameCleanPath(target, filePair.EncryptedPath) {
			continue
		}
		formatOverride, err := ResolveFormatOverride(cliFormat, filePair)
		if err != nil {
			return ResolvedEncryptedTarget{}, err
		}
		formatPath := filePair.PlaintextPath
		format, err := effectiveFormat(formatPath, formatOverride)
		if err != nil {
			return ResolvedEncryptedTarget{}, err
		}
		if formatOverride == "" {
			formatOverride = format
		}
		if output != "" {
			filePair.PlaintextPath = output
		}
		return ResolvedEncryptedTarget{
			PlaintextPath:  filePair.PlaintextPath,
			EncryptedPath:  filePair.EncryptedPath,
			FormatOverride: formatOverride,
			Format:         format,
		}, nil
	}

	formatOverride, err := ValidateCLIFormatOverride(cliFormat)
	if err != nil {
		return ResolvedEncryptedTarget{}, err
	}
	plaintextPath := output
	pathFormat := ""
	inferredPlaintextPath, pathFormat, err := fileformat.PlaintextPathForEncrypted(target, formatOverride)
	if err != nil {
		return ResolvedEncryptedTarget{}, err
	}
	if formatOverride == "" {
		formatOverride = pathFormat
	}
	if plaintextPath == "" {
		plaintextPath = inferredPlaintextPath
	}
	format, err := effectiveFormat(plaintextPath, formatOverride)
	if err != nil {
		return ResolvedEncryptedTarget{}, err
	}

	return ResolvedEncryptedTarget{
		PlaintextPath:  plaintextPath,
		EncryptedPath:  target,
		FormatOverride: formatOverride,
		Format:         format,
	}, nil
}

func sameCleanPath(a, b string) bool {
	return appCleanAbsPath(a) == appCleanAbsPath(b)
}

func appCleanAbsPath(path string) string {
	abs, err := filepath.Abs(path)
	if err == nil {
		return filepath.Clean(abs)
	}
	return filepath.Clean(path)
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

func WriteViewedTarget(w io.Writer, cfg *config.Config, target, keyFile, cliFormat string, verbose bool) error {
	result, err := config.SelectFilePairs(cfg, config.SelectionOptions{
		Command:             task.ModeDecrypt,
		Target:              target,
		Format:              cliFormat,
		RequireSingleTarget: true,
	})
	if err != nil {
		return err
	}
	filePairs, err := config.ValidateFilePairs(result.FilePairs)
	if err != nil {
		return err
	}
	filePair := filePairs[0]
	config.PrintSelection(verbose, cfg, result)

	plainData, err := seal.DecryptToBytes(seal.DecryptBytesOptions{
		InputFile:      filePair.EncryptedPath,
		OutputFile:     filePair.PlaintextPath,
		KeyFile:        keyFile,
		FormatOverride: filePair.Format,
		Verbose:        verbose,
		Output:         os.Stderr,
		Warnings:       os.Stderr,
	})
	if err != nil {
		return err
	}

	if _, err := w.Write(plainData); err != nil {
		return err
	}
	return nil
}
