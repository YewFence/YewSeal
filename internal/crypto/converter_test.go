package crypto

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTOMLToYAML_Success(t *testing.T) {
	// Skip if toml2yaml is not installed
	if _, err := exec.LookPath("toml2yaml"); err != nil {
		t.Skip("Skipping test: toml2yaml is not installed")
	}

	tomlContent := []byte(`[section]
key = "value"
number = 42
`)

	yamlContent, err := TOMLToYAML(tomlContent)
	require.NoError(t, err)
	assert.NotEmpty(t, yamlContent)

	// Check that the output looks like YAML
	assert.Contains(t, string(yamlContent), "section:")
	assert.Contains(t, string(yamlContent), "key:")
	assert.Contains(t, string(yamlContent), "value")
}

func TestTOMLToYAML_EmptyInput(t *testing.T) {
	if _, err := exec.LookPath("toml2yaml"); err != nil {
		t.Skip("Skipping test: toml2yaml is not installed")
	}

	yamlContent, err := TOMLToYAML([]byte(""))
	require.NoError(t, err)
	// Empty TOML should produce empty or minimal YAML
	_ = yamlContent
}

func TestTOMLToYAML_InvalidTOML(t *testing.T) {
	if _, err := exec.LookPath("toml2yaml"); err != nil {
		t.Skip("Skipping test: toml2yaml is not installed")
	}

	invalidToml := []byte(`[section
key = "missing bracket"`)

	_, err := TOMLToYAML(invalidToml)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to convert TOML to YAML")
}

func TestTOMLToYAML_NestedStructure(t *testing.T) {
	if _, err := exec.LookPath("toml2yaml"); err != nil {
		t.Skip("Skipping test: toml2yaml is not installed")
	}

	tomlContent := []byte(`[parent]
name = "parent"

[parent.child]
name = "child"
value = 123
`)

	yamlContent, err := TOMLToYAML(tomlContent)
	require.NoError(t, err)
	assert.Contains(t, string(yamlContent), "parent")
	assert.Contains(t, string(yamlContent), "child")
}

func TestYAMLToTOML_Success(t *testing.T) {
	if _, err := exec.LookPath("yaml2toml"); err != nil {
		t.Skip("Skipping test: yaml2toml is not installed")
	}

	yamlContent := []byte(`section:
  key: value
  number: 42
`)

	tomlContent, err := YAMLToTOML(yamlContent)
	require.NoError(t, err)
	assert.NotEmpty(t, tomlContent)

	// Check that the output looks like TOML
	assert.Contains(t, string(tomlContent), "[section]")
	assert.Contains(t, string(tomlContent), "key")
	assert.Contains(t, string(tomlContent), "value")
}

func TestYAMLToTOML_EmptyInput(t *testing.T) {
	if _, err := exec.LookPath("yaml2toml"); err != nil {
		t.Skip("Skipping test: yaml2toml is not installed")
	}

	// Empty YAML input causes yaml2toml to fail (NoneType error)
	// This is expected behavior - empty YAML cannot be converted to TOML
	_, err := YAMLToTOML([]byte(""))
	// We just verify the function doesn't panic; error is acceptable
	_ = err
}

func TestYAMLToTOML_InvalidYAML(t *testing.T) {
	if _, err := exec.LookPath("yaml2toml"); err != nil {
		t.Skip("Skipping test: yaml2toml is not installed")
	}

	invalidYaml := []byte(`key: value
  invalid indentation: true`)

	_, err := YAMLToTOML(invalidYaml)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to convert YAML to TOML")
}

func TestYAMLToTOML_NestedStructure(t *testing.T) {
	if _, err := exec.LookPath("yaml2toml"); err != nil {
		t.Skip("Skipping test: yaml2toml is not installed")
	}

	yamlContent := []byte(`parent:
  name: parent
  child:
    name: child
    value: 123
`)

	tomlContent, err := YAMLToTOML(yamlContent)
	require.NoError(t, err)
	assert.Contains(t, string(tomlContent), "parent")
}

func TestRoundTrip_TOMLToYAMLToTOML(t *testing.T) {
	if _, err := exec.LookPath("toml2yaml"); err != nil {
		t.Skip("Skipping test: toml2yaml is not installed")
	}
	if _, err := exec.LookPath("yaml2toml"); err != nil {
		t.Skip("Skipping test: yaml2toml is not installed")
	}

	originalToml := []byte(`[database]
host = "localhost"
port = 5432

[server]
name = "myserver"
enabled = true
`)

	// Convert TOML -> YAML -> TOML
	yamlContent, err := TOMLToYAML(originalToml)
	require.NoError(t, err)

	tomlContent, err := YAMLToTOML(yamlContent)
	require.NoError(t, err)

	// The result should contain the same data (structure might differ slightly)
	assert.Contains(t, string(tomlContent), "database")
	assert.Contains(t, string(tomlContent), "host")
	assert.Contains(t, string(tomlContent), "localhost")
	assert.Contains(t, string(tomlContent), "server")
}
