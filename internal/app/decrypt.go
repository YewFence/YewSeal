package app

import (
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
	preflight, err := PreflightDecrypt(cfg, req)
	if err != nil {
		return err
	}

	if req.UpdateProjectMetadata {
		metadataPairs := config.ResolvedFilePairsToFilePairs(preflight.MetadataPairs)
		if err := project.UpdateGitignore(config.DisplayFilePairs(metadataPairs, config.CurrentDir(cfg))); err != nil {
			return err
		}
	}

	printResolvedSelection(req.Verbose, cfg, preflight.Selection)
	opts := task.Options{
		FilePairs: config.ResolvedFilePairsToTaskPairs(preflight.Selection.FilePairs),
		KeyFile:   req.KeyFile,
		Parallel:  req.Parallel,
		Verbose:   req.Verbose,
		Force:     req.Force,
	}
	_, err = task.Decrypt(opts)
	return err
}
