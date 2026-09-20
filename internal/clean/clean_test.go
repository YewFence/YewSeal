package clean

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"filippo.io/age"
	"github.com/YewFence/YewSeal/internal/agekey"
	"github.com/YewFence/YewSeal/internal/sopsx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type cleanFixture struct {
	identity *age.X25519Identity
	bundle   agekey.IdentityBundle
	dir      string
}

func newCleanFixture(t *testing.T) cleanFixture {
	t.Helper()
	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	bundle, err := agekey.NewIdentityBundle([]string{identity.String()})
	require.NoError(t, err)
	return cleanFixture{identity: identity, bundle: bundle, dir: t.TempDir()}
}

// writeEncrypted 落盘一份由 fixture identity 加密的 yaml 密文并返回明文字节。
func (f cleanFixture) writeEncrypted(t *testing.T, name string, plain []byte) []byte {
	t.Helper()
	encData, err := sopsx.Encrypt(plain, "yaml", []string{f.identity.Recipient().String()})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(f.dir, name), encData, 0600))
	return plain
}

func (f cleanFixture) path(name string) string {
	return filepath.Join(f.dir, name)
}

func (f cleanFixture) writePlain(t *testing.T, name string, content []byte) string {
	t.Helper()
	path := f.path(name)
	require.NoError(t, os.WriteFile(path, content, 0600))
	return path
}

