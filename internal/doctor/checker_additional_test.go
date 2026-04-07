package doctor

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)

	os.Stdout = w
	t.Cleanup(func() {
		os.Stdout = oldStdout
	})

	fn()

	require.NoError(t, w.Close())
	output, err := io.ReadAll(r)
	require.NoError(t, err)

	return string(output)
}

func doctorFakeExecutablePath(dir, name string) string {
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(dir, name)
}

func TestCheckRemarshal_Missing(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	err := CheckRemarshal()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "remarshal is required for TOML format support")
	assert.Contains(t, err.Error(), "uv tool install remarshal")
}

func TestCheckRemarshal_Found(t *testing.T) {
	tempDir := t.TempDir()
	path := doctorFakeExecutablePath(tempDir, "remarshal")
	require.NoError(t, os.WriteFile(path, []byte("fake"), 0o755))
	t.Setenv("PATH", tempDir)

	assert.NoError(t, CheckRemarshal())
}

func TestCheckToolsVerbose_PrintsMissingOptionalTool(t *testing.T) {
	oldOptionalTools := OptionalTools
	OptionalTools = []string{"definitely_missing_optional_tool_12345"}
	t.Cleanup(func() {
		OptionalTools = oldOptionalTools
	})

	var ok bool
	output := captureStdout(t, func() {
		ok = CheckToolsVerbose()
	})

	assert.True(t, ok)
	assert.Contains(t, output, "Embedded libraries (no installation needed):")
	assert.Contains(t, output, "Optional tools (for TOML format):")
	assert.Contains(t, output, "definitely_missing_optional_tool_12345: not found")
	assert.Contains(t, output, "All core dependencies are embedded.")
}

func TestCheckToolsVerbose_PrintsInstalledOptionalTool(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("Skipping test: git is not installed")
	}

	oldOptionalTools := OptionalTools
	OptionalTools = []string{"git"}
	t.Cleanup(func() {
		OptionalTools = oldOptionalTools
	})

	var ok bool
	output := captureStdout(t, func() {
		ok = CheckToolsVerbose()
	})

	assert.True(t, ok)
	assert.Contains(t, output, "git:")
	assert.NotContains(t, output, "git: not found")
}
