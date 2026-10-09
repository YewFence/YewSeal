package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"filippo.io/age"
	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/errx"
	"github.com/YewFence/YewSeal/internal/presentation"
	"github.com/YewFence/YewSeal/internal/seal"
	"github.com/YewFence/YewSeal/internal/sopsx"
	"github.com/YewFence/YewSeal/internal/verify"
	"github.com/stretchr/testify/require"
)

func TestDecryptDeliveryRootPreconditions(t *testing.T) {
	for _, kind := range []string{"missing", "file", "symlink", "nonempty", "hidden"} {
		t.Run(kind, func(t *testing.T) {
			env := newAppCryptoTestEnv(t)
			cfg := configWithOwnerRecipient(&config.Config{Encryption: config.EncryptionConfig{Files: []config.FilePair{{PlaintextPath: "secret.yaml", EncryptedPath: "secret.enc.yaml"}}}}, env.publicKey)
			root := filepath.Join(t.TempDir(), "output")
			switch kind {
			case "file":
				require.NoError(t, os.WriteFile(root, []byte("untouched"), 0644))
			case "symlink":
				require.NoError(t, os.Symlink(t.TempDir(), root))
			case "nonempty", "hidden":
				require.NoError(t, os.Mkdir(root, 0755))
				name := "existing"
				if kind == "hidden" {
					name = ".hidden"
				}
				require.NoError(t, os.WriteFile(filepath.Join(root, name), []byte("untouched"), 0644))
			}
			err := DecryptFiles(cfg, DecryptRequest{OutputDir: root, KeyFile: "nonexistent-key", UpdateGitignore: true})
			require.ErrorContains(t, err, "--output")
			var usage *errx.UsageError
			require.ErrorAs(t, err, &usage)
			require.Equal(t, 2, usage.ExitCode())
			require.NoFileExists(t, "secret.yaml")
			require.NoFileExists(t, ".gitignore")
		})
	}
}

func TestDecryptDeliveryProjectionSelectorsAndPermissions(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	projectRoot, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll("config/nested", 0755))
	cfg := configWithOwnerRecipient(&config.Config{CurrentDir: projectRoot, ProjectRoot: projectRoot, Encryption: config.EncryptionConfig{Files: []config.FilePair{
		{PlaintextPath: "config/first.yaml", EncryptedPath: "config/first.enc.yaml", PlaintextMode: config.PlaintextDelivery},
		{PlaintextPath: "config/nested/second.yaml", EncryptedPath: "config/nested/second.enc.yaml"},
	}}}, env.publicKey)
	for _, pair := range cfg.Encryption.Files {
		data, err := sopsx.Encrypt([]byte("token: secret\n"), "yaml", []string{env.publicKey})
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(pair.EncryptedPath, data, 0644))
	}
	for _, tc := range []struct {
		name    string
		targets []string
		count   int
	}{
		{"scope", nil, 2},
		{"single", []string{"config/first.enc.yaml"}, 1},
		{"union", []string{"config/first.enc.yaml", "config/nested/second.enc.yaml"}, 2},
		{"directory", []string{"config"}, 2},
		{"glob", []string{"config/**/*.enc.yaml"}, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			require.NoError(t, os.Chmod(root, 0751))
			var stdout, stderr bytes.Buffer
			require.NoError(t, DecryptFiles(cfg, DecryptRequest{OutputDir: root, KeyFile: env.keyFile, Targets: tc.targets, Parallel: 4, Strict: true, UpdateGitignore: true, Presentation: presentation.New(&stdout, &stderr, true)}))
			require.Empty(t, stdout.String())
			require.Contains(t, stderr.String(), "Summary")
			require.NotContains(t, stderr.String(), "warning")
			count := 0
			require.NoError(t, filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
				require.NoError(t, walkErr)
				info, err := entry.Info()
				require.NoError(t, err)
				switch {
				case path == root:
					require.Equal(t, os.FileMode(0751), info.Mode().Perm())
				case entry.IsDir():
					require.Equal(t, os.FileMode(0700), info.Mode().Perm())
				default:
					count++
					require.Equal(t, os.FileMode(0600), info.Mode().Perm())
					data, err := os.ReadFile(path)
					require.NoError(t, err)
					require.Equal(t, "token: secret\n", string(data))
				}
				return nil
			}))
			require.Equal(t, tc.count, count)
			require.FileExists(t, filepath.Join(root, "config/first.yaml"))
			require.NoFileExists(t, "config/first.yaml")
			require.NoFileExists(t, ".gitignore")
			require.NoFileExists(t, ".sops.yaml")
		})
	}
	require.NoError(t, os.Mkdir("output", 0755))
	var stderr bytes.Buffer
	require.NoError(t, DecryptFiles(cfg, DecryptRequest{OutputDir: "output", KeyFile: env.keyFile, UpdateGitignore: true, Presentation: presentation.New(nil, &stderr, false)}))
	require.FileExists(t, "output/config/first.yaml")
	require.NotContains(t, stderr.String(), "warning")
	require.NoFileExists(t, ".gitignore")
}

