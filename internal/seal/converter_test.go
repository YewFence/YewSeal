package seal

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTOMLToYAMLSuccess(t *testing.T) {
	skipIfToolMissing(t, "toml2yaml")

	tomlContent := []byte(`[section]
key = "value"
number = 42
`)

	yamlContent, err := tomlToYAML(tomlContent)
	require.NoError(t, err)
	assert.NotEmpty(t, yamlContent)
	assert.Contains(t, string(yamlContent), "section:")
	assert.Contains(t, string(yamlContent), "key:")
	assert.Contains(t, string(yamlContent), "value")
}

func TestTOMLToYAMLInvalidTOML(t *testing.T) {
	skipIfToolMissing(t, "toml2yaml")

	invalidTOML := []byte(`[section
key = "missing bracket"`)

	_, err := tomlToYAML(invalidTOML)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to convert TOML to YAML")
}

func TestYAMLToTOMLSuccess(t *testing.T) {
	skipIfToolMissing(t, "yaml2toml")

	yamlContent := []byte(`section:
  key: value
  number: 42
`)

	tomlContent, err := yamlToTOML(yamlContent)
	require.NoError(t, err)
	assert.NotEmpty(t, tomlContent)
	assert.Contains(t, string(tomlContent), "[section]")
	assert.Contains(t, string(tomlContent), "key")
	assert.Contains(t, string(tomlContent), "value")
}

func TestYAMLToTOMLInvalidYAML(t *testing.T) {
	skipIfToolMissing(t, "yaml2toml")

	invalidYAML := []byte(`key: value
  invalid indentation: true`)

	_, err := yamlToTOML(invalidYAML)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to convert YAML to TOML")
}

func TestRoundTripTOMLToYAMLToTOML(t *testing.T) {
	skipIfToolMissing(t, "toml2yaml")
	skipIfToolMissing(t, "yaml2toml")

	originalTOML := []byte(`[database]
host = "localhost"
port = 5432

[server]
name = "myserver"
enabled = true
`)

	yamlContent, err := tomlToYAML(originalTOML)
	require.NoError(t, err)

	tomlContent, err := yamlToTOML(yamlContent)
	require.NoError(t, err)

	assert.Contains(t, string(tomlContent), "database")
	assert.Contains(t, string(tomlContent), "host")
	assert.Contains(t, string(tomlContent), "localhost")
	assert.Contains(t, string(tomlContent), "server")
}

func skipIfToolMissing(t *testing.T, tool string) {
	t.Helper()
	if _, err := exec.LookPath(tool); err != nil {
		t.Skipf("Skipping test because %s is not installed", tool)
	}
}
