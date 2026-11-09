package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProbeFileDistinguishesUnknownFromAbsent(t *testing.T) {
	root := t.TempDir()
	existing := filepath.Join(root, "config.toml")
	require.NoError(t, os.WriteFile(existing, []byte("x"), 0600))

	t.Run("existing file", func(t *testing.T) {
		require.NotNil(t, probeFile(existing))
		require.True(t, *probeFile(existing))
	})
	t.Run("definitively absent", func(t *testing.T) {
		missing := probeFile(filepath.Join(root, "absent.toml"))
		require.NotNil(t, missing)
		require.False(t, *missing)
	})
	t.Run("directory", func(t *testing.T) {
		dir := probeFile(root)
		require.NotNil(t, dir)
		require.False(t, *dir)
	})
	t.Run("undeterminable presence", func(t *testing.T) {
		// A path through a regular file fails with ENOTDIR, not absence:
		// the marker must not claim the file is missing.
		require.Nil(t, probeFile(filepath.Join(existing, "nested.toml")))
	})
}
