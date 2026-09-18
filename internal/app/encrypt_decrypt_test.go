package app

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"filippo.io/age"
	"github.com/YewFence/YewSeal/internal/agekey"
	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/presentation"
	"github.com/YewFence/YewSeal/internal/seal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryptDecryptSingleFileOutputUsesConfiguredFormat(t *testing.T) {
	env := newAppCryptoTestEnv(t)

	require.NoError(t, os.WriteFile("secrets.vars", []byte("TOKEN=secret\n"), 0644))
	cfg := configWithOwnerRecipient(&config.Config{Encryption: config.EncryptionConfig{Files: []config.FilePair{{PlaintextPath: "secrets.vars", EncryptedPath: "secrets.vars.enc.yaml", Format: "env"}}}}, env.publicKey)

	err := EncryptFiles(cfg, EncryptRequest{
		Targets:   []string{"secrets.vars"},
		Output:    "secrets.vars.enc.yaml",
		OutputSet: true,
		Parallel:  1,
	})
	require.NoError(t, err)

	require.NoError(t, os.Remove("secrets.vars"))
	err = DecryptFiles(cfg, DecryptRequest{
		KeyFile:   env.keyFile,
		Targets:   []string{"secrets.vars.enc.yaml"},
		Output:    "secrets.vars",
		OutputSet: true,
		Parallel:  1,
	})
	require.NoError(t, err)

	content, err := os.ReadFile("secrets.vars")
	require.NoError(t, err)
	assert.Equal(t, "TOKEN=secret\n", string(content))
}

func TestDecryptFilesFormatsJSONWithIndentation(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	plain := []byte(`{"database":{"host":"localhost","credentials":{"username":"admin","password":"secret"}}}`)
	require.NoError(t, os.WriteFile("config.json", plain, 0600))
	cfg := configWithOwnerRecipient(&config.Config{Encryption: config.EncryptionConfig{Files: []config.FilePair{{PlaintextPath: "config.json", EncryptedPath: "config.enc.json", Format: "json"}}}}, env.publicKey)

	require.NoError(t, EncryptFiles(cfg, EncryptRequest{Targets: []string{"config.json"}, Parallel: 1}))
	require.NoError(t, os.Remove("config.json"))
	require.NoError(t, DecryptFiles(cfg, DecryptRequest{KeyFile: env.keyFile, Targets: []string{"config.enc.json"}, Parallel: 1}))

	content, err := os.ReadFile("config.json")
	require.NoError(t, err)
	require.Equal(t, "{\n\t\"database\": {\n\t\t\"host\": \"localhost\",\n\t\t\"credentials\": {\n\t\t\t\"username\": \"admin\",\n\t\t\t\"password\": \"secret\"\n\t\t}\n\t}\n}\n", string(content))
}

func TestEncryptFilesUsesPerFileRecipientAliases(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	second, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	require.NoError(t, os.WriteFile("config.yaml", []byte("secret: value\n"), 0644))

	defaults := []string{"owner"}
	fileRecipients := []string{"backup"}
	cfg := &config.Config{
		Encryption: config.EncryptionConfig{
			Files: []config.FilePair{{
				PlaintextPath: "config.yaml",
				EncryptedPath: "config.enc.yaml",
				Recipients:    &fileRecipients,
			}},
		},
		Recipients: config.RecipientConfig{
			Defaults: &defaults,
			Registry: map[string]string{
				"owner":  env.publicKey,
				"backup": second.Recipient().String(),
			},
		},
	}

	require.NoError(t, EncryptFiles(cfg, EncryptRequest{Targets: []string{"config.yaml"}, Parallel: 1}))

	recipients, err := seal.ExtractAgeRecipientsFromEncryptedFile("config.enc.yaml", "config.yaml", "")
	require.NoError(t, err)
	require.Equal(t, []string{second.Recipient().String()}, recipients)
}

