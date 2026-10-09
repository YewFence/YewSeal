package app

import (
	"errors"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/presentation"
	"github.com/YewFence/YewSeal/internal/project"
	"github.com/YewFence/YewSeal/internal/task"
)

type DecryptRequest struct {
	Presentation    *presentation.Output
	KeyFile         string
	OutputDir       string
	Inplace         bool
	Targets         []string
	Parallel        int
	Force           bool
	Strict          bool
	JSON            bool
	UpdateGitignore bool
}

func DecryptFiles(cfg *config.Config, req DecryptRequest) (err error) {
	out := presentation.OrDiscard(req.Presentation)
	out.SetDirectory(config.CurrentDir(cfg))
	defer func() { err = out.Finish(err) }()
	preflight, err := PreflightDecrypt(cfg, req)
	if err != nil {
		return err
	}
	out.IdentityBundle(preflight.IdentityBundle)
	out.Selection(preflight.Selection)

	if req.UpdateGitignore && req.OutputDir == "" {
		metadataPairs := config.ResolvedFilePairsToFilePairs(preflight.MetadataPairs)
		if err := project.UpdateGitignore(metadataPairs); err != nil {
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
		Delivery:       req.OutputDir != "",
		Inplace:        req.Inplace,
	}
	summary, err := task.Decrypt(opts)
	out.BatchSummary(summary, "decrypted")
	if req.JSON {
		err = errors.Join(err, out.BatchReportJSON("decrypt", summary))
	}
	return err
}
