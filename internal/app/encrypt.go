package app

import (
	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/project"
	"github.com/YewFence/YewSeal/internal/task"
)

type EncryptRequest struct {
	KeyFile               string
	PublicKey             string // Deprecated compatibility field; use configured recipients.
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
	preflight, err := PreflightEncrypt(cfg, req)
	if err != nil {
		return err
	}

	if req.UpdateProjectMetadata {
		metadataDisplayPairs := config.DisplayFilePairs(config.ResolvedFilePairsToFilePairs(preflight.MetadataPairs), config.CurrentDir(cfg))
		if err := project.UpdateGitignore(metadataDisplayPairs); err != nil {
			return err
		}
		if err := project.SyncResolvedSopsYaml(preflight.MetadataPairs); err != nil {
			return err
		}
	}

	printResolvedSelection(req.Verbose, cfg, preflight.Selection)
	opts := task.Options{
		FilePairs: config.ResolvedFilePairsToTaskPairs(preflight.Selection.FilePairs),
		Parallel:  req.Parallel,
		PublicKey: req.PublicKey,
		Verbose:   req.Verbose,
	}
	_, err = task.Encrypt(opts)
	return err
}