func TestEncryptFiles_DirModeRejectsOutput(t *testing.T) {
	cfg := config.DefaultConfig()
	tempDir := t.TempDir()

	err := EncryptFiles(cfg, EncryptRequest{
		Targets:   []string{tempDir},
		Output:    "out.enc.yaml",
		OutputSet: true,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--output is only supported when the path target is a file")
}

func TestDecryptFiles_DirModeRejectsOutput(t *testing.T) {
	cfg := config.DefaultConfig()
	tempDir := t.TempDir()

	err := DecryptFiles(cfg, DecryptRequest{
		Targets:   []string{tempDir},
		Output:    "out.yaml",
		OutputSet: true,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--output is only supported when the path target is a file")
}

func TestEncryptFiles_TargetFileUsesConfiguredPair(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	require.NoError(t, os.WriteFile(".dev.vars", []byte("TOKEN=secret\n"), 0644))

	cfg := configWithOwnerRecipient(&config.Config{
		Encryption: config.EncryptionConfig{
			Files: []config.FilePair{
				{PlaintextPath: ".dev.vars", EncryptedPath: "configured.enc.yaml", Format: "env"},
			},
		},
	}, env.publicKey)

	err := EncryptFiles(cfg, EncryptRequest{
		Targets:  []string{".dev.vars"},
		Parallel: 1,
	})
	require.NoError(t, err)

	_, err = os.Stat("configured.enc.yaml")
	require.NoError(t, err)
	_, err = os.Stat(".dev.vars.enc.env")
	assert.True(t, os.IsNotExist(err))
}

func TestEncryptFilesConfiguredFormatHandlesExtensionlessInput(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	require.NoError(t, os.WriteFile("secret", []byte("TOKEN=secret\n"), 0644))

	cfg := configWithOwnerRecipient(&config.Config{Encryption: config.EncryptionConfig{Files: []config.FilePair{{PlaintextPath: "secret", EncryptedPath: "secret.enc.env", Format: "env"}}}}, env.publicKey)
	err := EncryptFiles(cfg, EncryptRequest{
		Targets:  []string{"secret"},
		Parallel: 1,
	})
	require.NoError(t, err)

	_, err = os.Stat("secret.enc.env")
	require.NoError(t, err)
}

func TestDecryptFiles_TargetFileOutputOverride(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	require.NoError(t, os.WriteFile("config.yaml", []byte("token: secret\n"), 0644))
	cfg := configWithOwnerRecipient(&config.Config{Encryption: config.EncryptionConfig{Files: []config.FilePair{{PlaintextPath: "config.yaml", EncryptedPath: "config.enc.yaml"}}}}, env.publicKey)
	err := EncryptFiles(cfg, EncryptRequest{
		Targets:  []string{"config.yaml"},
		Parallel: 1,
	})
	require.NoError(t, err)
	require.NoError(t, os.Remove("config.yaml"))

	err = DecryptFiles(cfg, DecryptRequest{
		KeyFile:   env.keyFile,
		Targets:   []string{"config.enc.yaml"},
		Output:    "custom.json",
		OutputSet: true,
		Parallel:  1,
	})
	require.NoError(t, err)

	content, err := os.ReadFile("custom.json")
	require.NoError(t, err)
	assert.Equal(t, "token: secret\n", string(content))
	_, err = os.Stat("config.yaml")
	assert.True(t, os.IsNotExist(err))
}

func TestEncryptFilesOutputOverrideKeepsInferredFormat(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	plain := []byte("token: secret\n")
	require.NoError(t, os.WriteFile("config.yaml", plain, 0600))
	cfg := configWithOwnerRecipient(&config.Config{Encryption: config.EncryptionConfig{Files: []config.FilePair{{PlaintextPath: "config.yaml", EncryptedPath: "config.enc.yaml"}}}}, env.publicKey)
	require.NoError(t, EncryptFiles(cfg, EncryptRequest{Targets: []string{"config.yaml"}, Output: "export.json", OutputSet: true}))
	decrypted, err := seal.DecryptToBytes(seal.DecryptBytesOptions{
		InputFile:      "export.json",
		OutputFile:     "config.yaml",
		IdentityBundle: mustTestBundle(t, env.keyFile),
		FormatOverride: "yaml",
	})
	require.NoError(t, err)
	require.Equal(t, plain, decrypted)
	require.NoFileExists(t, "config.enc.yaml")
}

func TestDecryptFilesOutputRejectsBatchWithOnlyOneSelectedFile(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	cfg := configWithOwnerRecipient(&config.Config{Encryption: config.EncryptionConfig{Files: []config.FilePair{{PlaintextPath: "config.yaml", EncryptedPath: "config.enc.yaml", Format: "yaml"}}}}, env.publicKey)
	err := DecryptFiles(cfg, DecryptRequest{Output: "export.yaml", OutputSet: true})
	require.ErrorContains(t, err, "--output is only supported when the path target is a file")
	require.NoFileExists(t, "export.yaml")
}

type appCryptoTestEnv struct {
	keyFile   string
	publicKey string
}

func newAppCryptoTestEnv(t *testing.T) appCryptoTestEnv {
	t.Helper()

	tempDir := t.TempDir()
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tempDir))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(oldWd))
	})

	keyFile, publicKey := createAgeKeyFile(t, tempDir)
	return appCryptoTestEnv{keyFile: keyFile, publicKey: publicKey}
}

