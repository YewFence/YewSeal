package app

import (
	"fmt"
	"os"

	"github.com/YewFence/YewSeal/internal/batch"
	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/project"
	"github.com/YewFence/YewSeal/internal/seal"
)

type DecryptRequest struct {
	KeyFile               string
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
	Force                 bool
	UpdateProjectMetadata bool
}

func DecryptFiles(cfg *config.Config, req DecryptRequest) error {
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
			opts := scanBatchOptionsFromRequest(cfg, req.Target, batch.ModeDecrypt, req.KeyFile, "", req.Parallel, req.Verbose, req.Force, scanRequestOptions{
				Patterns:           req.Patterns,
				FormatRules:        req.FormatRules,
				UnknownAsBinary:    req.UnknownAsBinary,
				UnknownAsBinarySet: req.UnknownAsBinarySet,
			})
			_, err := batch.Decrypt(opts)
			return err
		}
		output, pathFormat, err := batch.PlaintextPathForEncrypted(req.Target, cliFormat)
		if err != nil {
			return err
		}
		return seal.Decrypt(seal.DecryptOptions{
			InputFile:      req.Target,
			OutputFile:     output,
			KeyFile:        req.KeyFile,
			FormatOverride: pathFormat,
			Verbose:        req.Verbose,
			Force:          req.Force,
		})
	}

	if req.InputSet || req.OutputSet {
		filePair := cfg.GetPrimaryFilePair()
		if req.InputSet {
			filePair.EncryptedPath = req.Input
		}
		if req.OutputSet {
			filePair.PlaintextPath = req.Output
		}
		formatOverride, err := ResolveFormatOverride(cliFormat, filePair)
		if err != nil {
			return err
		}
		return seal.Decrypt(seal.DecryptOptions{
			InputFile:      filePair.EncryptedPath,
			OutputFile:     filePair.PlaintextPath,
			KeyFile:        req.KeyFile,
			FormatOverride: formatOverride,
			Verbose:        req.Verbose,
			Force:          req.Force,
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

	scanPairs, err := configScanPairs(cfg, batch.ModeDecrypt, scanRequestOptions{
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
		Parallel:  req.Parallel,
		Verbose:   req.Verbose,
		Force:     req.Force,
	}
	_, err = batch.Decrypt(opts)
	return err
}
