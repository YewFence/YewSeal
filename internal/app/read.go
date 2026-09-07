package app

import (
	"github.com/YewFence/YewSeal/internal/agekey"
	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/presentation"
)

func prepareRead(out *presentation.Output, cfg *config.Config, opts config.SelectionOptions, keyFile string) (config.ResolvedSelection, agekey.IdentityBundle, error) {
	selection, err := config.ResolveSelection(cfg, opts)
	if err != nil {
		return selection, agekey.IdentityBundle{}, err
	}
	bundle, err := agekey.GetIdentityBundle(keyFile)
	if err != nil {
		return selection, bundle, err
	}
	out.SetDirectory(config.CurrentDir(cfg))
	out.Selection(selection)
	return selection, bundle, nil
}