func configWithOwnerRecipient(cfg *config.Config, publicKey string) *config.Config {
	defaults := []string{"owner"}
	cfg.Recipients = config.RecipientConfig{
		Defaults: &defaults,
		Registry: map[string]string{"owner": publicKey},
	}
	return cfg
}

func TestEncryptFilesRejectsMissingAuthorizationBeforeWrites(t *testing.T) {
	root := t.TempDir()
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(root))
	t.Cleanup(func() { require.NoError(t, os.Chdir(oldWd)) })
	require.NoError(t, os.WriteFile("secret.yaml", []byte("token: value\n"), 0644))
	cfg := &config.Config{CurrentDir: root, UserConfig: true, Encryption: config.EncryptionConfig{Files: []config.FilePair{{PlaintextPath: "secret.yaml", EncryptedPath: "secret.enc.yaml", Format: "yaml"}}}}
	err = EncryptFiles(cfg, EncryptRequest{Targets: []string{"secret.yaml"}, UpdateProjectMetadata: true})
	require.Error(t, err)
	require.Contains(t, err.Error(), "no recipient set")
	_, statErr := os.Stat("secret.enc.yaml")
	require.ErrorIs(t, statErr, os.ErrNotExist)
	_, statErr = os.Stat(".sops.yaml")
	require.ErrorIs(t, statErr, os.ErrNotExist)
}

func TestEncryptFilesPreflightsEntireBatchBeforeFirstCiphertext(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	require.NoError(t, os.WriteFile("first.yaml", []byte("token: first\n"), 0644))
	require.NoError(t, os.WriteFile("second.yaml", []byte("token: second\n"), 0644))
	valid := []string{"owner"}
	invalid := []string{"missing"}
	cfg := &config.Config{CurrentDir: config.CurrentDir(&config.Config{}), Recipients: config.RecipientConfig{Registry: map[string]string{"owner": env.publicKey}}, Encryption: config.EncryptionConfig{Files: []config.FilePair{{PlaintextPath: "first.yaml", EncryptedPath: "first.enc.yaml", Format: "yaml", Recipients: &valid}, {PlaintextPath: "second.yaml", EncryptedPath: "second.enc.yaml", Format: "yaml", Recipients: &invalid}}}}
	err := EncryptFiles(cfg, EncryptRequest{Parallel: 1})
	require.Error(t, err)
	require.Contains(t, err.Error(), `unknown recipient alias "missing"`)
	_, statErr := os.Stat("first.enc.yaml")
	require.ErrorIs(t, statErr, os.ErrNotExist)
	_, statErr = os.Stat("second.enc.yaml")
	require.ErrorIs(t, statErr, os.ErrNotExist)
}

