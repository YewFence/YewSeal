package app

import (
	"errors"
	"fmt"
	"io"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/diff"
	"github.com/YewFence/YewSeal/internal/presentation"
	"github.com/YewFence/YewSeal/internal/task"
)

type DiffResult struct {
	Different bool
	Summary   diff.Summary
}

type DiffRequest struct {
	Targets   []string
	KeyFile   string
	Verbose   bool
	ColorMode string
	JSON      bool
}

func DiffTargets(w, diagnostics io.Writer, cfg *config.Config, req DiffRequest) (comparison DiffResult, err error) {
	out := presentation.New(w, diagnostics, req.Verbose)
	defer func() { err = out.Finish(errors.Join(err, comparison.Summary.Check())) }()
	selection, identityBundle, err := prepareRead(out, cfg, config.SelectionOptions{
		Command:          task.ModeDiff,
		Targets:          req.Targets,
		AllowEmptyTarget: true,
	}, req.KeyFile)
	if err != nil {
		return DiffResult{}, err
	}

	colorEnabled, err := presentation.ResolveDiffColor(req.ColorMode, w)
	if err != nil {
		return DiffResult{}, err
	}

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
		})
		comparison.Summary.Add(plainPath, encPath, result, err)
		out.ComparisonCompleted(comparison.Summary.Results[len(comparison.Summary.Results)-1])
		if err != nil || result.Skipped != "" {
			continue
		}
		if result.Different {
			comparison.Different = true
			if !req.JSON {
				if err := out.Diff(result.Diff, colorEnabled); err != nil {
					return comparison, fmt.Errorf("failed to write diff output: %w", err)
				}
			}
		}
	}

	if req.JSON {
		if err := out.DiffReportJSON(comparison.Summary); err != nil {
			return comparison, err
		}
	}
	out.ComparisonSummary(comparison.Summary)
	return comparison, nil
}
