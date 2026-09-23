package verify

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/sopsconfig"
)

// checkSOPSDrift compares .sops.yaml in cwd with the exact bytes encrypt's
// synchronization would write for the complete resolved policy.
func checkSOPSDrift(report *Report, allPairs []config.ResolvedFilePair, cwd string) {
	onDisk, err := os.ReadFile(filepath.Join(cwd, ".sops.yaml"))
	if os.IsNotExist(err) {
		report.AddSkip(".sops.yaml does not exist; drift not checked")
		return
	}
	if err != nil {
		report.Add(Finding{
			Code:     "sops_config_read_error",
			Severity: SeverityError,
			Message:  fmt.Sprintf("failed to read .sops.yaml: %v", err),
		})
		return
	}
	// encrypt synchronizes with cwd-relative paths; render the same view.
	expected, err := sopsconfig.Render(config.DisplayResolvedFilePairs(allPairs, cwd))
	if err != nil {
		report.Add(Finding{
			Code:     "sops_config_generate_error",
			Severity: SeverityError,
			Message:  fmt.Sprintf("failed to render the expected .sops.yaml: %v", err),
		})
		return
	}
	if bytes.Equal(onDisk, expected) {
		report.AddPass()
		return
	}
	report.Add(Finding{
		Code:     "sops_config_drift",
		Severity: SeverityError,
		Message:  ".sops.yaml differs from the resolved project policy",
		Hint:     "run 'yews encrypt' to regenerate it",
	})
}
