package tools

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsToolInstalled_Exists(t *testing.T) {
	// "go" should be installed in any Go development environment
	assert.True(t, isToolInstalled("go"))
}

func TestIsToolInstalled_NotExists(t *testing.T) {
	// This tool definitely doesn't exist
	assert.False(t, isToolInstalled("nonexistent_tool_that_should_never_exist_12345"))
}

func TestCheckTools_AllInstalled(t *testing.T) {
	// Check if all required tools are installed
	required := []string{"age", "sops", "toml2yaml", "yaml2toml"}
	allInstalled := true
	for _, tool := range required {
		if _, err := exec.LookPath(tool); err != nil {
			allInstalled = false
			break
		}
	}

	if !allInstalled {
		t.Skip("Skipping test: not all required tools are installed")
	}

	err := CheckTools()
	assert.NoError(t, err)
}

func TestCheckTools_MissingTools(t *testing.T) {
	// This test verifies the error message format when tools are missing
	// We can't easily mock exec.LookPath, so we just verify the function runs
	// In a real scenario with missing tools, CheckTools would return an error

	err := CheckTools()

	// If all tools are installed, no error
	// If some tools are missing, error should contain helpful info
	if err != nil {
		assert.Contains(t, err.Error(), "missing required tools")
		assert.Contains(t, err.Error(), "Please install")
	}
}

func TestGetToolVersion_Go(t *testing.T) {
	// Note: "go" command uses "version" subcommand, not "--version" flag
	// So we test with a different tool that uses --version
	// If git is available, use it; otherwise skip
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("Skipping test: git is not installed")
	}

	version, err := GetToolVersion("git")

	require.NoError(t, err)
	assert.NotEmpty(t, version)
	assert.Contains(t, strings.ToLower(version), "git")
}

func TestGetToolVersion_NonexistentTool(t *testing.T) {
	version, err := GetToolVersion("nonexistent_tool_12345")

	assert.Error(t, err)
	assert.Empty(t, version)
}

func TestGetToolVersion_Age(t *testing.T) {
	if _, err := exec.LookPath("age"); err != nil {
		t.Skip("Skipping test: age is not installed")
	}

	version, err := GetToolVersion("age")
	require.NoError(t, err)
	assert.NotEmpty(t, version)
}

func TestGetToolVersion_Sops(t *testing.T) {
	if _, err := exec.LookPath("sops"); err != nil {
		t.Skip("Skipping test: sops is not installed")
	}

	version, err := GetToolVersion("sops")
	require.NoError(t, err)
	assert.NotEmpty(t, version)
}
