package task

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGroupDiscoverySharesOnePrunedTraversalAcrossSides(t *testing.T) {
	for _, firstMode := range []string{ModeEncrypt, ModeDecrypt, ModePlan, ModeClean} {
		t.Run(firstMode, func(t *testing.T) {
			root := t.TempDir()
			for _, name := range []string{"app.yml", "app.enc.yaml", "history.enc.yaml", "ignored/deep/broken.unknown", ".git/objects/broken.unknown", ".jj/store/broken.unknown"} {
				path := filepath.Join(root, filepath.FromSlash(name))
				require.NoError(t, os.MkdirAll(filepath.Dir(path), 0700))
				require.NoError(t, os.WriteFile(path, []byte("fixture"), 0600))
			}
			var stats ScanStats
			discovery := NewGroupDiscovery(GroupOptions{
				Root: root, Patterns: []string{"*"}, Exclude: []string{"ignored/"}, Stats: &stats,
			})
			_, err := discovery.FilePairs(firstMode)
			require.NoError(t, err)
			wantStats := ScanStats{Entries: 7, Directories: 1}
			require.Equal(t, wantStats, stats)

			for range 2 {
				plain, err := discovery.FilePairs(ModeEncrypt)
				require.NoError(t, err)
				require.Equal(t, []FilePair{{PlaintextPath: filepath.Join(root, "app.yml"), EncryptedPath: filepath.Join(root, "app.enc.yaml"), Format: "yaml"}}, plain)
				encrypted, err := discovery.FilePairs(ModeDecrypt)
				require.NoError(t, err)
				require.Equal(t, []FilePair{
					{PlaintextPath: filepath.Join(root, "app.yaml"), EncryptedPath: filepath.Join(root, "app.enc.yaml"), Format: "yaml"},
					{PlaintextPath: filepath.Join(root, "history.yaml"), EncryptedPath: filepath.Join(root, "history.enc.yaml"), Format: "yaml"},
				}, encrypted)
				both, err := discovery.FilePairs(ModePlan)
				require.NoError(t, err)
				require.Equal(t, []FilePair{plain[0], encrypted[1]}, both)
				require.Equal(t, wantStats, stats)
			}
		})
	}
}

func TestGroupDiscoveryDefersPlaintextInterpretation(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "app.enc.yaml"), []byte("fixture"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "broken.unknown"), []byte("fixture"), 0600))
	var stats ScanStats
	discovery := NewGroupDiscovery(GroupOptions{Root: root, Patterns: []string{"*"}, Stats: &stats})
	encrypted, err := discovery.FilePairs(ModeDecrypt)
	require.NoError(t, err)
	require.Len(t, encrypted, 1)
	require.Equal(t, filepath.Join(root, "app.yaml"), encrypted[0].PlaintextPath)
	_, err = discovery.FilePairs(ModeEncrypt)
	require.ErrorContains(t, err, "could not detect format")
	again, err := discovery.FilePairs(ModeDecrypt)
	require.NoError(t, err)
	require.Equal(t, encrypted, again)
	require.Equal(t, ScanStats{Entries: 3, Directories: 1}, stats)
}

func TestGroupDiscoveryReusesEmptySidesAndRefreshesOnNextPreparation(t *testing.T) {
	root := t.TempDir()
	var stats ScanStats
	opts := GroupOptions{Root: root, Patterns: []string{"*.yaml"}, Stats: &stats}
	discovery := NewGroupDiscovery(opts)
	pairs, err := discovery.FilePairs(ModeEncrypt)
	require.NoError(t, err)
	require.Empty(t, pairs)
	require.NoError(t, os.WriteFile(filepath.Join(root, "new.enc.yaml"), []byte("fixture"), 0600))
	for _, mode := range []string{ModeDecrypt, ModePlan, ModeEncrypt} {
		pairs, err := discovery.FilePairs(mode)
		require.NoError(t, err)
		require.Empty(t, pairs)
	}
	require.Equal(t, ScanStats{Entries: 1, Directories: 1}, stats)
	pairs, err = NewGroupDiscovery(opts).FilePairs(ModePlan)
	require.NoError(t, err)
	require.Len(t, pairs, 1)
	require.Equal(t, filepath.Join(root, "new.yaml"), pairs[0].PlaintextPath)
	require.Equal(t, ScanStats{Entries: 3, Directories: 2}, stats)
}
