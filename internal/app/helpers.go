package app

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/YewFence/YewSeal/internal/batch"
	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/fileformat"
	"github.com/YewFence/YewSeal/internal/seal"
	"github.com/urfave/cli/v2"
)

func ValidateCLIFormatOverride(format string) (string, error) {
	if strings.TrimSpace(format) == "" {
		return "", nil
	}

	parsed, ok := seal.NormalizeFormatOverride(format)
	if !ok {
		return "", fmt.Errorf("unsupported format %q (supported: toml, yaml, json, env, ini, binary)", format)
	}

	return parsed, nil
}

func ResolveFormatOverride(cliFormat string, filePair config.FilePair) (string, error) {
	validatedFormat, err := ValidateCLIFormatOverride(cliFormat)
	if err != nil {
		return "", err
	}
	if validatedFormat != "" {
		return validatedFormat, nil
	}
	return filePair.Format, nil
}

func configFilePairsToBatch(filePairs []config.FilePair) []batch.FilePair {
	pairs := make([]batch.FilePair, 0, len(filePairs))
	for _, filePair := range filePairs {
		pairs = append(pairs, batch.FilePair{
			PlaintextPath: filePair.PlaintextPath,
			EncryptedPath: filePair.EncryptedPath,
			Format:        filePair.Format,
		})
	}
	return pairs
}

type scanRequestOptions struct {
	Patterns           []string
	FormatRules        []string
	UnknownAsBinary    bool
	UnknownAsBinarySet bool
}

func scanBatchOptionsFromRequest(
	cfg *config.Config,
	root,
	mode,
	keyFile,
	publicKey string,
	parallel int,
	verbose,
	force bool,
	req scanRequestOptions,
) batch.Options {
	scan := cfg.GetScan()
	patterns := scan.Patterns
	if len(req.Patterns) > 0 {
		patterns = append([]string(nil), req.Patterns...)
	}
	formatRules := append([]string(nil), scan.FormatRules...)
	formatRules = append(formatRules, req.FormatRules...)
	unknownAsBinary := scan.UnknownAsBinary
	if req.UnknownAsBinarySet {
		unknownAsBinary = req.UnknownAsBinary
	}

	return batch.Options{
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

func configScanPairs(cfg *config.Config, mode string, req scanRequestOptions) ([]batch.FilePair, error) {
	opts := scanBatchOptionsFromRequest(cfg, ".", mode, "", "", 1, false, false, req)
	if len(opts.Patterns) == 0 && len(opts.FormatRules) == 0 && !opts.UnknownAsBinary {
		return nil, nil
	}
	return batch.BuildProjectScanFilePairs(batch.ScanOptions{
		Root:            opts.InputDir,
		Patterns:        opts.Patterns,
		FormatRules:     opts.FormatRules,
		UnknownAsBinary: opts.UnknownAsBinary,
		Mode:            mode,
	})
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
	target = strings.TrimSpace(target)
	if target == "" {
		return ResolvedEncryptedTarget{}, fmt.Errorf("%s requires exactly one target", commandName)
	}

	for _, filePair := range cfg.GetFiles() {
		if target != filePair.PlaintextPath && target != filePair.EncryptedPath {
			continue
		}
		formatOverride, err := ResolveFormatOverride("", filePair)
		if err != nil {
			return ResolvedEncryptedTarget{}, err
		}
		format, err := effectiveFormat(filePair.PlaintextPath, formatOverride)
		if err != nil {
			return ResolvedEncryptedTarget{}, err
		}
		return ResolvedEncryptedTarget{
			PlaintextPath:  filePair.PlaintextPath,
			EncryptedPath:  filePair.EncryptedPath,
			FormatOverride: formatOverride,
			Format:         format,
		}, nil
	}

	plaintextPath, format, err := fileformat.PlaintextPathForEncrypted(target, "")
	if err != nil {
		return ResolvedEncryptedTarget{}, err
	}

	return ResolvedEncryptedTarget{
		PlaintextPath:  plaintextPath,
		EncryptedPath:  target,
		FormatOverride: format,
		Format:         format,
	}, nil
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
	filePair, err := ResolveSingleTargetFilePair(cfg, target, "view")
	if err != nil {
		return err
	}

	formatOverride, err := ResolveFormatOverride(cliFormat, filePair)
	if err != nil {
		return err
	}

	plainData, err := seal.DecryptToBytes(seal.DecryptBytesOptions{
		InputFile:      filePair.EncryptedPath,
		OutputFile:     filePair.PlaintextPath,
		KeyFile:        keyFile,
		FormatOverride: formatOverride,
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

func ResolveSyncKeyFile(c *cli.Context, cfg *config.Config) string {
	if c.IsSet("key-file") {
		return cfg.GetKeyFile(c.String("key-file"))
	}
	return cfg.GetKeyFile("")
}

func ResolveSyncProvider(c *cli.Context, cfg *config.Config) string {
	if c.IsSet("provider") {
		return cfg.GetSyncProvider(c.String("provider"))
	}
	return cfg.GetSyncProvider("")
}

func ResolveSyncSecretName(c *cli.Context, cfg *config.Config) string {
	if c.IsSet("name") {
		return cfg.GetSyncSecretName(c.String("name"))
	}
	return cfg.GetSyncSecretName("")
}

func ResolveSyncProjectID(c *cli.Context, cfg *config.Config) string {
	if c.IsSet("project-id") {
		return cfg.GetSyncProjectID(c.String("project-id"))
	}
	return cfg.GetSyncProjectID("")
}

func ResolveSyncPath(c *cli.Context, cfg *config.Config) string {
	if c.IsSet("path") {
		return cfg.GetSyncPath(c.String("path"))
	}
	return cfg.GetSyncPath("")
}

func ResolveSyncEnvironment(c *cli.Context, cfg *config.Config) string {
	if c.IsSet("env") {
		return cfg.GetSyncEnvironment(c.String("env"))
	}
	return cfg.GetSyncEnvironment("")
}
