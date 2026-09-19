package app

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"filippo.io/age"
	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/sopsx"
	"github.com/stretchr/testify/require"
)

func TestDiffReadOutcomes(t *testing.T) {
	for _, tc := range []struct {
		name       string
		inputs     []string
		compared   int
		missing    int
		noIdentity int
		failed     int
	}{
		{name: "equal", inputs: []string{"equal"}, compared: 1},
		{name: "different", inputs: []string{"different"}, compared: 1},
		{name: "missing", inputs: []string{"no-plain", "no-cipher", "neither"}, missing: 3},
		{name: "unavailable", inputs: []string{"unavailable"}, noIdentity: 1},
		{name: "partial-missing", inputs: []string{"no-cipher", "different"}, compared: 1, missing: 1},
		{name: "partial-identity", inputs: []string{"unavailable", "different"}, compared: 1, noIdentity: 1},
		{name: "mixed-skips", inputs: []string{"no-plain", "unavailable"}, missing: 1, noIdentity: 1},
		{name: "broken-continues", inputs: []string{"broken", "different"}, compared: 1, failed: 1},
		{name: "bad-input-continues", inputs: []string{"directory", "different"}, compared: 1, failed: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env := newAppCryptoTestEnv(t)
			other, err := age.GenerateX25519Identity()
			require.NoError(t, err)
			cfg := &config.Config{}
			for i, input := range tc.inputs {
				plain, encrypted := fmt.Sprintf("%d.yaml", i), fmt.Sprintf("%d.enc.yaml", i)
				cfg.Encryption.Files = append(cfg.Encryption.Files, config.FilePair{PlaintextPath: plain, EncryptedPath: encrypted})
				if input != "no-plain" && input != "neither" {
					data := []byte("token: value\n")
					if input == "different" {
						data = []byte("token: local\n")
					}
					if input == "directory" {
						require.NoError(t, os.Mkdir(plain, 0700))
					} else {
						require.NoError(t, os.WriteFile(plain, data, 0600))
					}
				}
				if input != "no-cipher" && input != "neither" {
					key := env.publicKey
					if input == "unavailable" {
						key = other.Recipient().String()
					}
					data, err := sopsx.Encrypt([]byte("token: value\n"), "yaml", []string{key})
					require.NoError(t, err)
					if input == "broken" || input == "no-plain" {
						data = []byte("not ciphertext")
					}
					require.NoError(t, os.WriteFile(encrypted, data, 0600))
				}
			}
			var out, diagnostics bytes.Buffer
			result, err := DiffTargets(&out, &diagnostics, cfg, DiffRequest{KeyFile: env.keyFile, Verbose: true, ColorMode: "never"})
			if tc.failed > 0 {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tc.compared, result.Summary.ComparedCount)
			require.Equal(t, tc.missing, result.Summary.MissingInputCount)
			require.Equal(t, tc.noIdentity, result.Summary.NoIdentityCount)
			require.Equal(t, tc.failed, result.Summary.FailedCount)
			if strings.Contains(strings.Join(tc.inputs, ","), "different") {
				require.True(t, result.Different)
				require.Contains(t, out.String(), "@@\n-token: local\n+token: value\n")
			} else {
				require.Empty(t, out.String())
			}
			require.NotContains(t, out.String(), "SKIPPED")
			require.NotContains(t, out.String(), "Selected")
			require.Contains(t, diagnostics.String(), "Summary:")
			require.NoFileExists(t, ".gitignore")
			require.NoFileExists(t, ".sops.yaml")
		})
	}
}

