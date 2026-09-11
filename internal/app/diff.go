package app

import (
	"fmt"
	"io"
	"os"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/diff"
	"github.com/YewFence/YewSeal/internal/task"
	"github.com/fatih/color"
)

type DiffResult struct {
	Different bool
	Summary   diff.Summary
}

func DiffPlaintextAgainstEncryptedTargets(w, diagnostics io.Writer, cfg *config.Config, targets []string, keyFile string, verbose bool, colorMode string, strict bool) (DiffResult, error) {
	diagnosticOutput := &diagnosticWriter{Writer: diagnostics}
	selection, identityBundle, err := prepareRead(diagnosticOutput, cfg, config.SelectionOptions{
		Command:          task.ModeDiff,
		Targets:          targets,
		AllowEmptyTarget: true,
	}, keyFile, verbose)
	if err != nil {
		return DiffResult{}, err
	}

	colorEnabled, err := ResolveDiffColor(colorMode, w)
	if err != nil {
		return DiffResult{}, err
	}

	comparison := DiffResult{}
	for _, filePair := range selection.FilePairs {
		plainPath := config.DisplayPath(config.CurrentDir(cfg), filePair.PlaintextPath)
		encPath := config.DisplayPath(config.CurrentDir(cfg), filePair.EncryptedPath)
		result, err := diff.PlaintextAgainstEncrypted(diff.Options{
			PlaintextFile:  filePair.PlaintextPath,
			EncryptedFile:  filePair.EncryptedPath,
			PlaintextLabel: plainPath,
			EncryptedLabel: encPath,
			IdentityBundle: identityBundle,
			FormatOverride: filePair.Format,
			Verbose:        verbose,
			Diagnostics:    diagnosticOutput,
		})
		if diagnosticOutput.err != nil {
			return comparison, fmt.Errorf("failed to write comparison diagnostics: %w", diagnosticOutput.err)
		}
		comparison.Summary.Add(plainPath, encPath, result, err)
		if err != nil || result.Skipped != "" {
			continue
		}
		if result.Different {
			comparison.Different = true
			output := diff.HighlightUnifiedDiff(result.Diff, colorEnabled)
			if n, err := fmt.Fprint(w, output); err != nil {
				return comparison, fmt.Errorf("failed to write diff output: %w", err)
			} else if n != len(output) {
				return comparison, fmt.Errorf("failed to write diff output: %w", io.ErrShortWrite)
			}
		}
	}

	if err := comparison.Summary.Report(diagnosticOutput); err != nil {
		return comparison, fmt.Errorf("failed to write comparison report: %w", err)
	}
	return comparison, comparison.Summary.Check(strict)
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
