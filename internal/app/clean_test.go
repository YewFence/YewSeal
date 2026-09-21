package app

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/presentation"
	"github.com/YewFence/YewSeal/internal/sopsx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type trackedReader struct {
	reader io.Reader
	read   bool
}

func (t *trackedReader) Read(p []byte) (int, error) {
	t.read = true
	return t.reader.Read(p)
}

type failingWriter struct{}

func (failingWriter) Write(p []byte) (int, error) { return 0, errors.New("channel closed") }

// prepareCleanFixture 建立两个已登记映射：same.yaml 与密文一致，
// wip.yaml 与密文不同，密文均由 env 的 key 加密。
func prepareCleanFixture(t *testing.T) *config.Config {
	t.Helper()
	env := newAppCryptoTestEnv(t)
	recipient := env.publicKey

	sameCipher, err := sopsx.Encrypt([]byte("token: stable\n"), "yaml", []string{recipient})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile("same.enc.yaml", sameCipher, 0600))
	require.NoError(t, os.WriteFile("same.yaml", []byte("token: stable\n"), 0600))

	wipCipher, err := sopsx.Encrypt([]byte("token: original\n"), "yaml", []string{recipient})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile("wip.enc.yaml", wipCipher, 0600))
	require.NoError(t, os.WriteFile("wip.yaml", []byte("token: wip\n"), 0600))

	return configWithOwnerRecipient(&config.Config{Encryption: config.EncryptionConfig{Files: []config.FilePair{
		{PlaintextPath: "same.yaml", EncryptedPath: "same.enc.yaml", Format: "yaml"},
		{PlaintextPath: "wip.yaml", EncryptedPath: "wip.enc.yaml", Format: "yaml"},
	}}}, recipient)
}

func TestCleanFilesDefaultRemovesSameAndPromptsForDifference(t *testing.T) {
	cfg := prepareCleanFixture(t)

	var stdout, stderr bytes.Buffer
	input := &trackedReader{reader: strings.NewReader("\nn\n")}
	err := CleanFiles(cfg, CleanRequest{
		Presentation: presentation.New(&stdout, &stderr, false),
		Input:        input,
		KeyFile:      ".age/keys.txt",
	})
	require.NoError(t, err)
	assert.Zero(t, stdout.Len(), "clean must keep stdout empty")
	assert.NoFileExists(t, "same.yaml")
	assert.FileExists(t, "wip.yaml")
	assert.Contains(t, stderr.String(), "REMOVED same.yaml")
	assert.Contains(t, stderr.String(), "RETAINED wip.yaml: plaintext differs from encrypted content")
	assert.Contains(t, stderr.String(), "Hint: run yews diff -- 'wip.yaml'")
	assert.Contains(t, stderr.String(), "Summary (cleaned): 1 removed, 0 already absent, 1 retained, 0 failed (2 selected)")
}

func TestCleanFilesDefaultConfirmYesRemovesDifference(t *testing.T) {
	cfg := prepareCleanFixture(t)

	var stdout, stderr bytes.Buffer
	err := CleanFiles(cfg, CleanRequest{
		Presentation: presentation.New(&stdout, &stderr, false),
		Input:        strings.NewReader("y\n"),
		KeyFile:      ".age/keys.txt",
	})
	require.NoError(t, err)
	assert.NoFileExists(t, "same.yaml")
	assert.NoFileExists(t, "wip.yaml")
	assert.Contains(t, stderr.String(), "REMOVED wip.yaml")
}

func TestCleanFilesMatchingBatchNeverReadsStdin(t *testing.T) {
	cfg := prepareCleanFixture(t)
	require.NoError(t, os.Remove("wip.yaml"))

	var stdout, stderr bytes.Buffer
	input := &trackedReader{reader: strings.NewReader("")}
	err := CleanFiles(cfg, CleanRequest{
		Presentation: presentation.New(&stdout, &stderr, false),
		Input:        input,
		KeyFile:      ".age/keys.txt",
	})
	require.NoError(t, err)
	assert.False(t, input.read, "matching batch must not read stdin")
	assert.NoFileExists(t, "same.yaml")
	assert.Contains(t, stderr.String(), "1 removed, 1 already absent")
}

