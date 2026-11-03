package config

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGroupManagedFilePairsUsesConfigurationOwnership(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "service")
	rootConfig := filepath.Join(root, ".yewseal.toml")
	childConfig := filepath.Join(child, ".config", ".yewseal.toml")
	empty := filepath.Join(root, "empty")
	parentPair := ResolvedFilePair{ConfigPath: rootConfig, PlaintextPath: filepath.Join(child, "parent.yaml")}
	childPair := ResolvedFilePair{ConfigPath: childConfig, PlaintextPath: filepath.Join(child, "secret.yaml")}
	scopes := GroupManagedFilePairs([]ResolvedFilePair{parentPair, childPair}, []LoadedFile{
		{Path: childConfig, Dir: child},
		{Path: filepath.Join(empty, ".yewseal.toml"), Dir: empty},
		{Path: rootConfig, Dir: root},
	}, filepath.Join(child, "nested"))
	require.Equal(t, []ManagedFileScope{
		{Dir: root, FilePairs: []ResolvedFilePair{parentPair}},
		{Dir: empty},
		{Dir: child, FilePairs: []ResolvedFilePair{childPair}},
	}, scopes)
}
