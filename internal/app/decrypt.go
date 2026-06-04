package app

import (
	"fmt"
	"os"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/project"
	"github.com/YewFence/YewSeal/internal/task"
)

type DecryptRequest struct {
	KeyFile               string
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
			filePairs, err := scanFilePairsFromRequest(cfg, req.Target, task.ModeDecrypt, scanRequestOptions{
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
				Parallel:  req.Parallel,
				Verbose:   req.Verbose,
				Force:     req.Force,
			}
			_, err = task.Decrypt(opts)
			return err
		}
		target, err := ResolveEncryptedTargetWithOverrides(cfg, req.Target, "decrypt", cliFormat, req.Output)
		if err != nil {
			return err
		}
		_, err = task.Decrypt(task.Options{
			FilePairs: []task.FilePair{{
				PlaintextPath: target.PlaintextPath,
				EncryptedPath: target.EncryptedPath,
				Format:        target.FormatOverride,
			}},
			KeyFile:  req.KeyFile,
			Parallel: req.Parallel,
			Verbose:  req.Verbose,
			Force:    req.Force,
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

	scanPairs, err := configScanPairs(cfg, task.ModeDecrypt, scanRequestOptions{
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
		Parallel:  req.Parallel,
		Verbose:   req.Verbose,
		Force:     req.Force,
	}
	_, err = task.Decrypt(opts)
	return err
}
