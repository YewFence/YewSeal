package crypto

import (
	"bytes"
	"fmt"
	"os/exec"
)

// YAMLToTOML converts YAML content to TOML format using remarshal
func YAMLToTOML(yamlContent []byte) ([]byte, error) {
	cmd := exec.Command("yaml2toml")
	cmd.Stdin = bytes.NewReader(yamlContent)
	
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to convert YAML to TOML: %w\n%s", err, stderr.String())
	}
	
	return stdout.Bytes(), nil
}

// TOMLToYAML converts TOML content to YAML format using remarshal
func TOMLToYAML(tomlContent []byte) ([]byte, error) {
	cmd := exec.Command("toml2yaml")
	cmd.Stdin = bytes.NewReader(tomlContent)
	
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to convert TOML to YAML: %w\n%s", err, stderr.String())
	}
	
	return stdout.Bytes(), nil
}
