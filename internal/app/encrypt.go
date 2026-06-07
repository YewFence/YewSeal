package app

import (
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
	preflight, err := PreflightEncrypt(cfg, req)
	if err != nil {
		return err
	}

	if req.UpdateProjectMetadata {
		metadataDisplayPairs := config.DisplayFilePairs(config.ResolvedFilePairsToFilePairs(preflight.MetadataPairs), config.CurrentDir(cfg))
		if err := project.UpdateGitignore(metadataDisplayPairs); err != nil {
			return err
		}
		if err := project.SyncSopsYaml(metadataDisplayPairs, preflight.PublicKey); err != nil {
			return err
		}
	}

	printResolvedSelection(req.Verbose, cfg, preflight.Selection)
	opts := task.Options{
		FilePairs: config.ResolvedFilePairsToTaskPairs(preflight.Selection.FilePairs),
		KeyFile:   req.KeyFile,
		PublicKey: preflight.PublicKey,
		Parallel:  req.Parallel,
		Verbose:   req.Verbose,
	}
	_, err = task.Encrypt(opts)
	return err
}
