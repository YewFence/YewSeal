package config

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGroupManagedFilePairsFallsBackToMappingSource(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "service")
	unattributed := ResolvedFilePair{PlaintextPath: filepath.Join(root, "root.yaml")}
	relativeSource := ResolvedFilePair{ConfigPath: filepath.Join("service", ".yewseal.toml"), PlaintextPath: filepath.Join(child, "relative.yaml")}
	absoluteSource := ResolvedFilePair{ConfigPath: filepath.Join(child, ".yewseal.toml"), PlaintextPath: filepath.Join(child, "absolute.yaml")}
	require.Equal(t, []ManagedFileScope{
		{Dir: root, FilePairs: []ResolvedFilePair{unattributed}},
		{Dir: child, FilePairs: []ResolvedFilePair{relativeSource, absoluteSource}},
	}, GroupManagedFilePairs([]ResolvedFilePair{unattributed, relativeSource, absoluteSource}, nil, root))
}

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
