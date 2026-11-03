package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCLIMetadataIndependentOfWorkingDirectory(t *testing.T) {
	binary := buildYews(t)
	clearCommandEnvironment(t)
	for _, location := range []string{".yewseal.toml", ".config/.yewseal.toml", ".yewseal/.yewseal.toml"} {
		t.Run(location, func(t *testing.T) {
			dir := t.TempDir()
			output, err := exec.Command("git", "init", "-q", dir).CombinedOutput()
			require.NoError(t, err, "%s", output)
			child := filepath.Join(dir, "app")
			nested := filepath.Join(child, "nested")
			require.NoError(t, os.MkdirAll(nested, 0700))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("token: fixture\n"), 0600))
			_, stderr, code := runMetadataCommand(t, binary, dir, nil, "init", "--input", "config.yaml")
			require.Zero(t, code, stderr)
			rootConfig, err := os.OpenFile(filepath.Join(dir, ".yewseal.toml"), os.O_APPEND|os.O_WRONLY, 0600)
			require.NoError(t, err)
			_, err = rootConfig.WriteString(`
[[encryption.files]]
plaintext = "app/parent.yaml"
encrypted = "app/parent.enc.yaml"
`)
			require.NoError(t, err)
			require.NoError(t, rootConfig.Close())
			require.NoError(t, os.WriteFile(filepath.Join(child, "parent.yaml"), []byte("token: parent\n"), 0600))
			childConfig := filepath.Join(child, location)
			require.NoError(t, os.MkdirAll(filepath.Dir(childConfig), 0700))
			require.NoError(t, os.WriteFile(childConfig, []byte(`[[encryption.groups]]
patterns = ["secret.yaml"]
`), 0600))
			require.NoError(t, os.WriteFile(filepath.Join(child, "secret.yaml"), []byte("token: child\n"), 0600))
			keyFile := filepath.Join(dir, ".age", "keys.txt")
			_, stderr, code = runMetadataCommand(t, binary, dir, nil, "encrypt", "--key-file", keyFile)
			require.Zero(t, code, stderr)

			metadata := make(map[string][]byte)
			for _, owner := range []string{dir, child} {
				for _, name := range []string{".gitignore", ".sops.yaml"} {
					path := filepath.Join(owner, name)
					metadata[path], err = os.ReadFile(path)
					require.NoError(t, err)
					if owner == dir {
						require.NotContains(t, string(metadata[path]), "secret")
						require.Contains(t, string(metadata[path]), "app/parent")
					} else {
						require.Contains(t, string(metadata[path]), "secret")
						require.NotContains(t, string(metadata[path]), "app/")
						require.NotContains(t, string(metadata[path]), "parent")
					}
				}
			}
			for _, cwd := range []string{dir, nested, nested} {
				for _, command := range []string{"encrypt", "decrypt"} {
					_, stderr, code = runMetadataCommand(t, binary, cwd, nil, command, dir, "--key-file", keyFile)
					require.Zero(t, code, stderr)
					for path, before := range metadata {
						after, err := os.ReadFile(path)
						require.NoError(t, err)
						require.Equal(t, before, after, "%s changed when running from %s", path, cwd)
					}
					for _, name := range []string{".gitignore", ".sops.yaml"} {
						require.NoFileExists(t, filepath.Join(nested, name))
					}
				}
			}
			_, stderr, code = runMetadataCommand(t, binary, nested, nil, "verify", dir, "--no-decrypt", "--key-file", keyFile)
			require.Zero(t, code, stderr)
			require.NoError(t, os.WriteFile(filepath.Join(child, ".sops.yaml"), []byte("# stale\n"), 0600))
			stdout, stderr, code := runMetadataCommand(t, binary, nested, nil, "verify", dir, "--no-decrypt", "--key-file", keyFile)
			require.Equal(t, 1, code, stderr)
			require.Contains(t, stdout, "sops_config_drift")
		})
	}
}
