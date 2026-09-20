package app

import (
	"io"
	"strings"

	"github.com/YewFence/YewSeal/internal/agekey"
	cleaner "github.com/YewFence/YewSeal/internal/clean"
	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/presentation"
	"github.com/YewFence/YewSeal/internal/task"
)

type CleanRequest struct {
	Presentation  *presentation.Output
	Input         io.Reader
	KeyFile       string
	Targets       []string
	Force         bool
	SkipDifferent bool
}

// cleanStage 记录轻量预检的结果，保持处理顺序与 selection 顺序一致。
type cleanStage struct {
	pair   config.ResolvedFilePair
	result *cleaner.Result
}

func CleanFiles(cfg *config.Config, req CleanRequest) (err error) {
	out := presentation.OrDiscard(req.Presentation)
	out.SetDirectory(config.CurrentDir(cfg))
	defer func() { err = out.Finish(err) }()

	selection, err := config.ResolveSelection(cfg, config.SelectionOptions{
		Command: task.ModeClean,
		Targets: req.Targets,
	})
	if err != nil {
		return err
	}
	out.Selection(selection)

	stages := make([]cleanStage, 0, len(selection.FilePairs))
	needsIdentity := false
	for _, pair := range selection.FilePairs {
		stage := cleanStage{pair: pair}
		exists, inspectErr := cleaner.Inspect(pair.PlaintextPath)
		switch {
		case inspectErr != nil:
			stage.result = &cleaner.Result{PlaintextPath: pair.PlaintextPath, Status: cleaner.StatusFailed, Error: inspectErr}
		case !exists:
			stage.result = &cleaner.Result{PlaintextPath: pair.PlaintextPath, Status: cleaner.StatusAlreadyAbsent}
		default:
			needsIdentity = true
		}
		stages = append(stages, stage)
	}

	var identityBundle agekey.IdentityBundle
	if needsIdentity {
		identityBundle, err = agekey.GetIdentityBundle(req.KeyFile)
		if err != nil {
			return err
		}
		out.IdentityBundle(identityBundle)
	}

	input := req.Input
	if input == nil {
		input = strings.NewReader("")
	}
	prompts := out.Prompts(input)
	summary := cleaner.Summary{Results: make([]cleaner.Result, 0, len(stages))}
	for _, stage := range stages {
		result := cleaner.Result{PlaintextPath: stage.pair.PlaintextPath}
		switch {
		case stage.result != nil:
			result = *stage.result
		default:
			pair := stage.pair
			outcome, processErr := cleaner.Process(pair.PlaintextPath, cleaner.Options{
				EncryptedPath:  pair.EncryptedPath,
				Format:         pair.Format,
				IdentityBundle: identityBundle,
				Force:          req.Force,
				SkipDifferent:  req.SkipDifferent,
				ConfirmDifferent: func() (bool, error) {
					return out.ConfirmCleanDifference(prompts, pair.PlaintextPath)
				},
			})
			switch {
			case processErr != nil:
				result.Status, result.Error = cleaner.StatusFailed, processErr
			case outcome == cleaner.Removed:
				result.Status = cleaner.StatusRemoved
			case outcome == cleaner.AlreadyAbsent:
				result.Status = cleaner.StatusAlreadyAbsent
			default:
				result.Status = cleaner.StatusRetained
			}
		}
		summary.Add(result)
		out.CleanCompleted(result)
	}
	out.CleanSummary(summary)
	return prompts.Check(summary.Check())
}
