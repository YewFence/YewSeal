package doctor

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)

	os.Stdout = w
	t.Cleanup(func() {
		os.Stdout = oldStdout
	})

	fn()

	require.NoError(t, w.Close())
	output, err := io.ReadAll(r)
	require.NoError(t, err)

	return string(output)
}