func TestDecryptDeliveryAllFormatsAndLogicalSymlink(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	for _, tc := range []struct{ format, plaintext string }{
		{"toml", "token = 'secret'\n"},
		{"yaml", "token: secret\n"},
		{"json", `{"token":"secret"}`},
		{"env", "TOKEN=secret\n"},
		{"ini", "[section]\ntoken=secret\n"},
		{"binary", "\x00\x01secret\xff"},
	} {
		t.Run(tc.format, func(t *testing.T) {
			data, err := sopsx.Encrypt([]byte(tc.plaintext), tc.format, []string{env.publicKey})
			require.NoError(t, err)
			ciphertext := "source.enc." + tc.format
			require.NoError(t, os.WriteFile(ciphertext, data, 0644))
			logical := "logical." + tc.format
			linkTarget := filepath.Join(t.TempDir(), "external")
			require.NoError(t, os.WriteFile(linkTarget, []byte("untouched"), 0644))
			require.NoError(t, os.Symlink(linkTarget, logical))
			cfg := &config.Config{Encryption: config.EncryptionConfig{Files: []config.FilePair{{PlaintextPath: logical, EncryptedPath: ciphertext, Format: tc.format, PlaintextMode: config.PlaintextDelivery}}}}
			root := t.TempDir()
			require.NoError(t, DecryptFiles(cfg, DecryptRequest{OutputDir: root, KeyFile: env.keyFile}))
			destination := filepath.Join(root, logical)
			info, err := os.Lstat(destination)
			require.NoError(t, err)
			require.True(t, info.Mode().IsRegular())
			actual, err := os.ReadFile(destination)
			require.NoError(t, err)
			expected, err := seal.DecryptToBytes(seal.DecryptBytesOptions{InputFile: ciphertext, OutputFile: logical, FormatOverride: tc.format, IdentityBundle: mustTestBundle(t, env.keyFile)})
			require.NoError(t, err)
			require.Equal(t, expected, actual)
			unchanged, err := os.ReadFile(linkTarget)
			require.NoError(t, err)
			require.Equal(t, "untouched", string(unchanged))
		})
	}
}

func TestDecryptDeliveryRejectsEntireInvalidProjection(t *testing.T) {
	for _, kind := range []string{"outside", "ancestor", "ciphertext"} {
		t.Run(kind, func(t *testing.T) {
			env := newAppCryptoTestEnv(t)
			root := t.TempDir()
			pairs := []config.FilePair{{PlaintextPath: "first.yaml", EncryptedPath: "first.enc.yaml"}}
			switch kind {
			case "outside":
				pairs = append(pairs, config.FilePair{PlaintextPath: filepath.Join(t.TempDir(), "external.yaml"), EncryptedPath: "external.enc.yaml"})
			case "ancestor":
				pairs = append(pairs, config.FilePair{PlaintextPath: "first.yaml/child.yaml", EncryptedPath: "child.enc.yaml"})
			case "ciphertext":
				pairs = append(pairs, config.FilePair{PlaintextPath: "second.yaml", EncryptedPath: filepath.Join(root, "first.yaml")})
			}
			cfg := &config.Config{Encryption: config.EncryptionConfig{Files: pairs}}
			err := DecryptFiles(cfg, DecryptRequest{OutputDir: root, KeyFile: env.keyFile, Targets: []string{pairs[0].EncryptedPath, pairs[1].EncryptedPath}, UpdateGitignore: true})
			require.Error(t, err)
			var usage *errx.UsageError
			require.ErrorAs(t, err, &usage)
			require.Equal(t, 2, usage.ExitCode())
			entries, err := os.ReadDir(root)
			require.NoError(t, err)
			require.Empty(t, entries)
			require.NoFileExists(t, ".gitignore")
		})
	}
}

