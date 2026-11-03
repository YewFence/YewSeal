package verify

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/sopsconfig"
)

// checkSOPSDrift compares .sops.yaml with the policy owned by its configuration directory.
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
	expected, err := sopsconfig.Render(allPairs, cwd)
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
		Message:  fmt.Sprintf("%s differs from its source configuration policy", filepath.Join(cwd, ".sops.yaml")),
		Hint:     "run 'yews encrypt' to restore the managed SOPS rules; for manually maintained rules, follow the opt-out instructions in the configuration guide" + hintAgentHandoff,
	})
}
