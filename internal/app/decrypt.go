package app

import (
	"fmt"

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
	Dir                   string
	Pattern               string
	OutputDir             string
	OutputSuffix          string
	Parallel              int
	Force                 bool
	UpdateProjectMetadata bool
}

func DecryptFiles(cfg *config.Config, req DecryptRequest) error {
	cliFormat, err := ValidateCLIFormatOverride(req.Format)
	if err != nil {
		return err
	}

	if req.Dir != "" {
		if cliFormat != "" {
			return fmt.Errorf("--format is only supported in single-file mode")
		}
		opts := batch.Options{
			InputDir:     req.Dir,
			Pattern:      req.Pattern,
			OutputDir:    req.OutputDir,
			OutputSuffix: req.OutputSuffix,
			KeyFile:      req.KeyFile,
			Parallel:     req.Parallel,
			Verbose:      req.Verbose,
			Force:        req.Force,
		}
		_, err := batch.Decrypt(opts)
		return err
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

	opts := batch.Options{
		FilePairs: configFilePairsToBatch(filePairs),
		KeyFile:   req.KeyFile,
		Parallel:  req.Parallel,
		Verbose:   req.Verbose,
		Force:     req.Force,
	}
	_, err = batch.Decrypt(opts)
	return err
}
