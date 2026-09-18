package app

import (
	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/presentation"
	"github.com/YewFence/YewSeal/internal/seal"
	"github.com/YewFence/YewSeal/internal/task"
	"io"
)

func ValidateCLIFormatOverride(format string) (string, error) {
	return config.ValidateFormatOverride(format)
}

type ViewRequest struct {
	Target   string
	KeyFile  string
	Verbose  bool
	JSON     bool
}

func ViewTarget(w, diagnostics io.Writer, cfg *config.Config, req ViewRequest) (err error) {
	out := presentation.New(w, diagnostics, req.Verbose)
	defer func() { err = out.Finish(err) }()
	result, identityBundle, err := prepareRead(out, cfg, config.SelectionOptions{
		Command:             task.ModeView,
		Targets:             []string{req.Target},
		RequireSingleTarget: true,
	}, req.KeyFile)
	if err != nil {
		return err
	}
	filePair := result.FilePairs[0]

	plainData, err := seal.DecryptToBytes(seal.DecryptBytesOptions{
		InputFile:      filePair.EncryptedPath,
		OutputFile:     filePair.PlaintextPath,
		IdentityBundle: identityBundle,
		FormatOverride: filePair.Format,
	})
	if err != nil {
		return err
	}

	if req.JSON {
		format := filePair.Format
		if normalized, ok := seal.NormalizeFormatOverride(format); ok {
			format = normalized
		}
		return out.ViewReportJSON(filePair.EncryptedPath, format, plainData)
	}
	if _, err := out.Write(plainData); err != nil {
		return err
	}
	return nil
}
