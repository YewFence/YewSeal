package app

import (
	"fmt"
	"os"

	"github.com/YewFence/YewSeal/internal/agekey"
	"github.com/YewFence/YewSeal/internal/batch"
	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/fileformat"
	"github.com/YewFence/YewSeal/internal/project"
	"github.com/YewFence/YewSeal/internal/seal"
)

type EncryptRequest struct {
	KeyFile               string
	PublicKey             string
	Verbose               bool
	Input                 string
	InputSet              bool
	Output                string
	OutputSet             bool
	Format                string
	Target                string
	Patterns              []string
	FormatRules           []string
	UnknownAsBinary       bool
	UnknownAsBinarySet    bool
	Parallel              int
	UpdateProjectMetadata bool
}

func EncryptFiles(cfg *config.Config, req EncryptRequest) error {
	cliFormat, err := ValidateCLIFormatOverride(req.Format)
	if err != nil {
		return err
	}

	if req.Target != "" {
		info, err := os.Stat(req.Target)
		if err != nil {
			return fmt.Errorf("failed to stat %s: %w", req.Target, err)
		}
		if info.IsDir() {
			if cliFormat != "" {
				return fmt.Errorf("--format is only supported in single-file mode")
			}
			opts := scanBatchOptionsFromRequest(cfg, req.Target, batch.ModeEncrypt, req.KeyFile, req.PublicKey, req.Parallel, req.Verbose, false, scanRequestOptions{
				Patterns:           req.Patterns,
				FormatRules:        req.FormatRules,
				UnknownAsBinary:    req.UnknownAsBinary,
				UnknownAsBinarySet: req.UnknownAsBinarySet,
			})
			_, err := batch.Encrypt(opts)
			return err
		}
		output, err := fileformat.EncryptPathForPlaintext(req.Target, cliFormat)
		if err != nil {
			return err
		}
		return seal.Encrypt(seal.EncryptOptions{
			InputFile:      req.Target,
			OutputFile:     output,
			KeyFile:        req.KeyFile,
			PublicKey:      req.PublicKey,
			FormatOverride: cliFormat,
			Verbose:        req.Verbose,
		})
	}

	if req.InputSet || req.OutputSet {
		filePair := cfg.GetPrimaryFilePair()
		if req.InputSet {
			filePair.PlaintextPath = req.Input
		}
		if req.OutputSet {
			filePair.EncryptedPath = req.Output
		}
		formatOverride, err := ResolveFormatOverride(cliFormat, filePair)
		if err != nil {
			return err
		}
		return seal.Encrypt(seal.EncryptOptions{
			InputFile:      filePair.PlaintextPath,
			OutputFile:     filePair.EncryptedPath,
			KeyFile:        req.KeyFile,
			PublicKey:      req.PublicKey,
			FormatOverride: formatOverride,
			Verbose:        req.Verbose,
		})
	}

	if cliFormat != "" {
		return fmt.Errorf("--format is only supported in single-file mode")
	}

	filePairs := cfg.GetFiles()
	if req.UpdateProjectMetadata {
		if err := project.UpdateGitignore(filePairs); err != nil {
			return err
		}
	}

	resolvedPublicKey, err := agekey.GetPublicKey(req.PublicKey, req.KeyFile, req.Verbose)
	if err != nil {
		return err
	}
	if req.UpdateProjectMetadata {
		if err := project.SyncSopsYaml(filePairs, resolvedPublicKey); err != nil {
			return err
		}
	}

	scanPairs, err := configScanPairs(cfg, batch.ModeEncrypt, scanRequestOptions{
		Patterns:           req.Patterns,
		FormatRules:        req.FormatRules,
		UnknownAsBinary:    req.UnknownAsBinary,
		UnknownAsBinarySet: req.UnknownAsBinarySet,
	})
	if err != nil {
		return err
	}
	opts := batch.Options{
		FilePairs: append(configFilePairsToBatch(filePairs), scanPairs...),
		KeyFile:   req.KeyFile,
		PublicKey: resolvedPublicKey,
		Parallel:  req.Parallel,
		Verbose:   req.Verbose,
	}
	_, err = batch.Encrypt(opts)
	return err
}
