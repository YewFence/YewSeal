package app

import (
	"fmt"
	"os"

	"github.com/YewFence/YewSeal/internal/agekey"
	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/project"
	"github.com/YewFence/YewSeal/internal/task"
)

type EncryptRequest struct {
	KeyFile               string
	PublicKey             string
	Verbose               bool
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
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to stat %s: %w", req.Target, err)
		}
		if err == nil && info.IsDir() {
			if cliFormat != "" {
				return fmt.Errorf("--format is only supported in single-file mode")
			}
			if req.OutputSet {
				return fmt.Errorf("--output is only supported when the path target is a file")
			}
			filePairs, err := scanFilePairsFromRequest(cfg, req.Target, task.ModeEncrypt, scanRequestOptions{
				Patterns:           req.Patterns,
				FormatRules:        req.FormatRules,
				UnknownAsBinary:    req.UnknownAsBinary,
				UnknownAsBinarySet: req.UnknownAsBinarySet,
			})
			if err != nil {
				return err
			}
			opts := task.Options{
				FilePairs: filePairs,
				KeyFile:   req.KeyFile,
				PublicKey: req.PublicKey,
				Parallel:  req.Parallel,
				Verbose:   req.Verbose,
			}
			_, err = task.Encrypt(opts)
			return err
		}
		filePair, err := ResolvePlaintextTarget(cfg, req.Target, cliFormat, req.Output)
		if err != nil {
			return err
		}
		_, err = task.Encrypt(task.Options{
			FilePairs: configFilePairsToTasks([]config.FilePair{filePair}),
			KeyFile:   req.KeyFile,
			PublicKey: req.PublicKey,
			Parallel:  req.Parallel,
			Verbose:   req.Verbose,
		})
		return err
	}

	if req.OutputSet {
		return fmt.Errorf("--output is only supported when the path target is a file")
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

	scanPairs, err := configScanPairs(cfg, task.ModeEncrypt, scanRequestOptions{
		Patterns:           req.Patterns,
		FormatRules:        req.FormatRules,
		UnknownAsBinary:    req.UnknownAsBinary,
		UnknownAsBinarySet: req.UnknownAsBinarySet,
	})
	if err != nil {
		return err
	}
	opts := task.Options{
		FilePairs: append(configFilePairsToTasks(filePairs), scanPairs...),
		KeyFile:   req.KeyFile,
		PublicKey: resolvedPublicKey,
		Parallel:  req.Parallel,
		Verbose:   req.Verbose,
	}
	_, err = task.Encrypt(opts)
	return err
}