func TestEncryptFilesWritesPortableSopsPaths(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	require.NoError(t, os.WriteFile("secret.yaml", []byte("token: value\n"), 0644))
	cfg := configWithOwnerRecipient(&config.Config{CurrentDir: config.CurrentDir(&config.Config{}), Encryption: config.EncryptionConfig{Files: []config.FilePair{{PlaintextPath: "secret.yaml", EncryptedPath: "secret.enc.yaml", Format: "yaml"}}}}, env.publicKey)
	require.NoError(t, EncryptFiles(cfg, EncryptRequest{Targets: []string{"secret.yaml"}, Parallel: 1, UpdateProjectMetadata: true, SyncSOPSConfig: true}))
	content, err := os.ReadFile(".sops.yaml")
	require.NoError(t, err)
	assert.Contains(t, string(content), `path_regex: ^secret\.enc\.yaml$`)
	assert.NotContains(t, string(content), config.CurrentDir(cfg))
}

func TestTargetedEncryptSyncsCompleteProjectSopsPolicy(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	for _, path := range []string{"first.yaml", "second.yaml"} {
		require.NoError(t, os.WriteFile(path, []byte("token: value\n"), 0o600))
	}
	cfg := configWithOwnerRecipient(&config.Config{Encryption: config.EncryptionConfig{Files: []config.FilePair{
		{PlaintextPath: "first.yaml", EncryptedPath: "first.enc.yaml", Format: "yaml"},
		{PlaintextPath: "second.yaml", EncryptedPath: "second.enc.yaml", Format: "yaml"},
	}}}, env.publicKey)

	require.NoError(t, EncryptFiles(cfg, EncryptRequest{
		Targets:               []string{"first.yaml"},
		Output:                "review.enc.yaml",
		OutputSet:             true,
		Parallel:              1,
		UpdateProjectMetadata: true,
		SyncSOPSConfig:        true,
	}))

	content, err := os.ReadFile(".sops.yaml")
	require.NoError(t, err)
	assert.Contains(t, string(content), `path_regex: ^first\.enc\.yaml$`)
	assert.Contains(t, string(content), `path_regex: ^second\.enc\.yaml$`)
	assert.NotContains(t, string(content), "review")
	require.FileExists(t, "review.enc.yaml")
	require.NoFileExists(t, "second.enc.yaml")
}

func TestEncryptLeavesSopsConfigUntouchedWhenSyncDisabled(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	require.NoError(t, os.WriteFile("secret.yaml", []byte("token: value\n"), 0o600))
	stale := []byte("custom: untouched\n")
	require.NoError(t, os.WriteFile(".sops.yaml", stale, 0o600))
	cfg := configWithOwnerRecipient(&config.Config{Encryption: config.EncryptionConfig{Files: []config.FilePair{{PlaintextPath: "secret.yaml", EncryptedPath: "secret.enc.yaml", Format: "yaml"}}}}, env.publicKey)

	require.NoError(t, EncryptFiles(cfg, EncryptRequest{Parallel: 1, UpdateProjectMetadata: true}))
	content, err := os.ReadFile(".sops.yaml")
	require.NoError(t, err)
	assert.Equal(t, stale, content)
}

func TestEncryptReportsSopsSyncFailureWithoutReplacingTaskResult(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	require.NoError(t, os.WriteFile("good.yaml", []byte("token: value\n"), 0o600))
	require.NoError(t, os.WriteFile("blocked.yaml", []byte("token: value\n"), 0o600))
	require.NoError(t, os.Mkdir("blocked.enc.yaml", 0o700))
	require.NoError(t, os.Mkdir(".sops.yaml", 0o700))
	cfg := configWithOwnerRecipient(&config.Config{Encryption: config.EncryptionConfig{Files: []config.FilePair{
		{PlaintextPath: "good.yaml", EncryptedPath: "good.enc.yaml", Format: "yaml"},
		{PlaintextPath: "blocked.yaml", EncryptedPath: "blocked.enc.yaml", Format: "yaml"},
	}}}, env.publicKey)
	var diagnostics bytes.Buffer

	err := EncryptFiles(cfg, EncryptRequest{
		Presentation:          presentation.New(nil, &diagnostics, false),
		Parallel:              1,
		UpdateProjectMetadata: true,
		SyncSOPSConfig:        true,
	})
	require.ErrorContains(t, err, "1 of 2 files failed to encrypt")
	require.FileExists(t, "good.enc.yaml")
	warningIndex := strings.Index(diagnostics.String(), "WARNING failed to update .sops.yaml")
	summaryIndex := strings.Index(diagnostics.String(), "Summary (encrypted)")
	require.GreaterOrEqual(t, warningIndex, 0)
	require.Greater(t, summaryIndex, warningIndex)
}