func TestDecryptDeliveryLenientStrictPartialTrees(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	other, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	for name, key := range map[string]string{"allowed": env.publicKey, "denied": other.Recipient().String()} {
		data, err := sopsx.Encrypt([]byte("token: secret\n"), "yaml", []string{key})
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(name+".enc.yaml", data, 0644))
	}
	for _, strict := range []bool{false, true} {
		for _, realError := range []bool{false, true} {
			root := t.TempDir()
			cfg := &config.Config{Encryption: config.EncryptionConfig{Files: []config.FilePair{
				{PlaintextPath: "allowed.yaml", EncryptedPath: "allowed.enc.yaml"},
				{PlaintextPath: "denied.yaml", EncryptedPath: "denied.enc.yaml"},
			}}}
			if realError {
				cfg.Encryption.Files = append(cfg.Encryption.Files, config.FilePair{PlaintextPath: "missing.yaml", EncryptedPath: "missing.enc.yaml"})
			}
			var stdout, stderr bytes.Buffer
			err := DecryptFiles(cfg, DecryptRequest{OutputDir: root, KeyFile: env.keyFile, Strict: strict, Parallel: 3, Presentation: presentation.New(&stdout, &stderr, false)})
			if strict || realError {
				require.Error(t, err)
				var usage *errx.UsageError
				require.False(t, errors.As(err, &usage))
			} else {
				require.NoError(t, err)
			}
			require.Empty(t, stdout.String())
			require.Contains(t, stderr.String(), "no matching age identity")
			require.FileExists(t, filepath.Join(root, "allowed.yaml"))
			require.NoFileExists(t, filepath.Join(root, "denied.yaml"))
		}
	}
}

func TestPlaintextDeliveryPolicyAndUnaffectedCommands(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	cfg := configWithOwnerRecipient(&config.Config{Encryption: config.EncryptionConfig{Files: []config.FilePair{
		{PlaintextPath: "first.yaml", EncryptedPath: "first.enc.yaml"},
		{PlaintextPath: "secret.yaml", EncryptedPath: "secret.enc.yaml", PlaintextMode: config.PlaintextDelivery},
	}}}, env.publicKey)
	for _, pair := range cfg.Encryption.Files {
		require.NoError(t, os.WriteFile(pair.PlaintextPath, []byte("token: secret\n"), 0600))
	}
	require.NoError(t, EncryptFiles(cfg, EncryptRequest{UpdateGitignore: true}))
	for _, pair := range cfg.Encryption.Files {
		require.NoError(t, os.Remove(pair.PlaintextPath))
	}
	for _, strict := range []bool{false, true} {
		err := DecryptFiles(cfg, DecryptRequest{KeyFile: env.keyFile, Strict: strict})
		require.ErrorContains(t, err, "--output DIR")
		require.ErrorContains(t, err, "--inplace")
		var usage *errx.UsageError
		require.ErrorAs(t, err, &usage)
		require.Equal(t, 2, usage.ExitCode())
		require.NoFileExists(t, "first.yaml")
		require.NoFileExists(t, "secret.yaml")
	}
	var view bytes.Buffer
	require.NoError(t, ViewTarget(&view, nil, cfg, ViewRequest{Target: "secret.enc.yaml", KeyFile: env.keyFile}))
	require.Equal(t, "token: secret\n", view.String())
	require.NoFileExists(t, "secret.yaml")
	require.NoError(t, DecryptFiles(cfg, DecryptRequest{KeyFile: env.keyFile, Inplace: true, UpdateGitignore: true}))
	require.FileExists(t, "secret.yaml")
	var diff bytes.Buffer
	_, err := DiffTargets(&diff, nil, cfg, DiffRequest{KeyFile: env.keyFile, ColorMode: "never"})
	require.NoError(t, err)
	root := t.TempDir()
	require.NoError(t, DecryptFiles(cfg, DecryptRequest{KeyFile: env.keyFile, OutputDir: root}))
	require.NoError(t, CleanFiles(cfg, CleanRequest{KeyFile: env.keyFile}))
	require.NoFileExists(t, "secret.yaml")
	require.FileExists(t, filepath.Join(root, "secret.yaml"))
	for _, asJSON := range []bool{false, true} {
		var plan bytes.Buffer
		require.NoError(t, PrintPlan(&plan, cfg, PlanRequest{}, presentation.PlanPrintOptions{JSON: asJSON}))
		require.Contains(t, plan.String(), "delivery")
		require.Contains(t, plan.String(), "inplace")
		selection, err := config.ResolveSelection(cfg, config.SelectionOptions{Command: "verify"})
		require.NoError(t, err)
		report, err := verify.Check(selection, config.CurrentDir(cfg), verify.Options{DecryptMode: verify.DecryptDisabled})
		require.NoError(t, err)
		var output bytes.Buffer
		require.NoError(t, presentation.New(&output, nil, false).VerifyReport(report, asJSON))
		require.Contains(t, output.String(), "delivery")
		if asJSON {
			var payload struct {
				FilePairs []struct {
					PlaintextMode string `json:"plaintext_mode"`
				} `json:"file_pairs"`
			}
			require.NoError(t, json.Unmarshal(output.Bytes(), &payload))
			require.Len(t, payload.FilePairs, 2)
			require.Equal(t, "delivery", payload.FilePairs[1].PlaintextMode)
		}
	}
}