func TestCleanFilesPromptEOFFailsAndRetains(t *testing.T) {
	cfg := prepareCleanFixture(t)

	var stdout, stderr bytes.Buffer
	err := CleanFiles(cfg, CleanRequest{
		Presentation: presentation.New(&stdout, &stderr, false),
		Input:        strings.NewReader(""),
		KeyFile:      ".age/keys.txt",
	})
	require.Error(t, err)
	assert.NoFileExists(t, "same.yaml", "already verified removal stays removed")
	assert.FileExists(t, "wip.yaml")
	assert.Contains(t, stderr.String(), "FAILED wip.yaml")
}

func TestCleanFilesNilInputFailsPromptAndRetains(t *testing.T) {
	cfg := prepareCleanFixture(t)

	var stdout, stderr bytes.Buffer
	err := CleanFiles(cfg, CleanRequest{
		Presentation: presentation.New(&stdout, &stderr, false),
		KeyFile:      ".age/keys.txt",
		Targets:      []string{"wip.yaml"},
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, io.EOF)
	assert.FileExists(t, "wip.yaml")
	assert.Contains(t, stderr.String(), "FAILED wip.yaml")
}

func TestCleanFilesSkipDifferentNeverReadsStdin(t *testing.T) {
	cfg := prepareCleanFixture(t)

	var stdout, stderr bytes.Buffer
	input := &trackedReader{reader: strings.NewReader("")}
	err := CleanFiles(cfg, CleanRequest{
		Presentation:  presentation.New(&stdout, &stderr, false),
		Input:         input,
		KeyFile:       ".age/keys.txt",
		SkipDifferent: true,
	})
	require.NoError(t, err)
	assert.False(t, input.read, "skip-different must not read stdin")
	assert.NoFileExists(t, "same.yaml")
	assert.FileExists(t, "wip.yaml")
}

func TestCleanFilesRemoveDifferentDoesNotReadStdin(t *testing.T) {
	cfg := prepareCleanFixture(t)

	var stdout, stderr bytes.Buffer
	input := &trackedReader{reader: strings.NewReader("")}
	err := CleanFiles(cfg, CleanRequest{
		Presentation:    presentation.New(&stdout, &stderr, false),
		Input:           input,
		KeyFile:         ".age/keys.txt",
		RemoveDifferent: true,
	})
	require.NoError(t, err)
	assert.False(t, input.read, "remove-different must not read stdin")
	assert.NoFileExists(t, "same.yaml")
	assert.NoFileExists(t, "wip.yaml")
}

func TestCleanFilesRemoveDifferentDoesNotBypassRecoverability(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	require.NoError(t, os.WriteFile("gone.yaml", []byte("token: value\n"), 0600))
	require.NoError(t, os.WriteFile("broken.yaml", []byte("token: value\n"), 0600))
	require.NoError(t, os.WriteFile("broken.enc.yaml", []byte("corrupt: yes\n"), 0600))
	cfg := configWithOwnerRecipient(&config.Config{Encryption: config.EncryptionConfig{Files: []config.FilePair{
		{PlaintextPath: "gone.yaml", EncryptedPath: "gone.enc.yaml", Format: "yaml"},
		{PlaintextPath: "broken.yaml", EncryptedPath: "broken.enc.yaml", Format: "yaml"},
	}}}, env.publicKey)

	var stdout, stderr bytes.Buffer
	err := CleanFiles(cfg, CleanRequest{
		Presentation:    presentation.New(&stdout, &stderr, false),
		KeyFile:         ".age/keys.txt",
		RemoveDifferent: true,
	})
	require.Error(t, err)
	assert.FileExists(t, "gone.yaml")
	assert.FileExists(t, "broken.yaml")
	assert.Contains(t, stderr.String(), "FAILED gone.yaml: input file ")
	assert.Contains(t, stderr.String(), "gone.enc.yaml does not exist")
	assert.Contains(t, stderr.String(), "FAILED broken.yaml")
	assert.Contains(t, stderr.String(), "0 removed, 0 already absent, 0 retained, 2 failed (2 selected)")
}

func TestCleanFilesForceIgnoresRecoveryInputs(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	require.NoError(t, os.WriteFile("secret.yaml", []byte("unsaved: local\n"), 0600))
	cfg := configWithOwnerRecipient(&config.Config{Encryption: config.EncryptionConfig{Files: []config.FilePair{
		{PlaintextPath: "secret.yaml", EncryptedPath: "missing.enc.yaml", Format: "yaml"},
	}}}, env.publicKey)
	input := &trackedReader{reader: strings.NewReader("")}

	var stderr bytes.Buffer
	err := CleanFiles(cfg, CleanRequest{
		Presentation: presentation.New(nil, &stderr, false),
		Input:        input,
		KeyFile:      "missing-keys.txt",
		Force:        true,
	})
	require.NoError(t, err)
	assert.False(t, input.read, "force must not read stdin")
	assert.NoFileExists(t, "secret.yaml")
	assert.Contains(t, stderr.String(), "REMOVED secret.yaml")
}

func TestCleanFilesMixedBatchContinuesAndSummarizes(t *testing.T) {
	cfg := prepareCleanFixture(t)
	require.NoError(t, os.Remove("wip.enc.yaml"))
	require.NoError(t, os.WriteFile("extra.yaml", []byte("token: extra\n"), 0600))
	cfg.Encryption.Files = append(cfg.Encryption.Files, config.FilePair{PlaintextPath: "extra.yaml", EncryptedPath: "extra.enc.yaml", Format: "yaml"})

	var stdout, stderr bytes.Buffer
	err := CleanFiles(cfg, CleanRequest{
		Presentation:  presentation.New(&stdout, &stderr, false),
		Input:         strings.NewReader(""),
		KeyFile:       ".age/keys.txt",
		SkipDifferent: true,
	})
	require.Error(t, err)
	assert.Zero(t, stdout.Len())
	assert.NoFileExists(t, "same.yaml")
	assert.FileExists(t, "wip.yaml")
	assert.FileExists(t, "extra.yaml")
	assert.Contains(t, stderr.String(), "2 failed")
}

func TestCleanFilesKeepsStableSelectionOrder(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	var pairs []config.FilePair
	for _, name := range []string{"zeta.yaml", "alpha.yaml", "mid.yaml"} {
		cipher, err := sopsx.Encrypt([]byte("token: value\n"), "yaml", []string{env.publicKey})
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(name, []byte("token: value\n"), 0600))
		require.NoError(t, os.WriteFile(strings.Replace(name, ".yaml", ".enc.yaml", 1), cipher, 0600))
		pairs = append(pairs, config.FilePair{PlaintextPath: name, EncryptedPath: strings.Replace(name, ".yaml", ".enc.yaml", 1), Format: "yaml"})
	}
	cfg := configWithOwnerRecipient(&config.Config{Encryption: config.EncryptionConfig{Files: pairs}}, env.publicKey)

	var stdout, stderr bytes.Buffer
	require.NoError(t, CleanFiles(cfg, CleanRequest{
		Presentation: presentation.New(&stdout, &stderr, false),
		KeyFile:      ".age/keys.txt",
	}))
	assert.Regexp(t, `(?s)REMOVED zeta\.yaml.*REMOVED alpha\.yaml.*REMOVED mid\.yaml`, stderr.String())
}