func TestInspectReportsMissingAndBrokenLinksAsAbsent(t *testing.T) {
	f := newCleanFixture(t)

	exists, err := Inspect(f.path("missing.yaml"))
	require.NoError(t, err)
	assert.False(t, exists)

	require.NoError(t, os.Symlink(f.path("nowhere.yaml"), f.path("broken.yaml")))
	exists, err = Inspect(f.path("broken.yaml"))
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestInspectRejectsLoopAndNonRegularTargets(t *testing.T) {
	f := newCleanFixture(t)

	loopA, loopB := f.path("loop-a"), f.path("loop-b")
	require.NoError(t, os.Symlink(loopB, loopA))
	require.NoError(t, os.Symlink(loopA, loopB))
	_, err := Inspect(loopA)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resolve plaintext path")

	require.NoError(t, os.Mkdir(f.path("adir"), 0755))
	_, err = Inspect(f.path("adir"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "plaintext target is not a regular file")
}

func TestProcessRemovesMatchingPlaintext(t *testing.T) {
	f := newCleanFixture(t)
	plain := f.writeEncrypted(t, "config.enc.yaml", []byte("token: value\n"))
	plainPath := f.writePlain(t, "config.yaml", plain)

	outcome, err := Process(plainPath, Options{
		EncryptedPath:  f.path("config.enc.yaml"),
		Format:         "yaml",
		IdentityBundle: f.bundle,
	})
	require.NoError(t, err)
	assert.Equal(t, Removed, outcome)
	assert.NoFileExists(t, plainPath)
	assert.FileExists(t, f.path("config.enc.yaml"))
}

func TestProcessRemovesDespitePermissionAndMtimeDifferences(t *testing.T) {
	f := newCleanFixture(t)
	plain := f.writeEncrypted(t, "config.enc.yaml", []byte("token: value\n"))
	plainPath := f.writePlain(t, "config.yaml", plain)
	past := time.Now().Add(-48 * time.Hour)
	require.NoError(t, os.Chtimes(plainPath, past, past))
	require.NoError(t, os.Chmod(plainPath, 0644))

	outcome, err := Process(plainPath, Options{
		EncryptedPath:  f.path("config.enc.yaml"),
		Format:         "yaml",
		IdentityBundle: f.bundle,
	})
	require.NoError(t, err)
	assert.Equal(t, Removed, outcome)
}

func TestProcessTreatsLayoutDifferenceAsDifferent(t *testing.T) {
	f := newCleanFixture(t)
	f.writeEncrypted(t, "config.enc.yaml", []byte("token: value\n"))
	plainPath := f.writePlain(t, "config.yaml", []byte("token:    value\n"))

	confirmed := false
	outcome, err := Process(plainPath, Options{
		EncryptedPath:  f.path("config.enc.yaml"),
		Format:         "yaml",
		IdentityBundle: f.bundle,
		ConfirmDifferent: func() (bool, error) {
			confirmed = true
			return false, nil
		},
	})
	require.NoError(t, err)
	assert.Equal(t, Retained, outcome)
	assert.True(t, confirmed)
	assert.FileExists(t, plainPath)
}

func TestProcessRetainsWithoutPromptUnderSkipDifferent(t *testing.T) {
	f := newCleanFixture(t)
	f.writeEncrypted(t, "config.enc.yaml", []byte("token: value\n"))
	plainPath := f.writePlain(t, "config.yaml", []byte("token: changed\n"))

	outcome, err := Process(plainPath, Options{
		EncryptedPath:  f.path("config.enc.yaml"),
		Format:         "yaml",
		IdentityBundle: f.bundle,
		SkipDifferent:  true,
		ConfirmDifferent: func() (bool, error) {
			t.Fatal("skip-different must not prompt")
			return false, nil
		},
	})
	require.NoError(t, err)
	assert.Equal(t, Retained, outcome)
}

func TestProcessForceRemovesDifferenceWithoutPrompt(t *testing.T) {
	f := newCleanFixture(t)
	f.writeEncrypted(t, "config.enc.yaml", []byte("token: value\n"))
	plainPath := f.writePlain(t, "config.yaml", []byte("token: changed\n"))

	outcome, err := Process(plainPath, Options{
		EncryptedPath:  f.path("config.enc.yaml"),
		Format:         "yaml",
		IdentityBundle: f.bundle,
		Force:          true,
		ConfirmDifferent: func() (bool, error) {
			t.Fatal("force must not prompt")
			return false, nil
		},
	})
	require.NoError(t, err)
	assert.Equal(t, Removed, outcome)
	assert.NoFileExists(t, plainPath)
}

func TestProcessPromptFailureIsUndecidedFailure(t *testing.T) {
	f := newCleanFixture(t)
	f.writeEncrypted(t, "config.enc.yaml", []byte("token: value\n"))
	plainPath := f.writePlain(t, "config.yaml", []byte("token: changed\n"))

	outcome, err := Process(plainPath, Options{
		EncryptedPath:  f.path("config.enc.yaml"),
		Format:         "yaml",
		IdentityBundle: f.bundle,
		ConfirmDifferent: func() (bool, error) {
			return false, errors.New("failed to read input: EOF")
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "EOF")
	assert.Empty(t, outcome)
	assert.FileExists(t, plainPath)
}

func TestProcessFailsWhenCiphertextMissingCorruptOrKeyMismatch(t *testing.T) {
	f := newCleanFixture(t)
	f.writeEncrypted(t, "config.enc.yaml", []byte("token: value\n"))
	plainPath := f.writePlain(t, "config.yaml", []byte("token: value\n"))

	_, err := Process(plainPath, Options{EncryptedPath: f.path("absent.enc.yaml"), Format: "yaml", IdentityBundle: f.bundle})
	require.Error(t, err)
	assert.FileExists(t, plainPath)

	require.NoError(t, os.WriteFile(f.path("corrupt.enc.yaml"), []byte("not: sops\n"), 0600))
	_, err = Process(plainPath, Options{EncryptedPath: f.path("corrupt.enc.yaml"), Format: "yaml", IdentityBundle: f.bundle})
	require.Error(t, err)
	assert.FileExists(t, plainPath)

	other, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	otherBundle, err := agekey.NewIdentityBundle([]string{other.String()})
	require.NoError(t, err)
	_, err = Process(plainPath, Options{EncryptedPath: f.path("config.enc.yaml"), Format: "yaml", IdentityBundle: otherBundle})
	require.Error(t, err)
	assert.ErrorIs(t, err, sopsx.ErrNoMatchingIdentity)
	assert.FileExists(t, plainPath)
}

func TestProcessRemovesSymlinkTargetAndKeepsLinks(t *testing.T) {
	f := newCleanFixture(t)
	plain := f.writeEncrypted(t, "config.enc.yaml", []byte("token: value\n"))
	require.NoError(t, os.MkdirAll(f.path("real"), 0755))
	final := f.writePlain(t, "real/config.yaml", plain)
	require.NoError(t, os.MkdirAll(f.path("link"), 0755))
	middle := f.path("link/config.yaml")
	require.NoError(t, os.Symlink(final, middle))
	logical := f.path("config.yaml")
	require.NoError(t, os.Symlink(middle, logical))

	outcome, err := Process(logical, Options{
		EncryptedPath:  f.path("config.enc.yaml"),
		Format:         "yaml",
		IdentityBundle: f.bundle,
	})
	require.NoError(t, err)
	assert.Equal(t, Removed, outcome)
	assert.NoFileExists(t, final)
	linkInfo, err := os.Lstat(logical)
	require.NoError(t, err)
	assert.Equal(t, os.ModeSymlink, linkInfo.Mode()&os.ModeSymlink)
	middleInfo, err := os.Lstat(middle)
	require.NoError(t, err)
	assert.Equal(t, os.ModeSymlink, middleInfo.Mode()&os.ModeSymlink)
}

func TestProcessRetargetedSymlinkFailsRecheck(t *testing.T) {
	f := newCleanFixture(t)
	f.writeEncrypted(t, "config.enc.yaml", []byte("token: value\n"))
	first := f.writePlain(t, "first.yaml", []byte("token: changed\n"))
	second := f.writePlain(t, "second.yaml", []byte("token: value\n"))
	logical := f.path("config.yaml")
	require.NoError(t, os.Symlink(first, logical))

	outcome, err := Process(logical, Options{
		EncryptedPath:  f.path("config.enc.yaml"),
		Format:         "yaml",
		IdentityBundle: f.bundle,
		ConfirmDifferent: func() (bool, error) {
			require.NoError(t, os.Remove(logical))
			require.NoError(t, os.Symlink(second, logical))
			return true, nil
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "plaintext changed before removal")
	assert.Empty(t, outcome)
	assert.FileExists(t, first)
	assert.FileExists(t, second)
}

func TestProcessModifiedPlaintextFailsRecheck(t *testing.T) {
	f := newCleanFixture(t)
	f.writeEncrypted(t, "config.enc.yaml", []byte("token: value\n"))
	plainPath := f.writePlain(t, "config.yaml", []byte("token: changed\n"))

	outcome, err := Process(plainPath, Options{
		EncryptedPath:  f.path("config.enc.yaml"),
		Format:         "yaml",
		IdentityBundle: f.bundle,
		ConfirmDifferent: func() (bool, error) {
			require.NoError(t, os.WriteFile(plainPath, []byte("token: saved\n"), 0600))
			return true, nil
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "plaintext changed before removal")
	assert.Empty(t, outcome)
	content, err := os.ReadFile(plainPath)
	require.NoError(t, err)
	assert.Equal(t, "token: saved\n", string(content))
}

func TestProcessChangedCiphertextFailsRecheck(t *testing.T) {
	f := newCleanFixture(t)
	f.writeEncrypted(t, "config.enc.yaml", []byte("token: original\n"))
	plainPath := f.writePlain(t, "config.yaml", []byte("token: changed\n"))

	outcome, err := Process(plainPath, Options{
		EncryptedPath:  f.path("config.enc.yaml"),
		Format:         "yaml",
		IdentityBundle: f.bundle,
		ConfirmDifferent: func() (bool, error) {
			f.writeEncrypted(t, "config.enc.yaml", []byte("token: replaced\n"))
			return true, nil
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ciphertext changed before removal")
	assert.Empty(t, outcome)
	assert.FileExists(t, plainPath)
}

func TestProcessRemoveFailureRetainsFile(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	f := newCleanFixture(t)
	plain := f.writeEncrypted(t, "config.enc.yaml", []byte("token: value\n"))
	locked := f.path("locked")
	require.NoError(t, os.Mkdir(locked, 0700))
	plainPath := filepath.Join(locked, "config.yaml")
	require.NoError(t, os.WriteFile(plainPath, plain, 0600))
	require.NoError(t, os.Chmod(locked, 0500))
	t.Cleanup(func() { _ = os.Chmod(locked, 0700) })

	outcome, err := Process(plainPath, Options{
		EncryptedPath:  f.path("config.enc.yaml"),
		Format:         "yaml",
		IdentityBundle: f.bundle,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to remove plaintext file")
	assert.Empty(t, outcome)
}

func TestProcessMissingPlaintextIsAlreadyAbsentWithoutIdentity(t *testing.T) {
	f := newCleanFixture(t)
	f.writeEncrypted(t, "config.enc.yaml", []byte("token: value\n"))

	outcome, err := Process(f.path("config.yaml"), Options{EncryptedPath: f.path("config.enc.yaml"), Format: "yaml"})
	require.NoError(t, err)
	assert.Equal(t, AlreadyAbsent, outcome)
}
