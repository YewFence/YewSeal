package app

import (
	"os"

	"github.com/YewFence/YewSeal/internal/agekey"
	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/task"
)

type PreflightResult struct {
	Selection       config.ResolvedSelection
	IdentityBundle  agekey.IdentityBundle
	MissingIdentity bool
}

func PreflightEncrypt(cfg *config.Config, req EncryptRequest) (PreflightResult, error) {
	selection, err := config.ResolveSelection(cfg, config.SelectionOptions{
		Command:             task.ModeEncrypt,
		Targets:             req.Targets,
		Output:              req.Output,
		OutputSet:           req.OutputSet,
		PrepareManagedFiles: req.UpdateGitignore || req.SyncSOPSConfig,
	})
	if err != nil {
		return PreflightResult{}, err
	}
	result := PreflightResult{Selection: selection}
	needsIdentity, err := encryptSelectionNeedsIdentity(selection)
	if err != nil {
		return PreflightResult{}, err
	}
	if req.Force || !needsIdentity {
		return result, nil
	}
	identityBundle, err := agekey.GetIdentityBundle(req.KeyFile)
	if err != nil {
		return PreflightResult{}, err
	}
	result.IdentityBundle = identityBundle
	result.MissingIdentity = len(identityBundle.Identities()) == 0
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
		Command:             task.ModeDecrypt,
		Targets:             req.Targets,
		PrepareManagedFiles: req.UpdateGitignore && req.OutputDir == "",
	})
	if err != nil {
		return PreflightResult{}, err
	}
	if err := prepareDecryptDelivery(cfg, &selection, req); err != nil {
		return PreflightResult{}, err
	}
	identityBundle, err := agekey.GetIdentityBundle(req.KeyFile)
	if err != nil {
		return PreflightResult{}, err
	}
	return PreflightResult{Selection: selection, IdentityBundle: identityBundle}, nil
}
