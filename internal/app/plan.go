package app

import (
	"io"
	"os"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/presentation"
	"github.com/YewFence/YewSeal/internal/task"
)

type PlanRequest struct {
	Targets     []string
	Diagnostics io.Writer
}

func PrintPlan(w io.Writer, cfg *config.Config, req PlanRequest, opts presentation.PlanPrintOptions) error {
	selection, err := config.ResolveSelection(cfg, config.SelectionOptions{Command: task.ModePlan, Targets: req.Targets})
	if err != nil {
		return err
	}
	out := presentation.New(w, req.Diagnostics, opts.Verbose)
	out.ScanSummary(selection.ScanStats)
	for i := range selection.FilePairs {
		selection.FilePairs[i].PlaintextExists = probeFile(selection.FilePairs[i].PlaintextPath)
		selection.FilePairs[i].EncryptedExists = probeFile(selection.FilePairs[i].EncryptedPath)
	}
	return out.Finish(out.Plan(cfg, selection, opts))
}

func probeFile(path string) *bool {
	exists := false
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		exists = true
	}
	return &exists
}
