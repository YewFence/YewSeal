package app

import (
	"errors"
	"io"
	"io/fs"
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

// probeFile reports whether path is a file on disk. false means the path
// is definitively absent or a directory; nil means the check itself
// failed, so presence is unknown — an unreadable directory must not read
// as a clean checkout.
func probeFile(path string) *bool {
	info, err := os.Stat(path)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		exists := false
		return &exists
	}
	exists := !info.IsDir()
	return &exists
}
