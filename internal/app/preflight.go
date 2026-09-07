package app

import (
	"github.com/YewFence/YewSeal/internal/agekey"
	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/task"
)

type PreflightResult struct {
	Selection      config.ResolvedSelection
	IdentityBundle agekey.IdentityBundle
	MetadataPairs  []config.ResolvedFilePair
}

func PreflightEncrypt(cfg *config.Config, req EncryptRequest) (PreflightResult, error) {
	selection, err := config.ResolveSelection(cfg, config.SelectionOptions{
		Command:   task.ModeEncrypt,
		Target:    req.Target,
		Output:    req.Output,
		OutputSet: req.OutputSet,
		Patterns:  req.Patterns,
	})
	if err != nil {
		return PreflightResult{}, err
	}
	return PreflightResult{Selection: selection, MetadataPairs: metadataPairsForSelection(selection)}, nil
}

func PreflightDecrypt(cfg *config.Config, req DecryptRequest) (PreflightResult, error) {
	selection, err := config.ResolveSelection(cfg, config.SelectionOptions{
		Command:   task.ModeDecrypt,
		Target:    req.Target,
		Output:    req.Output,
		OutputSet: req.OutputSet,
		Patterns:  req.Patterns,
	})
	if err != nil {
		return PreflightResult{}, err
	}
	identityBundle, err := agekey.GetIdentityBundle(req.KeyFile)
	if err != nil {
		return PreflightResult{}, err
	}
	return PreflightResult{Selection: selection, IdentityBundle: identityBundle, MetadataPairs: metadataPairsForSelection(selection)}, nil
}

func metadataPairsForSelection(selection config.ResolvedSelection) []config.ResolvedFilePair {
	if selection.ConfigMode && len(selection.AllConfigPairs) > 0 {
		return selection.AllConfigPairs
	}
	return selection.FilePairs
}
