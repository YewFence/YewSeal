package app

import (
	"fmt"

	"github.com/YewFence/YewSeal/internal/agekey"
	"github.com/YewFence/YewSeal/internal/batch"
	"github.com/YewFence/YewSeal/internal/config"
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
	Dir                   string
	Pattern               string
	OutputDir             string
	OutputSuffix          string
	Parallel              int
	UpdateProjectMetadata bool
}

func EncryptFiles(cfg *config.Config, req EncryptRequest) error {
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
			PublicKey:    req.PublicKey,
			Parallel:     req.Parallel,
			Verbose:      req.Verbose,
		}
		_, err := batch.Encrypt(opts)
		return err
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

	opts := batch.Options{
		FilePairs: configFilePairsToBatch(filePairs),
		KeyFile:   req.KeyFile,
		PublicKey: resolvedPublicKey,
		Parallel:  req.Parallel,
		Verbose:   req.Verbose,
	}
	_, err = batch.Encrypt(opts)
	return err
}