func TestEncryptFilesLeavesUnchangedCiphertextUntouched(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	plain := []byte("token: value\n")
	require.NoError(t, os.WriteFile("secret.yaml", plain, 0644))
	cfg := configWithOwnerRecipient(&config.Config{Encryption: config.EncryptionConfig{Files: []config.FilePair{{PlaintextPath: "secret.yaml", EncryptedPath: "secret.enc.yaml", Format: "yaml"}}}}, env.publicKey)
	require.NoError(t, EncryptFiles(cfg, EncryptRequest{KeyFile: env.keyFile, Parallel: 1}))
	original, err := os.ReadFile("secret.enc.yaml")
	require.NoError(t, err)
	require.NoError(t, os.Chmod("secret.enc.yaml", 0400))

	var diagnostics bytes.Buffer
	err = EncryptFiles(cfg, EncryptRequest{KeyFile: env.keyFile, Parallel: 1, Presentation: presentation.New(nil, &diagnostics, true)})
	require.NoError(t, err)
	current, err := os.ReadFile("secret.enc.yaml")
	require.NoError(t, err)
	assert.Equal(t, original, current)
	assert.Contains(t, diagnostics.String(), "UNCHANGED secret.yaml -> secret.enc.yaml")
	assert.Contains(t, diagnostics.String(), "0 encrypted, 1 unchanged, 0 missing plaintext, 0 failed (1 selected)")
}

func TestEncryptFilesWithoutIdentityWarnsAndFreshlyEncrypts(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	plain := []byte("token: value\n")
	require.NoError(t, os.WriteFile("secret.yaml", plain, 0644))
	cfg := configWithOwnerRecipient(&config.Config{Encryption: config.EncryptionConfig{Files: []config.FilePair{{PlaintextPath: "secret.yaml", EncryptedPath: "secret.enc.yaml", Format: "yaml"}}}}, env.publicKey)
	require.NoError(t, EncryptFiles(cfg, EncryptRequest{KeyFile: env.keyFile, Parallel: 1}))
	original, err := os.ReadFile("secret.enc.yaml")
	require.NoError(t, err)
	require.NoError(t, os.Remove(env.keyFile))
	for _, name := range []string{"YEWSEAL_AGE_IDENTITIES", "SOPS_AGE_KEY", "SOPS_AGE_KEY_FILE", "SOPS_AGE_KEY_CMD"} {
		t.Setenv(name, "")
	}

	var diagnostics bytes.Buffer
	err = EncryptFiles(cfg, EncryptRequest{Parallel: 1, Presentation: presentation.New(nil, &diagnostics, false)})
	require.NoError(t, err)
	current, err := os.ReadFile("secret.enc.yaml")
	require.NoError(t, err)
	assert.NotEqual(t, original, current)
	assert.Contains(t, diagnostics.String(), "no Age identity found")
	assert.Contains(t, diagnostics.String(), "1 encrypted, 0 unchanged")
}

