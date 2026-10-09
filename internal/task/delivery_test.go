package task

import (
	"fmt"
	"os"
	"testing"

	"github.com/YewFence/YewSeal/internal/sopsx"
	"github.com/stretchr/testify/require"
)

func TestDecryptDeliverySkipsBeforeInputChecksAndPreservesPlaintext(t *testing.T) {
	for _, parallel := range []int{1, 4} {
		for _, strict := range []bool{false, true} {
			t.Run(fmt.Sprintf("parallel=%d/strict=%t", parallel, strict), func(t *testing.T) {
				t.Chdir(t.TempDir())
				stale := []byte("local changes\n")
				require.NoError(t, os.WriteFile("stale.yaml", stale, 0644))
				require.NoError(t, os.WriteFile("broken.enc.yaml", []byte("not SOPS"), 0600))
				pairs := []FilePair{
					{PlaintextPath: "absent/secret.yaml", EncryptedPath: "missing.enc.yaml", Format: "yaml", PlaintextMode: "delivery"},
					{PlaintextPath: "stale.yaml", EncryptedPath: "broken.enc.yaml", Format: "yaml", PlaintextMode: "delivery"},
				}
				var completed []Result
				summary, err := Decrypt(Options{FilePairs: pairs, Parallel: parallel, Strict: strict, Force: true, OnComplete: func(result Result) { completed = append(completed, result) }})
				if strict {
					require.ErrorContains(t, err, "2 of 2 files skipped")
				} else {
					require.NoError(t, err)
				}
				require.Equal(t, 2, summary.TotalFiles)
				require.Equal(t, 2, summary.SkippedCount)
				require.Zero(t, summary.FailedCount)
				require.Zero(t, summary.SuccessCount)
				require.Len(t, completed, 2)
				for _, results := range [][]Result{summary.Results, completed} {
					for _, result := range results {
						require.Equal(t, Skipped, result.Status)
						require.Equal(t, OutcomeDeliveryRestricted, result.Outcome)
						require.NoError(t, result.Error)
					}
				}
				require.NoDirExists(t, "absent")
				actual, err := os.ReadFile("stale.yaml")
				require.NoError(t, err)
				require.Equal(t, stale, actual)
				info, err := os.Stat("stale.yaml")
				require.NoError(t, err)
				require.Equal(t, os.FileMode(0644), info.Mode().Perm())
			})
		}
	}
}

func TestDecryptDeliveryConsentProcessesCiphertext(t *testing.T) {
	for _, consent := range []string{"inplace", "output"} {
		t.Run(consent, func(t *testing.T) {
			_, publicKey, bundle := setupBatchTestEnv(t)
			plain := []byte("token: value\n")
			ciphertext, err := sopsx.Encrypt(plain, "yaml", []string{publicKey})
			require.NoError(t, err)
			require.NoError(t, os.WriteFile("secret.enc.yaml", ciphertext, 0600))
			summary, err := Decrypt(Options{
				FilePairs:      []FilePair{{PlaintextPath: "secret.yaml", EncryptedPath: "secret.enc.yaml", Format: "yaml", PlaintextMode: "delivery"}},
				IdentityBundle: bundle, Inplace: consent == "inplace", Delivery: consent == "output", Strict: true,
			})
			require.NoError(t, err)
			require.Equal(t, 1, summary.SuccessCount)
			require.Zero(t, summary.SkippedCount)
			require.Equal(t, OutcomeProcessed, summary.Results[0].Outcome)
			actual, err := os.ReadFile("secret.yaml")
			require.NoError(t, err)
			require.Equal(t, plain, actual)
		})
	}
}
