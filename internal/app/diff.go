package app

import (
	"fmt"
	"io"
	"os"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/diff"
	"github.com/fatih/color"
)

type DiffResult struct {
	Different bool
}

func DiffPlaintextAgainstEncryptedTargets(w io.Writer, cfg *config.Config, target, keyFile, cliFormat string, verbose bool, colorMode string) (DiffResult, error) {
	filePairs, err := ResolveTargetFilePairs(cfg, target)
	if err != nil {
		return DiffResult{}, err
	}
	colorEnabled, err := ResolveDiffColor(colorMode, w)
	if err != nil {
		return DiffResult{}, err
	}

	different := false
	for _, filePair := range filePairs {
		formatOverride, err := ResolveFormatOverride(cliFormat, filePair)
		if err != nil {
			return DiffResult{}, err
		}

		result, err := diff.PlaintextAgainstEncrypted(diff.Options{
			PlaintextFile:  filePair.PlaintextPath,
			EncryptedFile:  filePair.EncryptedPath,
			KeyFile:        keyFile,
			FormatOverride: formatOverride,
			Verbose:        verbose,
		})
		if err != nil {
			return DiffResult{}, err
		}
		if result.Different {
			different = true
			output := diff.HighlightUnifiedDiff(result.Diff, colorEnabled)
			if _, err := fmt.Fprint(w, output); err != nil {
				return DiffResult{}, err
			}
		}
	}

	return DiffResult{Different: different}, nil
}

func ResolveDiffColor(mode string, w io.Writer) (bool, error) {
	switch mode {
	case "", "auto":
		return w == os.Stdout && !color.NoColor, nil
	case "always":
		return true, nil
	case "never":
		return false, nil
	default:
		return false, fmt.Errorf("unsupported color mode %q (supported: auto, always, never)", mode)
	}
}
