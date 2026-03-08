package tools

import (
	"fmt"
	"os/exec"
	"strings"
)

// RequiredTools is empty — age and sops are now embedded as Go libraries.
var RequiredTools = []string{}

// OptionalTools are external tools only needed for specific formats.
var OptionalTools = []string{"remarshal"}

// CheckTools verifies that all required external tools are installed.
// With age/sops embedded, this always returns nil.
func CheckTools() error {
	return nil
}

// CheckRemarshal checks if remarshal (toml2yaml/yaml2toml) is installed
// This is only required for TOML format support
func CheckRemarshal() error {
	if !isToolInstalled("remarshal") {
		return fmt.Errorf("remarshal is required for TOML format support\n\nPlease install: pipx install remarshal OR uv tool install remarshal")
	}
	return nil
}

// CheckToolsVerbose checks tool status and displays embedded library versions
func CheckToolsVerbose() bool {
	fmt.Println("Embedded libraries (no installation needed):")
	fmt.Println("  ✓ age: filippo.io/age (embedded)")
	fmt.Println("  ✓ sops: github.com/getsops/sops/v3 (embedded)")

	fmt.Println()
	fmt.Println("Optional tools (for TOML format):")
	for _, tool := range OptionalTools {
		version, err := GetToolVersion(tool)
		if err != nil {
			fmt.Printf("  ○ %s: not found (install for TOML support)\n", tool)
		} else {
			if idx := strings.Index(version, "\n"); idx != -1 {
				version = version[:idx]
			}
			fmt.Printf("  ✓ %s: %s\n", tool, version)
		}
	}

	fmt.Println()
	fmt.Println("All core dependencies are embedded. Only remarshal is needed for TOML format.")
	return true
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
