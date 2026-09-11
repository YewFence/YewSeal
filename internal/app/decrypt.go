package app

import (
	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/presentation"
	"github.com/YewFence/YewSeal/internal/project"
	"github.com/YewFence/YewSeal/internal/task"
)

type DecryptRequest struct {
	Presentation          *presentation.Output
	KeyFile               string
	Output                string
	OutputSet             bool
	Targets               []string
	Parallel              int
	Force                 bool
	Strict                bool
	UpdateProjectMetadata bool
}

func DecryptFiles(cfg *config.Config, req DecryptRequest) (err error) {
	out := presentation.OrDiscard(req.Presentation)
	out.SetDirectory(config.CurrentDir(cfg))
	defer func() { err = out.Finish(err) }()
	preflight, err := PreflightDecrypt(cfg, req)
	if err != nil {
		return err
	}
	out.Selection(preflight.Selection)

	if req.UpdateProjectMetadata {
		metadataPairs := config.ResolvedFilePairsToFilePairs(preflight.MetadataPairs)
		if err := project.UpdateGitignore(config.DisplayFilePairs(metadataPairs, config.CurrentDir(cfg))); err != nil {
			return err
		}
	}

	opts := task.Options{
		FilePairs:      config.ResolvedFilePairsToTaskPairs(preflight.Selection.FilePairs),
		IdentityBundle: preflight.IdentityBundle,
		Parallel:       req.Parallel,
		OnComplete:     out.FileCompleted,
		Force:          req.Force,
		Strict:         req.Strict,
	}
	summary, err := task.Decrypt(opts)
	out.BatchSummary(summary, "decrypted")
	return err
}
