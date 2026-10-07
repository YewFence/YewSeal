package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"filippo.io/age"
	"github.com/stretchr/testify/require"
)

func TestLoadConfigDiscoversWholeGitRepositoryAndKeepsCurrentDirectoryScope(t *testing.T) {
	root := t.TempDir()
	runConfigGit(t, root, "init", "-q")
	api := filepath.Join(root, "api")
	worker := filepath.Join(root, "worker")
	require.NoError(t, os.MkdirAll(api, 0755))
	require.NoError(t, os.MkdirAll(worker, 0755))

	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	writeConfig(t, filepath.Join(root, ".yewseal.toml"), `[recipients]
defaults = ["owner"]
[recipients.registry]
owner = "`+identity.Recipient().String()+`"
[[encryption.files]]
plaintext = "root.yaml"
encrypted = "root.enc.yaml"
`)
	writeConfig(t, filepath.Join(api, ".yewseal.toml"), `[[encryption.files]]
plaintext = "secret.yaml"
encrypted = "secret.enc.yaml"
recipients = ["owner"]
`)
	writeConfig(t, filepath.Join(worker, ".yewseal.toml"), `[[encryption.files]]
plaintext = "worker.yaml"
encrypted = "worker.enc.yaml"
`)

	t.Chdir(api)
	cfg, err := LoadConfig()
	require.NoError(t, err)
	require.Equal(t, []string{
		filepath.Join(root, ".yewseal.toml"),
		filepath.Join(api, ".yewseal.toml"),
		filepath.Join(worker, ".yewseal.toml"),
	}, loadedConfigPaths(cfg))
	require.Len(t, cfg.GetFiles(), 3)

	selection, err := SelectFilePairs(cfg, SelectionOptions{Command: "encrypt"})
	require.NoError(t, err)
	require.Len(t, selection.FilePairs, 1)
	require.Equal(t, filepath.Join(api, "secret.yaml"), selection.FilePairs[0].PlaintextPath)
}

func TestLoadConfigGitDiscoveryHonorsTrackedIgnoredAndDeletedState(t *testing.T) {
	root := t.TempDir()
	runConfigGit(t, root, "init", "-q")
	writeConfig(t, filepath.Join(root, ".yewseal.toml"), fileConfig("root"))
	for _, dir := range []string{"tracked", "deleted", "ignored"} {
		require.NoError(t, os.MkdirAll(filepath.Join(root, dir), 0755))
		writeConfig(t, filepath.Join(root, dir, ".yewseal.toml"), fileConfig(dir))
	}
	runConfigGit(t, root, "add", "tracked/.yewseal.toml", "deleted/.yewseal.toml")
	runConfigGit(t, root, "commit", "-q", "-m", "track configs")
	require.NoError(t, os.Remove(filepath.Join(root, "deleted", ".yewseal.toml")))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".gitignore"), []byte("tracked/.yewseal.toml\nignored/.yewseal.toml\ndeleted/.yewseal.toml\n"), 0644))

	t.Chdir(root)
	cfg, err := LoadConfig()
	require.NoError(t, err)
	require.Equal(t, []string{
		filepath.Join(root, ".yewseal.toml"),
		filepath.Join(root, "tracked", ".yewseal.toml"),
	}, loadedConfigPaths(cfg))
}

func TestLoadConfigPreservesDirectIgnoredConfigLookup(t *testing.T) {
	root := t.TempDir()
	runConfigGit(t, root, "init", "-q")
	service := filepath.Join(root, "service")
	require.NoError(t, os.MkdirAll(service, 0755))
	writeConfig(t, filepath.Join(root, ".yewseal.toml"), fileConfig("root"))
	writeConfig(t, filepath.Join(service, ".yewseal.toml"), fileConfig("service"))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".gitignore"), []byte("service/.yewseal.toml\n"), 0644))

	t.Chdir(service)
	cfg, err := LoadConfig()
	require.NoError(t, err)
	require.Equal(t, []string{
		filepath.Join(root, ".yewseal.toml"),
		filepath.Join(service, ".yewseal.toml"),
	}, loadedConfigPaths(cfg))
}

func TestLoadConfigDoesNotPromoteIgnoredHigherPriorityConfig(t *testing.T) {
	root := t.TempDir()
	runConfigGit(t, root, "init", "-q")
	child := filepath.Join(root, "child")
	require.NoError(t, os.MkdirAll(filepath.Join(child, ".yewseal"), 0755))
	writeConfig(t, filepath.Join(root, ".yewseal.toml"), fileConfig("root"))
	writeConfig(t, filepath.Join(child, ".yewseal.toml"), fileConfig("visible"))
	writeConfig(t, filepath.Join(child, ".yewseal", ".yewseal.toml"), fileConfig("ignored"))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".gitignore"), []byte("child/.yewseal/.yewseal.toml\n"), 0644))

	t.Chdir(root)
	cfg, err := LoadConfig()
	require.NoError(t, err)
	require.Equal(t, []string{
		filepath.Join(root, ".yewseal.toml"),
		filepath.Join(child, ".yewseal.toml"),
	}, loadedConfigPaths(cfg))
}

