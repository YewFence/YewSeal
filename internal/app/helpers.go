package app

import (
	"io"
	"os"

	"github.com/YewFence/YewSeal/internal/agekey"
	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/seal"
	"github.com/YewFence/YewSeal/internal/task"
)

func ValidateCLIFormatOverride(format string) (string, error) {
	return config.ValidateFormatOverride(format)
}

func WriteViewedTarget(w io.Writer, cfg *config.Config, target, keyFile, cliFormat string, verbose bool) error {
	result, err := config.SelectFilePairs(cfg, config.SelectionOptions{
		Command:             task.ModeDecrypt,
		Target:              target,
		Format:              cliFormat,
		RequireSingleTarget: true,
	})
	if err != nil {
		return err
	}
	filePairs, err := config.ValidateFilePairs(result.FilePairs)
	if err != nil {
		return err
	}
	filePair := filePairs[0]
	config.PrintSelection(verbose, cfg, result)

	identityBundle, err := resolveIdentityBundle(cfg, keyFile)
	if err != nil {
		return err
	}

	plainData, err := seal.DecryptToBytes(seal.DecryptBytesOptions{
		InputFile:      filePair.EncryptedPath,
		OutputFile:     filePair.PlaintextPath,
		IdentityBundle: identityBundle,
		FormatOverride: filePair.Format,
		Verbose:        verbose,
		Output:         os.Stderr,
	})
	if err != nil {
		return err
	}

	if _, err := w.Write(plainData); err != nil {
		return err
	}
	return nil
}

func resolveIdentityBundle(cfg *config.Config, explicitKeyFile string) (string, error) {
	fallback := ""
	if cfg != nil {
		fallback = cfg.GetKeyFile("")
	}
	bundle, err := agekey.GetIdentityBundleWithFallback(explicitKeyFile, fallback)
	if err != nil {
		return "", err
	}
	return bundle.String(), nil
}
