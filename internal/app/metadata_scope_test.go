package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/sopsx"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestEncryptMetadataStaysWithinCurrentDirectory(t *testing.T) {
	for _, targets := range [][]string{nil, {"secret.yaml"}, {"../root.yaml"}} {
		name := "default"
		if len(targets) > 0 {
			name = targets[0]
		}
		t.Run(name, func(t *testing.T) {
			cfg, env, root := directoryMetadataTestConfig(t)
			require.NoError(t, EncryptFiles(cfg, EncryptRequest{
				Targets:               targets,
				Parallel:              1,
				UpdateProjectMetadata: true,
				SyncSOPSConfig:        true,
			}))

			requireDirectoryGitignore(t, root)
			data, err := os.ReadFile(".sops.yaml")
			require.NoError(t, err)
			var policy struct {
				Rules []struct {
					Path string `yaml:"path_regex"`
					Age  string `yaml:"age"`
				} `yaml:"creation_rules"`
			}
			require.NoError(t, yaml.Unmarshal(data, &policy))
			paths := make([]string, 0, len(policy.Rules))
			for _, rule := range policy.Rules {
				paths = append(paths, rule.Path)
				require.Equal(t, env.publicKey, rule.Age)
			}
			require.ElementsMatch(t, []string{
				`^secret\.enc\.yaml$`,
				`^second\.enc\.yaml$`,
				`^nested/child\.enc\.yaml$`,
				`^history\.enc\.yaml$`,
			}, paths)
			require.NotContains(t, string(data), root)

			report, err := Verify(cfg, VerifyRequest{
				Targets:        targets,
				NoDecrypt:      true,
				SyncSOPSConfig: true,
			})
			require.NoError(t, err)
			for _, finding := range report.Findings {
				require.NotEqual(t, "sops_config_drift", finding.Code)
			}
		})
	}
}

func TestDecryptGitignoreStaysWithinCurrentDirectory(t *testing.T) {
	cfg, env, root := directoryMetadataTestConfig(t)
	for _, name := range []string{"secret", "second", "nested/child"} {
		ciphertext, err := sopsx.Encrypt([]byte("token: fixture\n"), "yaml", []string{env.publicKey})
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(name+".enc.yaml", ciphertext, 0600))
		require.NoError(t, os.Remove(name+".yaml"))
	}
	before, err := os.ReadFile(filepath.Join(root, "root.yaml"))
	require.NoError(t, err)

	require.NoError(t, DecryptFiles(cfg, DecryptRequest{
		KeyFile:               env.keyFile,
		Parallel:              1,
		UpdateProjectMetadata: true,
	}))

	requireDirectoryGitignore(t, root)
	for _, name := range []string{"secret", "second", "nested/child", "history"} {
		require.FileExists(t, name+".yaml")
	}
	after, err := os.ReadFile(filepath.Join(root, "root.yaml"))
	require.NoError(t, err)
	require.Equal(t, before, after)
	require.NoFileExists(t, ".sops.yaml")
}

func directoryMetadataTestConfig(t *testing.T) (*config.Config, appCryptoTestEnv, string) {
	t.Helper()
	env := newAppCryptoTestEnv(t)
	root, err := os.Getwd()
	require.NoError(t, err)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	output, err := exec.Command("git", "init", "-q", root).CombinedOutput()
	require.NoError(t, err, "%s", output)
	child := filepath.Join(root, "service")
	require.NoError(t, os.MkdirAll(filepath.Join(child, "nested"), 0755))
	require.NoError(t, os.Mkdir(filepath.Join(root, "other"), 0755))
	rootConfig := `[recipients]
defaults = ["owner"]
[recipients.registry]
owner = "` + env.publicKey + `"
[[encryption.files]]
plaintext = "root.yaml"
encrypted = "root.enc.yaml"
[[encryption.files]]
plaintext = "other/outside.yaml"
encrypted = "other/outside.enc.yaml"
`
	childConfig := `[[encryption.files]]
plaintext = "secret.yaml"
encrypted = "secret.enc.yaml"
[[encryption.files]]
plaintext = "second.yaml"
encrypted = "second.enc.yaml"
[[encryption.files]]
plaintext = "nested/child.yaml"
encrypted = "nested/child.enc.yaml"
[[encryption.files]]
plaintext = "history.yaml"
encrypted = "history.enc.yaml"
`
	require.NoError(t, os.WriteFile(filepath.Join(root, ".yewseal.toml"), []byte(rootConfig), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(child, ".yewseal.toml"), []byte(childConfig), 0644))
	for _, path := range []string{"root.yaml", "other/outside.yaml", "service/secret.yaml", "service/second.yaml", "service/nested/child.yaml"} {
		require.NoError(t, os.WriteFile(filepath.Join(root, path), []byte("token: fixture\n"), 0600))
	}
	ciphertext, err := sopsx.Encrypt([]byte("token: fixture\n"), "yaml", []string{env.publicKey})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(child, "history.enc.yaml"), ciphertext, 0600))
	require.NoError(t, os.Chdir(child))
	cfg, err := config.LoadConfig()
	require.NoError(t, err)
	require.Len(t, cfg.LoadedFiles, 2)
	return cfg, env, root
}

func requireDirectoryGitignore(t *testing.T, root string) {
	t.Helper()
	data, err := os.ReadFile(".gitignore")
	require.NoError(t, err)
	require.NotContains(t, string(data), root)
	require.NotContains(t, string(data), "root.yaml")
	require.NotContains(t, string(data), "outside.yaml")
	for _, path := range []string{"secret.yaml", "second.yaml", "nested/child.yaml", "history.yaml"} {
		require.Contains(t, string(data), "\n"+path+"\n")
	}
}
