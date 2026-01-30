package tools

import (
	"fmt"
	"os/exec"
	"strings"
)

// RequiredTools 是所有必需的外部工具列表
var RequiredTools = []string{"age", "sops", "remarshal"}

// CheckTools verifies that all required external tools are installed
func CheckTools() error {
	missing := []string{}

	for _, tool := range RequiredTools {
		if !isToolInstalled(tool) {
			missing = append(missing, tool)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required tools: %s\n\nPlease install:\n- age: https://github.com/FiloSottile/age\n- sops: https://github.com/getsops/sops\n- remarshal: pipx install remarshal OR uv tool install remarshal", strings.Join(missing, ", "))
	}

	return nil
}

// CheckToolsVerbose 检查工具并输出详细信息，返回是否全部安装成功
func CheckToolsVerbose() bool {
	allOk := true
	fmt.Println("Checking required tools...")
	fmt.Println()

	for _, tool := range RequiredTools {
		version, err := GetToolVersion(tool)
		if err != nil {
			fmt.Printf("  ✗ %s: not found\n", tool)
			allOk = false
		} else {
			if idx := strings.Index(version, "\n"); idx != -1 {
				version = version[:idx]
			}
			fmt.Printf("  ✓ %s: %s\n", tool, version)
		}
	}

	fmt.Println()
	if !allOk {
		fmt.Println("Some tools are missing. Install instructions:")
		fmt.Println("  - age: https://github.com/FiloSottile/age")
		fmt.Println("  - sops: https://github.com/getsops/sops")
		fmt.Println("  - remarshal: pipx install remarshal")
	} else {
		fmt.Println("All tools are installed!")
	}
	return allOk
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
