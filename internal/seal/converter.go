package seal

import (
	"bytes"
	"os/exec"

	"github.com/YewFence/YewSeal/internal/errx"
)

func yamlToTOML(yamlContent []byte) ([]byte, error) {
	cmd := exec.Command("yaml2toml")
	cmd.Stdin = bytes.NewReader(yamlContent)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, &errx.ExternalCommandError{Op: "failed to convert YAML to TOML", Cmd: "yaml2toml", Stderr: stderr.String(), Err: err}
	}

	return stdout.Bytes(), nil
}

func tomlToYAML(tomlContent []byte) ([]byte, error) {
	cmd := exec.Command("toml2yaml")
	cmd.Stdin = bytes.NewReader(tomlContent)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, &errx.ExternalCommandError{Op: "failed to convert TOML to YAML", Cmd: "toml2yaml", Stderr: stderr.String(), Err: err}
	}

	return stdout.Bytes(), nil
}