func TestEncryptFilesWithUnmatchedIdentityWarnsPerFileAndReplaces(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	plain := []byte("token: value\n")
	require.NoError(t, os.WriteFile("secret.yaml", plain, 0644))
	cfg := configWithOwnerRecipient(&config.Config{Encryption: config.EncryptionConfig{Files: []config.FilePair{{PlaintextPath: "secret.yaml", EncryptedPath: "secret.enc.yaml", Format: "yaml"}}}}, env.publicKey)
	require.NoError(t, EncryptFiles(cfg, EncryptRequest{KeyFile: env.keyFile, Parallel: 1}))
	original, err := os.ReadFile("secret.enc.yaml")
	require.NoError(t, err)
	other, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	otherKeyFile := "other-keys.txt"
	require.NoError(t, os.WriteFile(otherKeyFile, []byte(other.String()+"\n"), 0600))

	var diagnostics bytes.Buffer
	err = EncryptFiles(cfg, EncryptRequest{KeyFile: otherKeyFile, Parallel: 1, Presentation: presentation.New(nil, &diagnostics, false)})
	require.NoError(t, err)
	current, err := os.ReadFile("secret.enc.yaml")
	require.NoError(t, err)
	assert.NotEqual(t, original, current)
	assert.Contains(t, diagnostics.String(), "secret.yaml: no matching age identity for existing ciphertext")
}

func TestEncryptFilesProtectsBrokenCiphertextUnlessForced(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	require.NoError(t, os.WriteFile("secret.yaml", []byte("token: value\n"), 0644))
	broken := []byte("not sops\n")
	require.NoError(t, os.WriteFile("secret.enc.yaml", broken, 0644))
	cfg := configWithOwnerRecipient(&config.Config{Encryption: config.EncryptionConfig{Files: []config.FilePair{{PlaintextPath: "secret.yaml", EncryptedPath: "secret.enc.yaml", Format: "yaml"}}}}, env.publicKey)

	err := EncryptFiles(cfg, EncryptRequest{KeyFile: env.keyFile, Parallel: 1})
	require.Error(t, err)
	current, readErr := os.ReadFile("secret.enc.yaml")
	require.NoError(t, readErr)
	assert.Equal(t, broken, current)

	require.NoError(t, EncryptFiles(cfg, EncryptRequest{Parallel: 1, Force: true}))
	decrypted, err := seal.DecryptToBytes(seal.DecryptBytesOptions{InputFile: "secret.enc.yaml", OutputFile: "secret.yaml", IdentityBundle: mustTestBundle(t, env.keyFile), FormatOverride: "yaml"})
	require.NoError(t, err)
	assert.Equal(t, "token: value\n", string(decrypted))
}

func TestEncryptFilesRejectsInvalidIdentityBeforeMetadataWrites(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	require.NoError(t, os.WriteFile("secret.yaml", []byte("token: value\n"), 0644))
	cfg := configWithOwnerRecipient(&config.Config{Encryption: config.EncryptionConfig{Files: []config.FilePair{{PlaintextPath: "secret.yaml", EncryptedPath: "secret.enc.yaml", Format: "yaml"}}}}, env.publicKey)
	require.NoError(t, EncryptFiles(cfg, EncryptRequest{KeyFile: env.keyFile, Parallel: 1}))
	original, err := os.ReadFile("secret.enc.yaml")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile("invalid-keys.txt", []byte("invalid\n"), 0600))

	err = EncryptFiles(cfg, EncryptRequest{KeyFile: "invalid-keys.txt", Parallel: 1, UpdateProjectMetadata: true})
	require.Error(t, err)
	current, readErr := os.ReadFile("secret.enc.yaml")
	require.NoError(t, readErr)
	assert.Equal(t, original, current)
	require.NoFileExists(t, ".gitignore")
	require.NoFileExists(t, ".sops.yaml")
}

