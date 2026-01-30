package tools

import (
	"fmt"
	"os/exec"
	"strings"
)

// CheckTools verifies that all required external tools are installed
func CheckTools() error {
	required := []string{"age", "sops", "toml2yaml", "yaml2toml"}
	missing := []string{}

	for _, tool := range required {
		if !isToolInstalled(tool) {
			missing = append(missing, tool)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required tools: %s\n\nPlease install:\n- age: https://github.com/FiloSottile/age\n- sops: https://github.com/getsops/sops\n- remarshal (provides toml2yaml/yaml2toml): https://github.com/remarshal-project/remarshal\n  Install with: pipx install remarshal OR uv tool install remarshal", strings.Join(missing, ", "))
	}

	return nil
}

// isToolInstalled checks if a tool is available in PATH
func isToolInstalled(tool string) bool {
	_, err := exec.LookPath(tool)
	return err == nil
}

// GetToolVersion returns the version of a tool
func GetToolVersion(tool string) (string, error) {
	cmd := exec.Command(tool, "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}
