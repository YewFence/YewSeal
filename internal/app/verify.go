package app

import (
	"strings"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/errx"
	"github.com/YewFence/YewSeal/internal/task"
	"github.com/YewFence/YewSeal/internal/verify"
)

type VerifyRequest struct {
	Targets        []string
	KeyFile        string
	Decrypt        bool
	NoDecrypt      bool
	SyncSOPSConfig bool
}

// Verify resolves the selection with plan's directionless strict semantics and
// runs every verification layer. Errors mean verify could not run (exit 2);
// project problems are findings in the report.
func Verify(cfg *config.Config, req VerifyRequest) (*verify.Report, error) {
	selection, err := config.ResolveSelection(cfg, config.SelectionOptions{
		Command: task.ModeVerify,
		Targets: req.Targets,
	})
	if err != nil {
		return nil, err
	}
	mode := verify.DecryptAuto
	switch {
	case req.Decrypt:
		mode = verify.DecryptRequired
	case req.NoDecrypt:
		mode = verify.DecryptDisabled
	}
	aliases := make(map[string]string, len(cfg.Recipients.Registry))
	for alias, recipient := range cfg.Recipients.Registry {
		aliases[strings.TrimSpace(recipient)] = alias
	}
	report, err := verify.Check(selection, config.CurrentDir(cfg), verify.Options{
		KeyFile:          req.KeyFile,
		DecryptMode:      mode,
		CheckSOPSConfig:  req.SyncSOPSConfig,
		RecipientAliases: aliases,
	})
	return report, errx.Usage(err)
}
