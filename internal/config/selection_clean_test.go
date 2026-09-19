package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/YewFence/YewSeal/internal/task"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func cleanPairPaths(pairs []ResolvedFilePair) []string {
	paths := make([]string, 0, len(pairs))
	for _, pair := range pairs {
		paths = append(paths, pair.PlaintextPath)
	}
	return paths
}

// TestCleanGroupDiscoveryUsesBothSides 覆盖 issue #40 §5.2：clean 的 group
// discovery 是明文与密文两侧的并集，只剩密文的映射按 protocol path 推导
// logical plaintext 后仍是候选。
func TestCleanGroupDiscoveryUsesBothSides(t *testing.T) {
	cfg := selectionConfig(t)
	bothPlain := selectionFile(t, cfg.CurrentDir, "both.yaml")
	bothCipher := selectionFile(t, cfg.CurrentDir, "both.enc.yaml")
	onlyPlain := selectionFile(t, cfg.CurrentDir, "only.yaml")
	selectionFile(t, cfg.CurrentDir, "only.enc.yaml")
	cfg.Encryption.Groups = []GroupConfig{{ConfigDir: cfg.CurrentDir, Patterns: []string{"*.yaml"}}}

	selection, err := ResolveSelection(cfg, SelectionOptions{Command: task.ModeClean})
	require.NoError(t, err)
	// 同一 logical mapping 去重为一个；只有明文和只有密文的映射都进入候选。
	assert.ElementsMatch(t, []string{bothPlain, onlyPlain}, cleanPairPaths(selection.FilePairs))
	assert.Equal(t, bothCipher, selection.FilePairs[cleanIndex(t, selection.FilePairs, bothPlain)].EncryptedPath)
	assert.Equal(t, filepath.Join(cfg.CurrentDir, "only.enc.yaml"), selection.FilePairs[cleanIndex(t, selection.FilePairs, onlyPlain)].EncryptedPath)

	// 模拟第一次 clean 后：删除明文，密文侧仍能发现映射（幂等 clean）。
	require.NoError(t, os.Remove(bothPlain))
	require.NoError(t, os.Remove(onlyPlain))
	again, err := ResolveSelection(cfg, SelectionOptions{Command: task.ModeClean})
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{
		filepath.Join(cfg.CurrentDir, "both.yaml"),
		filepath.Join(cfg.CurrentDir, "only.yaml"),
	}, cleanPairPaths(again.FilePairs))
}

func cleanIndex(t *testing.T, pairs []ResolvedFilePair, plaintext string) int {
	t.Helper()
	for i, pair := range pairs {
		if pair.PlaintextPath == plaintext {
			return i
		}
	}
	t.Fatalf("no pair with plaintext %s", plaintext)
	return -1
}

func TestCleanScopeAndGlobUseLogicalPlaintextPath(t *testing.T) {
	cfg := selectionConfig(t)
	cfg.Encryption.Groups = []GroupConfig{{ConfigDir: cfg.CurrentDir, Patterns: []string{"nested/*.yaml"}}}
	cipher := selectionFile(t, cfg.CurrentDir, "nested/gone.enc.yaml")
	logical := filepath.Join(cfg.CurrentDir, "nested", "gone.yaml")

	// 密文推导出的 logical plaintext 在 nested/ 下：cwd scope 应选中。
	byScope, err := ResolveSelection(cfg, SelectionOptions{Command: task.ModeClean})
	require.NoError(t, err)
	assert.Equal(t, []string{logical}, cleanPairPaths(byScope.FilePairs))

	// glob 按明文路径匹配，能命中只剩密文的映射。
	byGlob, err := ResolveSelection(cfg, SelectionOptions{Command: task.ModeClean, Targets: []string{"nested/*.yaml"}})
	require.NoError(t, err)
	assert.Equal(t, []string{logical}, cleanPairPaths(byGlob.FilePairs))

	// 密文精确路径同样命中。
	byCipher, err := ResolveSelection(cfg, SelectionOptions{Command: task.ModeClean, Targets: []string{cipher}})
	require.NoError(t, err)
	assert.Equal(t, []string{logical}, cleanPairPaths(byCipher.FilePairs))
}

func TestCleanExplicitMappingSurvivesWithoutEitherSide(t *testing.T) {
	cfg := selectionConfig(t)
	cfg.Encryption.Files = []FilePair{{
		PlaintextPath: filepath.Join(cfg.CurrentDir, "vanish.yaml"),
		EncryptedPath: filepath.Join(cfg.CurrentDir, "vanish.enc.yaml"),
		Format:        "yaml",
	}}
	selection, err := ResolveSelection(cfg, SelectionOptions{Command: task.ModeClean})
	require.NoError(t, err)
	assert.Len(t, selection.FilePairs, 1)
}

func TestCleanExplicitMappingOverridesGroup(t *testing.T) {
	cfg := selectionConfig(t)
	selectionFile(t, cfg.CurrentDir, "config.yaml")
	explicit := FilePair{
		PlaintextPath: filepath.Join(cfg.CurrentDir, "custom.yaml"),
		EncryptedPath: filepath.Join(cfg.CurrentDir, "config.enc.yaml"),
		Format:        "yaml",
	}
	cfg.Encryption.Groups = []GroupConfig{{ConfigDir: cfg.CurrentDir, Patterns: []string{"*.yaml"}}}
	cfg.Encryption.Files = []FilePair{explicit}
	selection, err := ResolveSelection(cfg, SelectionOptions{Command: task.ModeClean})
	require.NoError(t, err)
	assert.Equal(t, []string{explicit.PlaintextPath}, cleanPairPaths(selection.FilePairs))
}

func TestCleanOutsideProjectMappingSelectedExplicitlyButNotByScope(t *testing.T) {
	cfg := selectionConfig(t)
	outside := t.TempDir()
	outsidePlain := selectionFile(t, outside, "outside.yaml")
	cfg.Encryption.Files = []FilePair{{
		PlaintextPath: outsidePlain,
		EncryptedPath: filepath.Join(outside, "outside.enc.yaml"),
		Format:        "yaml",
	}}

	_, err := ResolveSelection(cfg, SelectionOptions{Command: task.ModeClean})
	require.ErrorContains(t, err, "no configured file pairs selected for current directory scope")

	byTarget, err := ResolveSelection(cfg, SelectionOptions{Command: task.ModeClean, Targets: []string{outsidePlain}})
	require.NoError(t, err)
	assert.Equal(t, []string{outsidePlain}, cleanPairPaths(byTarget.FilePairs))
}

func TestCleanRejectsConflictingGroupMappings(t *testing.T) {
	cfg := selectionConfig(t)
	for _, name := range []string{"config.yaml", "config.yml", "config.enc.yaml"} {
		selectionFile(t, cfg.CurrentDir, name)
	}
	cfg.Encryption.Groups = []GroupConfig{
		{ConfigDir: cfg.CurrentDir, Patterns: []string{"*.yaml"}},
		{ConfigDir: cfg.CurrentDir, Patterns: []string{"*.yml"}},
	}
	_, err := ResolveSelection(cfg, SelectionOptions{Command: task.ModeClean})
	require.ErrorContains(t, err, "conflicting group file pairs for clean")
}
