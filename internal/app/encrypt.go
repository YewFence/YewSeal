package app

import (
	"errors"
	"fmt"

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
	JSON                  bool
	UpdateProjectMetadata bool
	SyncSOPSConfig        bool
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
	}

	out.Selection(preflight.Selection)
	opts := task.Options{
		FilePairs:      config.ResolvedFilePairsToTaskPairs(preflight.Selection.FilePairs),
		IdentityBundle: preflight.IdentityBundle,
		Parallel:       req.Parallel,
		OnComplete:     out.FileCompleted,
		Force:          req.Force,
	}
	summary, encryptErr := task.Encrypt(opts)
	out.BatchSummary(summary, "encrypted")
	var syncErr error
	if req.UpdateProjectMetadata && req.SyncSOPSConfig {
		resolvedDisplay := config.DisplayResolvedFilePairs(preflight.Selection.AllConfigPairs, config.CurrentDir(cfg))
		if err := project.SyncResolvedSopsYaml(resolvedDisplay); err != nil {
			syncErr = fmt.Errorf("failed to update .sops.yaml after encryption: %w\nCiphertext processing completed. Fix the synchronization error, or use --sync-sops-config=false when direct SOPS interoperability is not required", err)
		}
	}
	var reportErr error
	if req.JSON {
		reportErr = out.BatchReportJSON("encrypt", summary)
	}
	return errors.Join(encryptErr, syncErr, reportErr)
}
