package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/errx"
	"github.com/YewFence/YewSeal/internal/presentation"
	"github.com/YewFence/YewSeal/internal/sopsx"
	"github.com/stretchr/testify/require"
)

func TestDecryptExplicitDeliveryTargetSkipsUntilInplaceConsent(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	ciphertext, err := sopsx.Encrypt([]byte("token: secret\n"), "yaml", []string{env.publicKey})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile("delivery.enc.yaml", ciphertext, 0644))
	cfg := &config.Config{Encryption: config.EncryptionConfig{Files: []config.FilePair{{PlaintextPath: "delivery.yaml", EncryptedPath: "delivery.enc.yaml", PlaintextMode: config.PlaintextDelivery}}}}
	for _, strict := range []bool{false, true} {
		var stdout, stderr bytes.Buffer
		err := DecryptFiles(cfg, DecryptRequest{Targets: []string{"delivery.enc.yaml"}, KeyFile: env.keyFile, Strict: strict, Presentation: presentation.New(&stdout, &stderr, false)})
		if strict {
			require.ErrorContains(t, err, "1 of 1 files skipped")
		} else {
			require.NoError(t, err)
		}
		require.Empty(t, stdout.String())
		require.Contains(t, stderr.String(), "0 succeeded, 1 skipped, 0 failed (1 selected)")
		require.NoFileExists(t, "delivery.yaml")
	}
	require.NoError(t, DecryptFiles(cfg, DecryptRequest{Targets: []string{"delivery.enc.yaml"}, KeyFile: env.keyFile, Inplace: true, Strict: true}))
	actual, err := os.ReadFile("delivery.yaml")
	require.NoError(t, err)
	require.Equal(t, "token: secret\n", string(actual))
}

func TestDecryptSkipsDeliveryWithoutBlockingScope(t *testing.T) {
	for _, parallel := range []int{1, 4} {
		for _, strict := range []bool{false, true} {
			t.Run(fmt.Sprintf("parallel=%d/strict=%t", parallel, strict), func(t *testing.T) {
				env := newAppCryptoTestEnv(t)
				ciphertext, err := sopsx.Encrypt([]byte("token: secret\n"), "yaml", []string{env.publicKey})
				require.NoError(t, err)
				require.NoError(t, os.WriteFile("local.enc.yaml", ciphertext, 0644))
				cfg := &config.Config{Encryption: config.EncryptionConfig{Files: []config.FilePair{
					{PlaintextPath: "local.yaml", EncryptedPath: "local.enc.yaml"},
					{PlaintextPath: "delivery.yaml", EncryptedPath: "delivery.enc.yaml", PlaintextMode: config.PlaintextDelivery},
				}}}
				var stdout, stderr bytes.Buffer
				err = DecryptFiles(cfg, DecryptRequest{KeyFile: env.keyFile, Parallel: parallel, Strict: strict, JSON: true, Presentation: presentation.New(&stdout, &stderr, false)})
				if strict {
					require.ErrorContains(t, err, "1 of 2 files skipped")
					var usage *errx.UsageError
					require.False(t, errors.As(err, &usage))
				} else {
					require.NoError(t, err)
				}
				require.FileExists(t, "local.yaml")
				require.NoFileExists(t, "delivery.yaml")
				require.Contains(t, stderr.String(), "SKIPPED delivery.enc.yaml")
				require.Contains(t, stderr.String(), "--inplace")
				require.Contains(t, stderr.String(), "--output DIR")
				require.Contains(t, stderr.String(), "1 succeeded, 1 skipped, 0 failed (2 selected)")
				var report struct {
					Summary struct{ Total, Succeeded, Skipped, Failed int } `json:"summary"`
					Files   []struct{ Status, Outcome, Error string }       `json:"files"`
				}
				require.NoError(t, json.Unmarshal(stdout.Bytes(), &report))
				require.Equal(t, 2, report.Summary.Total)
				require.Equal(t, 1, report.Summary.Succeeded)
				require.Equal(t, 1, report.Summary.Skipped)
				require.Zero(t, report.Summary.Failed)
				require.Len(t, report.Files, 2)
				require.Equal(t, "succeeded", report.Files[0].Status)
				require.Equal(t, "skipped", report.Files[1].Status)
				require.Equal(t, "delivery-restricted", report.Files[1].Outcome)
				require.Empty(t, report.Files[1].Error)
			})
		}
	}
}
