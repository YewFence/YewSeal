package app

import (
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
	result, err := config.SelectFilePairs(cfg, config.SelectionOptions{
		Command:              task.ModeEncrypt,
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

	publicKeyCandidate := req.PublicKey
	if publicKeyCandidate == "" {
		publicKeyCandidate = cfg.GetPublicKey()
	}
	resolvedPublicKey, err := agekey.GetPublicKey(publicKeyCandidate, req.KeyFile, req.Verbose)
	if err != nil {
		return err
	}

	if req.UpdateProjectMetadata {
		metadataPairs := result.FilePairs
		if result.ConfigMode && len(result.AllConfigPairs) > 0 {
			metadataPairs = result.AllConfigPairs
		}
		metadataDisplayPairs := config.DisplayFilePairs(metadataPairs, config.CurrentDir(cfg))
		if err := project.UpdateGitignore(metadataDisplayPairs); err != nil {
			return err
		}
		if err := project.SyncSopsYaml(metadataDisplayPairs, resolvedPublicKey); err != nil {
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
		PublicKey: resolvedPublicKey,
		Parallel:  req.Parallel,
		Verbose:   req.Verbose,
	}
	_, err = task.Encrypt(opts)
	return err
}
