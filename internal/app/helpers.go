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

func WriteViewedTarget(w, diagnostics io.Writer, cfg *config.Config, target, keyFile string, verbose bool) (err error) {
	out := presentation.New(w, diagnostics, verbose)
	defer func() { err = out.Finish(err) }()
	result, identityBundle, err := prepareRead(out, cfg, config.SelectionOptions{
		Command:             task.ModeView,
		Target:              target,
		RequireSingleTarget: true,
	}, keyFile)
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

	if _, err := out.Write(plainData); err != nil {
		return err
	}
	return nil
}