func TestCleanFilesAllAbsentRequiresNoIdentity(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	cipher, err := sopsx.Encrypt([]byte("token: value\n"), "yaml", []string{env.publicKey})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile("gone.enc.yaml", cipher, 0600))
	cfg := configWithOwnerRecipient(&config.Config{Encryption: config.EncryptionConfig{Files: []config.FilePair{
		{PlaintextPath: "gone.yaml", EncryptedPath: "gone.enc.yaml", Format: "yaml"},
	}}}, env.publicKey)

	var stdout, stderr bytes.Buffer
	err = CleanFiles(cfg, CleanRequest{
		Presentation: presentation.New(&stdout, &stderr, false),
		KeyFile:      "definitely-missing-keys.txt",
	})
	require.NoError(t, err)
	assert.Contains(t, stderr.String(), "0 removed, 1 already absent, 0 retained, 0 failed (1 selected)")
}

func TestCleanFilesExistingPlaintextLoadsIdentityBeforeCiphertext(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	require.NoError(t, os.WriteFile("secret.yaml", []byte("token: value\n"), 0600))
	cfg := configWithOwnerRecipient(&config.Config{Encryption: config.EncryptionConfig{Files: []config.FilePair{
		{PlaintextPath: "secret.yaml", EncryptedPath: "missing.enc.yaml", Format: "yaml"},
	}}}, env.publicKey)

	err := CleanFiles(cfg, CleanRequest{KeyFile: "missing-keys.txt"})
	require.ErrorContains(t, err, "failed to read Age key file missing-keys.txt")
	assert.FileExists(t, "secret.yaml")
}

