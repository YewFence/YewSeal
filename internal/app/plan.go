package app

import (
	"io"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/presentation"
	"github.com/YewFence/YewSeal/internal/task"
)

type PlanRequest struct {
	Target   string
	Patterns []string
}

func PrintPlan(w io.Writer, cfg *config.Config, req PlanRequest, opts presentation.PlanPrintOptions) error {
	selection, err := config.ResolveSelection(cfg, config.SelectionOptions{Command: task.ModePlan, Target: req.Target, Patterns: req.Patterns})
	if err != nil {
		return err
	}
	out := presentation.New(w, io.Discard, opts.Verbose)
	return out.Finish(out.Plan(cfg, selection, opts))
}