func TestDecryptDeliveryUsesDiscoveryRootFromRootAndSubdirectory(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	root, err := os.Getwd()
	require.NoError(t, err)
	output, err := exec.Command("git", "init", "-q", root).CombinedOutput()
	require.NoError(t, err, "%s", output)
	require.NoError(t, os.Mkdir("child", 0755))
	require.NoError(t, os.WriteFile("child/.yewseal.toml", []byte("[[encryption.files]]\nplaintext='secret.yaml'\nencrypted='secret.enc.yaml'\nplaintext_mode='delivery'\n"), 0644))
	ciphertext, err := sopsx.Encrypt([]byte("token: secret\n"), "yaml", []string{env.publicKey})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile("child/secret.enc.yaml", ciphertext, 0644))
	for _, cwd := range []string{root, filepath.Join(root, "child")} {
		require.NoError(t, os.Chdir(cwd))
		cfg, err := config.LoadConfig()
		require.NoError(t, err)
		require.Equal(t, root, cfg.ProjectRoot)
		delivery := t.TempDir()
		require.NoError(t, DecryptFiles(cfg, DecryptRequest{OutputDir: delivery, KeyFile: env.keyFile}))
		require.FileExists(t, filepath.Join(delivery, "child/secret.yaml"))
		require.NoFileExists(t, filepath.Join(delivery, "secret.yaml"))
	}
}

func TestDecryptDeliveryGroupMappingsAndMetadataSnapshot(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Mkdir("config", 0755))
	ciphertext, err := sopsx.Encrypt([]byte("token: secret\n"), "yaml", []string{env.publicKey})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile("config/secret.enc.yaml", ciphertext, 0644))
	cfg := &config.Config{CurrentDir: cwd, ProjectRoot: cwd, Encryption: config.EncryptionConfig{Groups: []config.GroupConfig{{ConfigDir: cwd, Patterns: []string{"config/*.yaml"}}}}}
	root := t.TempDir()
	req := DecryptRequest{Targets: []string{"config"}, OutputDir: root, KeyFile: env.keyFile}
	preflight, err := PreflightDecrypt(cfg, req)
	require.NoError(t, err)
	require.Len(t, preflight.Selection.FilePairs, 1)
	require.Equal(t, filepath.Join(root, "config/secret.yaml"), preflight.Selection.FilePairs[0].PlaintextPath)
	require.Equal(t, filepath.Join(cwd, "config/secret.yaml"), preflight.MetadataPairs[0].PlaintextPath)
	require.Equal(t, "yaml", preflight.Selection.FilePairs[0].Format)
	require.Equal(t, config.PlaintextInplace, preflight.Selection.FilePairs[0].PlaintextMode)
	require.NoError(t, DecryptFiles(cfg, req))
	require.FileExists(t, filepath.Join(root, "config/secret.yaml"))
}
