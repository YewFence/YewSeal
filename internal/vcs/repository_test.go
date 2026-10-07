package vcs

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDiscoveryFilesDoesNotQueryHistory(t *testing.T) {
	adapter := &countingAdapter{}
	repository := &Repository{root: t.TempDir(), adapter: adapter}

	files, err := repository.DiscoveryFiles()
	require.NoError(t, err)
	require.Equal(t, []string{"a/.yewseal.toml", "z/.yewseal.toml"}, files)
	require.Equal(t, 1, adapter.discoveryCalls)
	require.Zero(t, adapter.historyCalls)
	require.Zero(t, adapter.pendingCalls)
}

type countingAdapter struct {
	discoveryCalls int
	historyCalls   int
	pendingCalls   int
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

func (a *countingAdapter) historyFiles() (map[string]bool, error) {
	a.historyCalls++
	return nil, nil
}

func (a *countingAdapter) pendingFiles() (map[string]bool, error) {
	a.pendingCalls++
	return nil, nil
}
