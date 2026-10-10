package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"filippo.io/age"
	"github.com/YewFence/YewSeal/internal/task"
	"github.com/stretchr/testify/require"
)

func TestExcludeStaysWithDeclaringConfigAndPreservesOrigins(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "child")
	require.NoError(t, os.MkdirAll(child, 0700))
	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	parentConfig := filepath.Join(root, ".yewseal.toml")
	childConfig := filepath.Join(child, ".yewseal.toml")
	require.NoError(t, os.WriteFile(parentConfig, fmt.Appendf(nil, `exclude = ["child/", "target/", "!app.toml"]
[[encryption.groups]]
patterns = ["*.toml", "!.yewseal.toml"]
[recipients]
defaults = ["owner"]
[recipients.registry]
owner = %q
`, identity.Recipient().String()), 0600))
	require.NoError(t, os.WriteFile(childConfig, []byte(`exclude = ["target/", "app.toml", "!app.toml"]
[[encryption.groups]]
patterns = ["*.toml", "!.yewseal.toml"]
[[encryption.files]]
plaintext = "target/explicit.toml"
encrypted = "target/explicit.enc.toml"
`), 0600))
	for _, path := range []string{"app.toml", "target/build.toml", "child/app.toml", "child/target/build.toml", "child/target/explicit.toml"} {
		full := filepath.Join(root, filepath.FromSlash(path))
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0700))
		require.NoError(t, os.WriteFile(full, []byte("value = 1\n"), 0600))
	}
	cfg, err := loadConfigFiles(root, []LoadedFile{{Path: parentConfig, Dir: root}, {Path: childConfig, Dir: child}})
	require.NoError(t, err)
	require.Equal(t, []ExcludeRule{
		{ConfigPath: parentConfig, Pattern: "child/", Index: 1},
		{ConfigPath: parentConfig, Pattern: "target/", Index: 2},
		{ConfigPath: parentConfig, Pattern: "!app.toml", Index: 3},
		{ConfigPath: childConfig, Pattern: "target/", Index: 1},
		{ConfigPath: childConfig, Pattern: "app.toml", Index: 2},
		{ConfigPath: childConfig, Pattern: "!app.toml", Index: 3},
	}, cfg.GetExcludeRules())
	for _, mode := range []string{task.ModeEncrypt, task.ModePlan, task.ModeClean} {
		selection, err := ResolveSelection(cfg, SelectionOptions{Command: mode})
		require.NoError(t, err)
		paths := make([]string, 0, len(selection.FilePairs))
		for _, pair := range selection.FilePairs {
			paths = append(paths, pair.PlaintextPath)
		}
		require.ElementsMatch(t, []string{
			filepath.Join(root, "app.toml"), filepath.Join(child, "app.toml"), filepath.Join(child, "target", "explicit.toml"),
		}, paths)
	}
	_, err = ResolveSelection(cfg, SelectionOptions{Command: task.ModeEncrypt, Targets: []string{"target/build.toml"}})
	require.ErrorContains(t, err, "not configured")
	explicit, err := ResolveSelection(cfg, SelectionOptions{Command: task.ModeEncrypt, Targets: []string{"child/target/explicit.toml"}})
	require.NoError(t, err)
	require.Len(t, explicit.FilePairs, 1)
	require.Equal(t, PairSourceExact, explicit.FilePairs[0].Source)
}

func TestLoadConfigValidatesExcludeWithoutGroups(t *testing.T) {
	for _, pattern := range []string{"", "!", "/", "a//b"} {
		t.Run(pattern, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, ".yewseal.toml")
			require.NoError(t, os.WriteFile(path, fmt.Appendf(nil, `exclude = [%q]
[[encryption.files]]
plaintext = "app.toml"
encrypted = "app.enc.toml"
`, pattern), 0600))
			_, err := loadConfigFiles(root, []LoadedFile{{Path: path, Dir: root}})
			require.ErrorContains(t, err, "invalid exclude in "+path)
		})
	}
}

func TestScanStatsAccumulateAcrossGroupsAndDoNotMutateConfig(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "app.toml"), []byte("value = 1\n"), 0600))
	cfg := &Config{CurrentDir: root, Encryption: EncryptionConfig{Groups: []GroupConfig{
		{ConfigDir: root, Patterns: []string{"*.toml"}},
		{ConfigDir: root, Patterns: []string{"*.toml"}},
	}}}
	for range 2 {
		selection, err := SelectFilePairs(cfg, SelectionOptions{Command: task.ModePlan})
		require.NoError(t, err)
		visited := []string{".", "app.toml"}
		require.Equal(t, task.ScanStats{Entries: len(visited) * len(cfg.Encryption.Groups) * 2, Directories: len(cfg.Encryption.Groups) * 2}, selection.ScanStats)
		require.Len(t, selection.FilePairs, 1)
	}
}
