package app

import (
	"errors"
	"os"

	"github.com/YewFence/YewSeal/internal/agekey"
	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/errx"
	"github.com/YewFence/YewSeal/internal/task"
)

type PreflightResult struct {
	Selection       config.ResolvedSelection
	IdentityBundle  agekey.IdentityBundle
	MissingIdentity bool
	MetadataPairs   []config.ResolvedFilePair
}

func PreflightEncrypt(cfg *config.Config, req EncryptRequest) (PreflightResult, error) {
	selection, err := config.ResolveSelection(cfg, config.SelectionOptions{
		Command:   task.ModeEncrypt,
		Targets:   req.Targets,
		Output:    req.Output,
		OutputSet: req.OutputSet,
	})
	if err != nil {
		return PreflightResult{}, err
	}
	result := PreflightResult{Selection: selection, MetadataPairs: metadataPairsForSelection(selection)}
	needsIdentity, err := encryptSelectionNeedsIdentity(selection)
	if err != nil {
		return PreflightResult{}, err
	}
	if req.Force || !needsIdentity {
		return result, nil
	}
	identityBundle, err := agekey.GetIdentityBundle(req.KeyFile)
	if err != nil {
		var missing *errx.AgeKeyNotFoundError
		if errors.As(err, &missing) {
			result.MissingIdentity = true
			return result, nil
		}
		return PreflightResult{}, err
	}
	result.IdentityBundle = identityBundle
	return result, nil
}

func encryptSelectionNeedsIdentity(selection config.ResolvedSelection) (bool, error) {
	for _, pair := range selection.FilePairs {
		plaintextExists, err := pathExists(pair.PlaintextPath)
		if err != nil {
			return false, err
		}
		encryptedExists, err := pathExists(pair.EncryptedPath)
		if err != nil {
			return false, err
		}
		if plaintextExists && encryptedExists {
			return true, nil
		}
	}
	return false, nil
}

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func PreflightDecrypt(cfg *config.Config, req DecryptRequest) (PreflightResult, error) {
	selection, err := config.ResolveSelection(cfg, config.SelectionOptions{
		Command:   task.ModeDecrypt,
		Targets:   req.Targets,
		Output:    req.Output,
		OutputSet: req.OutputSet,
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