func TestReadCommandsPreserveHistoryAndOutputChannels(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	plain := []byte("TOKEN=historical\n")
	cipher, err := sopsx.Encrypt(plain, "env", []string{env.publicKey})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile("secrets", cipher, 0600))
	missing := []string{"deleted"}
	cfg := &config.Config{Encryption: config.EncryptionConfig{Files: []config.FilePair{{PlaintextPath: ".dev.vars", EncryptedPath: "secrets", Format: "env", Recipients: &missing}}}}
	for _, target := range []string{".dev.vars", "secrets"} {
		var out, diagnostics bytes.Buffer
		require.NoError(t, ViewTarget(&out, &diagnostics, cfg, ViewRequest{Target: target, KeyFile: env.keyFile, Verbose: true}))
		require.Equal(t, plain, out.Bytes())
		require.Contains(t, diagnostics.String(), "WARNING")
		require.Contains(t, diagnostics.String(), "Selected 1")
		require.NoFileExists(t, ".dev.vars")
	}
	require.NoError(t, os.WriteFile(".dev.vars", []byte("TOKEN=local\n"), 0600))
	var out, diagnostics bytes.Buffer
	result, err := DiffTargets(&out, &diagnostics, cfg, DiffRequest{Targets: []string{"secrets"}, KeyFile: env.keyFile, Verbose: true, ColorMode: "never"})
	require.NoError(t, err)
	require.True(t, result.Different)
	require.Contains(t, diagnostics.String(), "WARNING")
	require.Equal(t, "--- .dev.vars\n+++ secrets (decrypted)\n@@\n-TOKEN=local\n+TOKEN=historical\n", out.String())
	data, err := os.ReadFile(".dev.vars")
	require.NoError(t, err)
	require.Equal(t, "TOKEN=local\n", string(data))
	data, err = os.ReadFile("secrets")
	require.NoError(t, err)
	require.Equal(t, cipher, data)
	require.NoFileExists(t, ".gitignore")
	require.NoFileExists(t, ".sops.yaml")
}

func TestDiffAlwaysLoadsIdentityBeforeMissingInputSkips(t *testing.T) {
	t.Chdir(t.TempDir())
	cfg := &config.Config{Encryption: config.EncryptionConfig{Files: []config.FilePair{{PlaintextPath: "missing.yaml", EncryptedPath: "missing.enc.yaml"}}}}
	var out, diagnostics bytes.Buffer
	_, err := DiffTargets(&out, &diagnostics, cfg, DiffRequest{KeyFile: filepath.Join(t.TempDir(), "missing-key"), ColorMode: "never"})
	require.Error(t, err)
	require.Empty(t, out.String())
	require.Empty(t, diagnostics.String())
}

var errReadOutput = errors.New("output unavailable")

type rejectedOutput struct {
	match string
}

func (w rejectedOutput) Write(data []byte) (int, error) {
	if strings.Contains(string(data), w.match) {
		return 0, errReadOutput
	}
	return len(data), nil
}

func TestReadCommandsPropagateOutputErrors(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	cipher, err := sopsx.Encrypt([]byte("token: saved\n"), "yaml", []string{env.publicKey})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile("config.enc.yaml", cipher, 0600))
	require.NoError(t, os.WriteFile("config.yaml", []byte("token: changed\n"), 0600))
	aliases := []string{"deleted"}
	cfg := &config.Config{Encryption: config.EncryptionConfig{Files: []config.FilePair{{PlaintextPath: "config.yaml", EncryptedPath: "config.enc.yaml", Recipients: &aliases}}}}
	for _, command := range []string{"view", "diff"} {
		for _, match := range []string{"WARNING", "Selected", ""} {
			t.Run(command+"/"+match, func(t *testing.T) {
				var out bytes.Buffer
				if command == "view" {
					err = ViewTarget(&out, rejectedOutput{match: match}, cfg, ViewRequest{Target: "config.yaml", KeyFile: env.keyFile, Verbose: true})
				} else {
					_, err = DiffTargets(&out, rejectedOutput{match: match}, cfg, DiffRequest{KeyFile: env.keyFile, Verbose: true, ColorMode: "never"})
				}
				require.ErrorIs(t, err, errReadOutput)
				require.NotEmpty(t, out.String(), "diagnostic failure must not stop content delivery")
			})
		}
	}
	require.ErrorIs(t, ViewTarget(rejectedOutput{}, io.Discard, cfg, ViewRequest{Target: "config.yaml", KeyFile: env.keyFile}), errReadOutput)
	_, err = DiffTargets(rejectedOutput{}, io.Discard, cfg, DiffRequest{KeyFile: env.keyFile, ColorMode: "never"})
	require.ErrorIs(t, err, errReadOutput)
	_, err = DiffTargets(io.Discard, rejectedOutput{match: "Summary"}, cfg, DiffRequest{KeyFile: env.keyFile, ColorMode: "never"})
	require.ErrorIs(t, err, errReadOutput)
}
