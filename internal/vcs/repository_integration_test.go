package vcs

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// isolateVCSConfig keeps the developer's global git/jj configuration (commit
// signing, hooks, identity) out of the fixtures, and applies to the git and jj
// processes spawned by the adapters under test.
func isolateVCSConfig(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	gitConfig := filepath.Join(home, "gitconfig")
	require.NoError(t, os.WriteFile(gitConfig, []byte("[user]\n\tname = Test\n\temail = test@example.com\n[commit]\n\tgpgsign = false\n[init]\n\tdefaultBranch = main\n"), 0600))
	jjConfig := filepath.Join(home, "jjconfig.toml")
	require.NoError(t, os.WriteFile(jjConfig, []byte("[user]\nname = \"Test\"\nemail = \"test@example.com\"\n"), 0600))
	t.Setenv("GIT_CONFIG_GLOBAL", gitConfig)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("JJ_CONFIG", jjConfig)
	t.Setenv("HOME", home)
}

func runVCS(t *testing.T, dir, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s %v failed:\n%s", name, args, out)
}

func requireJJ(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("jj"); err == nil {
		return
	}
	if os.Getenv("CI") != "" {
		t.Fatal("jj is required in CI; it is declared in mise.toml")
	}
	t.Skip("jj is not installed")
}

func writeFile(t *testing.T, root, rel string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte("content"), 0o644))
}

func TestDetectFindsRepositoryFromNestedDirectory(t *testing.T) {
	isolateVCSConfig(t)
	root := t.TempDir()
	runVCS(t, root, "git", "init", "-q")
	nested := filepath.Join(root, "sub", "dir")
	require.NoError(t, os.MkdirAll(nested, 0o755))

	repository, err := Detect(nested)
	require.NoError(t, err)
	require.NotNil(t, repository)
	assert.Equal(t, "git", repository.Name())
	assert.Equal(t, root, repository.Root())
}

func TestDetectColocatedJJTakesPrecedenceOverGit(t *testing.T) {
	isolateVCSConfig(t)
	requireJJ(t)
	root := t.TempDir()
	runVCS(t, root, "jj", "git", "init", "--colocate")

	repository, err := Detect(root)
	require.NoError(t, err)
	require.NotNil(t, repository)
	assert.Equal(t, "jj", repository.Name())
	assert.Equal(t, root, repository.Root())
}

func TestDetectOutsideRepositoryReturnsNil(t *testing.T) {
	isolateVCSConfig(t)
	// HOME 也是临时目录，向上遍历不会碰到真实仓库
	dir := filepath.Join(t.TempDir(), "plain")
	require.NoError(t, os.MkdirAll(dir, 0o755))

	repository, err := Detect(dir)
	require.NoError(t, err)
	assert.Nil(t, repository)
}

func TestGitSnapshotClassifiesTrackedPendingAndUnmanaged(t *testing.T) {
	isolateVCSConfig(t)
	root := t.TempDir()
	runVCS(t, root, "git", "init", "-q")
	writeFile(t, root, "tracked.txt")
	writeFile(t, root, ".yewseal.toml")
	runVCS(t, root, "git", "add", ".")
	runVCS(t, root, "git", "commit", "-q", "-m", "init")
	writeFile(t, root, "pending.txt")

	repository, err := Detect(root)
	require.NoError(t, err)
	snapshot, err := repository.Snapshot()
	require.NoError(t, err)

	assert.Equal(t, root, snapshot.Root())
	assert.Equal(t, History, snapshot.Status(filepath.Join(root, "tracked.txt")))
	assert.Equal(t, Pending, snapshot.Status(filepath.Join(root, "pending.txt")))
	assert.Equal(t, Unmanaged, snapshot.Status(filepath.Join(root, "absent.txt")))
	assert.Equal(t, Unmanaged, snapshot.Status(filepath.Join(root, "..", "outside.txt")))

	files, err := repository.DiscoveryFiles()
	require.NoError(t, err)
	assert.Contains(t, files, ".yewseal.toml")
}

func TestJJSnapshotClassifiesHistoryAndPending(t *testing.T) {
	isolateVCSConfig(t)
	requireJJ(t)
	root := t.TempDir()
	runVCS(t, root, "jj", "git", "init")
	writeFile(t, root, ".yewseal.toml")
	writeFile(t, root, "tracked.txt")
	runVCS(t, root, "jj", "commit", "-m", "init")

	// 新文件在下一次 snapshot 时仍属于工作副本提交，被记为 pending
	writeFile(t, root, "pending.txt")

	repository, err := Detect(root)
	require.NoError(t, err)
	require.NotNil(t, repository)
	assert.Equal(t, "jj", repository.Name())

	snapshot, err := repository.Snapshot()
	require.NoError(t, err)
	assert.Equal(t, History, snapshot.Status(filepath.Join(root, "tracked.txt")))
	assert.Equal(t, Pending, snapshot.Status(filepath.Join(root, "pending.txt")))
	assert.Equal(t, Unmanaged, snapshot.Status(filepath.Join(root, "absent.txt")))
}

func TestJJDiscoveryFilesListsWorkingCopy(t *testing.T) {
	isolateVCSConfig(t)
	requireJJ(t)
	root := t.TempDir()
	runVCS(t, root, "jj", "git", "init")
	writeFile(t, root, ".yewseal.toml")
	writeFile(t, root, filepath.Join("nested", ".yewseal.toml"))

	repository, err := Detect(root)
	require.NoError(t, err)
	files, err := repository.DiscoveryFiles()
	require.NoError(t, err)
	assert.Equal(t, []string{".yewseal.toml", "nested/.yewseal.toml"}, files)
}

func TestDiscoveryFilesSurfacesCLIErrors(t *testing.T) {
	isolateVCSConfig(t)
	root := t.TempDir()
	// 只有 .git 标记目录而没有真实仓库内容，git 命令会失败
	require.NoError(t, os.Mkdir(filepath.Join(root, ".git"), 0o755))

	repository, err := Detect(root)
	require.NoError(t, err)
	require.NotNil(t, repository)
	_, err = repository.DiscoveryFiles()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to query git repository")
}
