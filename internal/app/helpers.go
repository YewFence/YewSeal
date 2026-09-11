package app

import (
	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/seal"
	"github.com/YewFence/YewSeal/internal/task"
	"io"
)

func ValidateCLIFormatOverride(format string) (string, error) {
	return config.ValidateFormatOverride(format)
}

func WriteViewedTarget(w, diagnostics io.Writer, cfg *config.Config, target, keyFile string, verbose bool) error {
	diagnostics = &diagnosticWriter{Writer: diagnostics}
	result, identityBundle, err := prepareRead(diagnostics, cfg, config.SelectionOptions{
		Command:             task.ModeView,
		Targets:             []string{target},
		RequireSingleTarget: true,
	}, keyFile, verbose)
	if err != nil {
		return err
	}
	filePair := result.FilePairs[0]

	plainData, err := seal.DecryptToBytes(seal.DecryptBytesOptions{
		InputFile:      filePair.EncryptedPath,
		OutputFile:     filePair.PlaintextPath,
		IdentityBundle: identityBundle,
		FormatOverride: filePair.Format,
		Verbose:        verbose,
		Output:         diagnostics,
	})
	if err != nil {
		return err
	}

	if n, err := w.Write(plainData); err != nil {
		return err
	} else if n != len(plainData) {
		return io.ErrShortWrite
	}
	return nil
}
