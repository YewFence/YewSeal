package seal

import (
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

func TestWriteDeliveredFilePropagatesWriteFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secret.yaml")
	var original unix.Rlimit
	require.NoError(t, unix.Getrlimit(unix.RLIMIT_FSIZE, &original))
	ignored := signal.Ignored(syscall.SIGXFSZ)
	signal.Ignore(syscall.SIGXFSZ)
	// The process-wide limit must be restored before coverage output is flushed.
	t.Cleanup(func() {
		require.NoError(t, unix.Setrlimit(unix.RLIMIT_FSIZE, &original))
		if !ignored {
			signal.Reset(syscall.SIGXFSZ)
		}
	})
	limited := original
	limited.Cur = 1
	require.NoError(t, unix.Setrlimit(unix.RLIMIT_FSIZE, &limited))
	err := writeDeliveredFile(path, []byte("token: secret\n"))
	require.ErrorContains(t, err, "failed to write delivery file")
	require.ErrorIs(t, err, unix.EFBIG)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, []byte("t"), data)
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0600), info.Mode().Perm())
}
