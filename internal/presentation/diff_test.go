package presentation

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDiffColorAndChannels(t *testing.T) {
	plain := "--- config.yaml\n+++ config.enc.yaml (decrypted)\n@@\n-host: local\n+host: saved\n"
	for _, mode := range []string{"auto", "always", "never"} {
		var body, diagnostics bytes.Buffer
		out := New(&body, &diagnostics, false)
		enabled, err := ResolveDiffColor(mode, &body)
		require.NoError(t, err)
		require.Equal(t, mode == "always", enabled)
		require.NoError(t, out.Diff(plain, enabled))
		require.Empty(t, diagnostics.String())
		if enabled {
			require.Contains(t, body.String(), "\x1b[")
			require.Contains(t, body.String(), "-host: local")
		} else { require.Equal(t, plain, body.String()) }
	}
	_, err := ResolveDiffColor("invalid", nil)
	require.Error(t, err)
}
