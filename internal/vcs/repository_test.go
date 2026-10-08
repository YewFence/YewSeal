package vcs

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDiscoveryFilesDoesNotQuerySnapshot(t *testing.T) {
	adapter := &countingAdapter{}
	repository := &Repository{root: t.TempDir(), adapter: adapter}

	files, err := repository.DiscoveryFiles()
	require.NoError(t, err)
	require.Equal(t, []string{"a/.yewseal.toml", "z/.yewseal.toml"}, files)
	require.Equal(t, 1, adapter.discoveryCalls)
	require.Zero(t, adapter.snapshotCalls)
}

func TestSnapshotQueriesAdapterOnce(t *testing.T) {
	adapter := &countingAdapter{}
	repository := &Repository{root: t.TempDir(), adapter: adapter}

	snapshot, err := repository.Snapshot()
	require.NoError(t, err)
	require.Equal(t, History, snapshot.Status(filepath.Join(repository.root, "tracked")))
	require.Equal(t, Pending, snapshot.Status(filepath.Join(repository.root, "pending")))
	require.Equal(t, 1, adapter.snapshotCalls)
}

type countingAdapter struct {
	discoveryCalls int
	snapshotCalls  int
}

func (a *countingAdapter) name() string {
	return "test"
}

func (a *countingAdapter) discoveryFiles() (map[string]bool, error) {
	a.discoveryCalls++
	return map[string]bool{
		"z/.yewseal.toml": true,
		"a/.yewseal.toml": true,
	}, nil
}

func (a *countingAdapter) snapshot() (map[string]bool, map[string]bool, error) {
	a.snapshotCalls++
	return map[string]bool{"tracked": true}, map[string]bool{"pending": true}, nil
}
