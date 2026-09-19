package app

import (
	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/presentation"
	"github.com/YewFence/YewSeal/internal/project"
	"github.com/YewFence/YewSeal/internal/task"
)

type EncryptRequest struct {
	Presentation          *presentation.Output
	KeyFile               string
	Output                string
	OutputSet             bool
	Targets               []string
	Parallel              int
	Force                 bool
	UpdateProjectMetadata bool
}

func EncryptFiles(cfg *config.Config, req EncryptRequest) (err error) {
	out := presentation.OrDiscard(req.Presentation)
	out.SetDirectory(config.CurrentDir(cfg))
	defer func() { err = out.Finish(err) }()
	preflight, err := PreflightEncrypt(cfg, req)
	if err != nil {
		return err
	}
	if preflight.MissingIdentity {
		out.Warning("no Age identity found; existing ciphertext will be replaced instead of updated incrementally")
	}
	out.IdentityBundle(preflight.IdentityBundle)

	if req.UpdateProjectMetadata {
		metadataDisplayPairs := config.DisplayFilePairs(config.ResolvedFilePairsToFilePairs(preflight.MetadataPairs), config.CurrentDir(cfg))
		if err := project.UpdateGitignore(metadataDisplayPairs); err != nil {
			return err
		}
		metadataResolvedDisplay := config.DisplayResolvedFilePairs(preflight.MetadataPairs, config.CurrentDir(cfg))
		if err := project.SyncResolvedSopsYaml(metadataResolvedDisplay); err != nil {
			return err
		}
	}

	out.Selection(preflight.Selection)
	opts := task.Options{
		FilePairs:      config.ResolvedFilePairsToTaskPairs(preflight.Selection.FilePairs),
		IdentityBundle: preflight.IdentityBundle,
		Parallel:       req.Parallel,
		OnComplete:     out.FileCompleted,
		Force:          req.Force,
	}
	summary, err := task.Encrypt(opts)
	out.BatchSummary(summary, "encrypted")
	return err
}
