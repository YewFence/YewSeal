package sync

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/YewFence/YewSeal/internal/errx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fakeExecutablePath(dir, name string) string {
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(dir, name)
}

func writeFakeExecutable(t *testing.T, dir, name string) string {
	t.Helper()

	path := fakeExecutablePath(dir, name)
	require.NoError(t, os.WriteFile(path, []byte("fake"), 0o755))
	return path
}

func withWorkingDir(t *testing.T, dir string) {
	t.Helper()

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() {
		_ = os.Chdir(oldWd)
	})
}

func TestInfisicalProviderName(t *testing.T) {
	provider := &InfisicalProvider{}
	assert.Equal(t, "infisical", provider.Name())
}

func TestGetProvider(t *testing.T) {
	t.Run("known provider", func(t *testing.T) {
		provider, err := GetProvider("infisical")
		require.NoError(t, err)
		assert.IsType(t, &InfisicalProvider{}, provider)
	})

	t.Run("unknown provider", func(t *testing.T) {
		provider, err := GetProvider("vault")
		assert.Nil(t, provider)

		var providerErr *errx.UnknownProviderError
		require.ErrorAs(t, err, &providerErr)
		assert.Equal(t, "vault", providerErr.Name)
		assert.Equal(t, []string{"infisical"}, providerErr.Supported)
	})
}

func TestSyncKey_MissingKeyFile(t *testing.T) {
	err := SyncKey("infisical", filepath.Join(t.TempDir(), "missing-key.txt"), "secret", "")

	var keyErr *errx.KeyFileNotFoundError
	require.ErrorAs(t, err, &keyErr)
	assert.Contains(t, keyErr.Path, "missing-key.txt")
}

func TestPullKey_UnknownProvider(t *testing.T) {
	err := PullKey("vault", "ignored.txt", "secret", "")

	var providerErr *errx.UnknownProviderError
	require.ErrorAs(t, err, &providerErr)
	assert.Equal(t, "vault", providerErr.Name)
}

func TestInfisicalProviderCheck_MissingBinary(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	provider := &InfisicalProvider{}
	err := provider.Check()

	var depErr *errx.MissingDependencyError
	require.ErrorAs(t, err, &depErr)
	assert.Equal(t, "infisical CLI", depErr.Name)
}

func TestInfisicalProviderCheck_MissingProjectConfig(t *testing.T) {
	tempDir := t.TempDir()
	withWorkingDir(t, tempDir)
	writeFakeExecutable(t, tempDir, "infisical")
	t.Setenv("PATH", tempDir)

	provider := &InfisicalProvider{}
	err := provider.Check()

	var configErr *errx.MissingProjectConfigError
	require.ErrorAs(t, err, &configErr)
	assert.Equal(t, ".infisical.json", configErr.Path)
	assert.Equal(t, "Run 'infisical init' first to initialize the project", configErr.Hint)
}

func TestSyncKey_PropagatesProviderCheckError(t *testing.T) {
	tempDir := t.TempDir()
	withWorkingDir(t, tempDir)
	writeFakeExecutable(t, tempDir, "infisical")
	t.Setenv("PATH", tempDir)

	keyFile := filepath.Join(tempDir, "keys.txt")
	require.NoError(t, os.WriteFile(keyFile, []byte("AGE-SECRET-KEY-1TEST"), 0o600))

	err := SyncKey("infisical", keyFile, "secret", "")

	var configErr *errx.MissingProjectConfigError
	require.ErrorAs(t, err, &configErr)
}

func TestPullKey_PropagatesProviderCheckError(t *testing.T) {
	tempDir := t.TempDir()
	withWorkingDir(t, tempDir)
	writeFakeExecutable(t, tempDir, "infisical")
	t.Setenv("PATH", tempDir)

	err := PullKey("infisical", filepath.Join(tempDir, "keys.txt"), "secret", "")

	var configErr *errx.MissingProjectConfigError
	require.ErrorAs(t, err, &configErr)
}

func TestSyncKey_ErrorTypeStaysStable(t *testing.T) {
	err := SyncKey("infisical", filepath.Join(t.TempDir(), "missing.txt"), "secret", "")
	assert.True(t, errors.As(err, new(*errx.KeyFileNotFoundError)))
}