func TestDecryptFilesWarnsForStaleAliasAndUsesEncryptedMetadata(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	require.NoError(t, os.WriteFile("secret.yaml", []byte("token: value\n"), 0644))
	require.NoError(t, seal.Encrypt(seal.EncryptOptions{InputFile: "secret.yaml", OutputFile: "secret.enc.yaml", Recipients: []string{env.publicKey}, FormatOverride: "yaml"}))
	require.NoError(t, os.Remove("secret.yaml"))
	aliases := []string{"retired-owner"}
	cfg := &config.Config{Encryption: config.EncryptionConfig{Files: []config.FilePair{{PlaintextPath: "secret.yaml", EncryptedPath: "secret.enc.yaml", Format: "yaml", Recipients: &aliases, ConfigPath: ".yewseal.toml"}}}}
	var warning bytes.Buffer
	err := DecryptFiles(cfg, DecryptRequest{KeyFile: env.keyFile, Targets: []string{"secret.enc.yaml"}, Parallel: 1, Presentation: presentation.New(nil, &warning, false)})
	require.NoError(t, err)
	assert.Contains(t, warning.String(), `unknown recipient alias "retired-owner"`)
	assert.Contains(t, warning.String(), ".yewseal.toml")
	content, err := os.ReadFile("secret.yaml")
	require.NoError(t, err)
	assert.Equal(t, "token: value\n", string(content))
}

func TestDecryptFilesUsesEnvironmentBundleWithoutKeyFile(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	bundle, err := agekey.GetIdentityBundle(env.keyFile)
	require.NoError(t, err)
	malformed := "AGE-SECRET-KEY-1DELIBERATELY-CORRUPTED"
	t.Setenv("YEWSEAL_AGE_IDENTITIES", bundle.String()+","+malformed)
	require.NoError(t, os.Remove(env.keyFile))
	require.NoError(t, os.WriteFile("secret.yaml", []byte("token: value\n"), 0644))
	require.NoError(t, seal.Encrypt(seal.EncryptOptions{InputFile: "secret.yaml", OutputFile: "secret.enc.yaml", Recipients: []string{env.publicKey}, FormatOverride: "yaml"}))
	require.NoError(t, os.Remove("secret.yaml"))
	cfg := &config.Config{Encryption: config.EncryptionConfig{Files: []config.FilePair{{PlaintextPath: "secret.yaml", EncryptedPath: "secret.enc.yaml", Format: "yaml"}}}}
	var diagnostics bytes.Buffer
	require.NoError(t, DecryptFiles(cfg, DecryptRequest{Targets: []string{"secret.enc.yaml"}, Parallel: 1, Presentation: presentation.New(nil, &diagnostics, false)}))
	require.Contains(t, diagnostics.String(), "WARNING ignored malformed Age identity bundle item at line 1, item 2 (AGE-SECRET-KEY-…PTED)")
	require.NotContains(t, diagnostics.String(), bundle.String())
	require.NotContains(t, diagnostics.String(), malformed)
	content, err := os.ReadFile("secret.yaml")
	require.NoError(t, err)
	assert.Equal(t, "token: value\n", string(content))
}

func TestDecryptFilesUsesSecondIdentityFromDefaultFile(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	for _, name := range []string{"YEWSEAL_AGE_IDENTITIES", "SOPS_AGE_KEY", "SOPS_AGE_KEY_FILE", "SOPS_AGE_KEY_CMD"} {
		t.Setenv(name, "")
	}
	keyContent, err := os.ReadFile(env.keyFile)
	require.NoError(t, err)
	unrelated, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(env.keyFile, append([]byte(unrelated.String()+"\n"), keyContent...), 0600))
	require.NoError(t, os.WriteFile("secret.yaml", []byte("token: value\n"), 0644))
	require.NoError(t, seal.Encrypt(seal.EncryptOptions{InputFile: "secret.yaml", OutputFile: "secret.enc.yaml", Recipients: []string{env.publicKey}, FormatOverride: "yaml"}))
	require.NoError(t, os.Remove("secret.yaml"))
	cfg := &config.Config{Encryption: config.EncryptionConfig{Files: []config.FilePair{{PlaintextPath: "secret.yaml", EncryptedPath: "secret.enc.yaml", Format: "yaml"}}}}
	require.NoError(t, DecryptFiles(cfg, DecryptRequest{Targets: []string{"secret.enc.yaml"}, Parallel: 1}))
	content, err := os.ReadFile("secret.yaml")
	require.NoError(t, err)
	assert.Equal(t, "token: value\n", string(content))
}
