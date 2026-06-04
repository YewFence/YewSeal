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
	result, err := config.SelectFilePairs(cfg, config.SelectionOptions{
		Command:              task.ModeDecrypt,
		Target:               req.Target,
		Output:               req.Output,
		OutputSet:            req.OutputSet,
		Format:               req.Format,
		Patterns:             req.Patterns,
		FormatRules:          req.FormatRules,
		UnknownAsBinary:      req.UnknownAsBinary,
		UnknownAsBinarySet:   req.UnknownAsBinarySet,
		AllowEmptyTarget:     true,
		UseConfiguredDefault: true,
	})
	if err != nil {
		return err
	}

	if req.UpdateProjectMetadata {
		metadataPairs := result.FilePairs
		if result.ConfigMode && len(result.AllConfigPairs) > 0 {
			metadataPairs = result.AllConfigPairs
		}
		if err := project.UpdateGitignore(config.DisplayFilePairs(metadataPairs, config.CurrentDir(cfg))); err != nil {
			return err
		}
	}

	filePairs, err := config.ValidateFilePairs(result.FilePairs)
	if err != nil {
		return err
	}
	config.PrintSelection(req.Verbose, cfg, result)
	opts := task.Options{
		FilePairs: configFilePairsToTasks(filePairs),
		KeyFile:   req.KeyFile,
		Parallel:  req.Parallel,
		Verbose:   req.Verbose,
		Force:     req.Force,
	}
	_, err = task.Decrypt(opts)
	return err
}