func TestCleanFilesWithoutIdentityFailsAndKeepsPlaintext(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	plain := []byte("token: value\n")
	cipher, err := sopsx.Encrypt(plain, "yaml", []string{env.publicKey})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile("secret.yaml", plain, 0600))
	require.NoError(t, os.WriteFile("secret.enc.yaml", cipher, 0600))
	require.NoError(t, os.Remove(env.keyFile))
	for _, name := range []string{"YEWSEAL_AGE_IDENTITIES", "SOPS_AGE_KEY", "SOPS_AGE_KEY_FILE", "YEWSEAL_AGE_KEY_CMD", "SOPS_AGE_KEY_CMD"} {
		t.Setenv(name, "")
	}
	cfg := configWithOwnerRecipient(&config.Config{Encryption: config.EncryptionConfig{Files: []config.FilePair{
		{PlaintextPath: "secret.yaml", EncryptedPath: "secret.enc.yaml", Format: "yaml"},
	}}}, env.publicKey)

	var stderr bytes.Buffer
	err = CleanFiles(cfg, CleanRequest{Presentation: presentation.New(nil, &stderr, false)})
	require.Error(t, err)
	assert.FileExists(t, "secret.yaml")
	assert.Contains(t, stderr.String(), "FAILED secret.yaml: no age identity is available")
	assert.Contains(t, stderr.String(), "0 removed, 0 already absent, 0 retained, 1 failed (1 selected)")
}

func TestCleanFilesVerbosePrintsAlreadyAbsent(t *testing.T) {
	cfg := prepareCleanFixture(t)
	require.NoError(t, os.Remove("wip.yaml"))

	var stdout, stderr bytes.Buffer
	err := CleanFiles(cfg, CleanRequest{
		Presentation: presentation.New(&stdout, &stderr, true),
		KeyFile:      ".age/keys.txt",
	})
	require.NoError(t, err)
	assert.Contains(t, stderr.String(), "ALREADY ABSENT wip.yaml")
	assert.Contains(t, stderr.String(), "Selected 2 file pairs")
}

func TestCleanFilesDiagnosticsFailureFailsAfterCompletedRemovals(t *testing.T) {
	cfg := prepareCleanFixture(t)

	err := CleanFiles(cfg, CleanRequest{
		Presentation: presentation.New(io.Discard, failingWriter{}, false),
		Input:        strings.NewReader("y\n"),
		KeyFile:      ".age/keys.txt",
	})
	require.Error(t, err)
	assert.NoFileExists(t, "same.yaml", "completed removals are not rolled back")
	assert.FileExists(t, "wip.yaml")
}