func TestLoadConfigOrdersParentsBeforeSortedSiblings(t *testing.T) {
	root := t.TempDir()
	runConfigGit(t, root, "init", "-q")
	for _, dir := range []string{"", "z", "a", filepath.Join("z", "nested")} {
		require.NoError(t, os.MkdirAll(filepath.Join(root, dir), 0755))
		name := "root"
		if dir != "" {
			name = filepath.Base(dir)
		}
		writeConfig(t, filepath.Join(root, dir, ".yewseal.toml"), fileConfig(name))
	}

	t.Chdir(root)
	cfg, err := LoadConfig()
	require.NoError(t, err)
	require.Equal(t, []string{
		filepath.Join(root, ".yewseal.toml"),
		filepath.Join(root, "a", ".yewseal.toml"),
		filepath.Join(root, "z", ".yewseal.toml"),
		filepath.Join(root, "z", "nested", ".yewseal.toml"),
	}, loadedConfigPaths(cfg))
}

func TestLoadConfigRejectsInvalidConfigOutsideCurrentDirectory(t *testing.T) {
	root := t.TempDir()
	runConfigGit(t, root, "init", "-q")
	writeConfig(t, filepath.Join(root, ".yewseal.toml"), fileConfig("root"))
	require.NoError(t, os.Mkdir(filepath.Join(root, "other"), 0755))
	writeConfig(t, filepath.Join(root, "other", ".yewseal.toml"), "[broken")

	t.Chdir(root)
	_, err := LoadConfig()
	require.ErrorContains(t, err, filepath.Join("other", ".yewseal.toml"))
	require.ErrorContains(t, err, "failed to parse config file")
}

func TestLoadConfigDiscoversNewJJConfig(t *testing.T) {
	if _, err := exec.LookPath("jj"); err != nil {
		if os.Getenv("CI") != "" {
			t.Fatal("jj is required in CI; it is declared in mise.toml")
		}
		t.Skip("jj is not installed")
	}
	root := t.TempDir()
	isolateConfigVCS(t)
	runConfigJJ(t, root, "git", "init", "--no-colocate")
	child := filepath.Join(root, "child")
	require.NoError(t, os.Mkdir(child, 0755))
	writeConfig(t, filepath.Join(root, ".yewseal.toml"), fileConfig("root"))
	writeConfig(t, filepath.Join(child, ".yewseal.toml"), fileConfig("child"))

	t.Chdir(root)
	cfg, err := LoadConfig()
	require.NoError(t, err)
	require.Equal(t, []string{
		filepath.Join(root, ".yewseal.toml"),
		filepath.Join(child, ".yewseal.toml"),
	}, loadedConfigPaths(cfg))
}

func TestLoadConfigReportsVCSQueryFailure(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(root, ".git"), 0755))
	writeConfig(t, filepath.Join(root, ".yewseal.toml"), fileConfig("root"))

	t.Chdir(root)
	_, err := LoadConfig()
	require.ErrorContains(t, err, "failed to query git repository")
}

func TestLoadConfigJJRespectsIgnoredTrackedAndDeletedConfigs(t *testing.T) {
	if _, err := exec.LookPath("jj"); err != nil {
		if os.Getenv("CI") != "" {
			t.Fatal("jj is required in CI; it is declared in mise.toml")
		}
		t.Skip("jj is not installed")
	}
	root := t.TempDir()
	isolateConfigVCS(t)
	runConfigJJ(t, root, "git", "init", "--no-colocate")
	writeConfig(t, filepath.Join(root, ".yewseal.toml"), fileConfig("root"))
	for _, dir := range []string{"tracked", "deleted"} {
		require.NoError(t, os.Mkdir(filepath.Join(root, dir), 0755))
		writeConfig(t, filepath.Join(root, dir, ".yewseal.toml"), fileConfig(dir))
	}
	runConfigJJ(t, root, "commit", "-m", "track configs")
	require.NoError(t, os.Remove(filepath.Join(root, "deleted", ".yewseal.toml")))
	require.NoError(t, os.Mkdir(filepath.Join(root, "ignored"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".gitignore"), []byte("tracked/.yewseal.toml\ndeleted/.yewseal.toml\nignored/.yewseal.toml\n"), 0644))
	writeConfig(t, filepath.Join(root, "ignored", ".yewseal.toml"), fileConfig("ignored"))

	t.Chdir(root)
	cfg, err := LoadConfig()
	require.NoError(t, err)
	require.Equal(t, []string{
		filepath.Join(root, ".yewseal.toml"),
		filepath.Join(root, "tracked", ".yewseal.toml"),
	}, loadedConfigPaths(cfg))
}

func fileConfig(name string) string {
	return `[[encryption.files]]
plaintext = "` + name + `.yaml"
encrypted = "` + name + `.enc.yaml"
`
}

func writeConfig(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
}

func loadedConfigPaths(cfg *Config) []string {
	paths := make([]string, 0, len(cfg.LoadedFiles))
	for _, file := range cfg.LoadedFiles {
		paths = append(paths, file.Path)
	}
	return paths
}

func isolateConfigVCS(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	gitConfig := filepath.Join(home, "gitconfig")
	require.NoError(t, os.WriteFile(gitConfig, []byte("[user]\n\tname = Test\n\temail = test@example.com\n[commit]\n\tgpgsign = false\n"), 0600))
	jjConfig := filepath.Join(home, "jjconfig.toml")
	require.NoError(t, os.WriteFile(jjConfig, []byte("[user]\nname = \"Test\"\nemail = \"test@example.com\"\n"), 0600))
	t.Setenv("GIT_CONFIG_GLOBAL", gitConfig)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("JJ_CONFIG", jjConfig)
	t.Setenv("HOME", home)
}

func runConfigGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-c", "user.name=Test", "-c", "user.email=test@example.com", "-c", "commit.gpgsign=false"}, args...)...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v failed:\n%s", args, output)
}

func runConfigJJ(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("jj", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "jj %v failed:\n%s", args, output)
}
