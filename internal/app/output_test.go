package app

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/presentation"
	"github.com/YewFence/YewSeal/internal/sopsx"
	"github.com/stretchr/testify/require"
)

type failAtOutput struct {
	match             string
	callsAfterFailure int
	failed            bool
	short             bool
}

func (w *failAtOutput) Write(p []byte) (int, error) {
	if w.failed {
		w.callsAfterFailure++
	}
	if w.failed || strings.Contains(string(p), w.match) {
		w.failed = true
		if w.short {
			return 0, nil
		}
		return 0, errReadOutput
	}
	return len(p), nil
}

func TestBatchCompletesDespiteDiagnosticFailure(t *testing.T) {
	for _, action := range []string{"encrypt", "decrypt"} {
		for _, workers := range []int{1, 4} {
			for _, failOn := range []string{"Selected", "SUCCEEDED", "Summary"} {
				for _, short := range []bool{false, true} {
					t.Run(fmt.Sprintf("%s/%d/%s/short=%v", action, workers, failOn, short), func(t *testing.T) {
						env := newAppCryptoTestEnv(t)
						cfg := configWithOwnerRecipient(&config.Config{}, env.publicKey)
						plain := []byte("token: saved\n")
						cipher, err := sopsx.Encrypt(plain, "yaml", []string{env.publicKey})
						require.NoError(t, err)
						for i := range 8 {
							pair := config.FilePair{PlaintextPath: fmt.Sprintf("%d.yaml", i), EncryptedPath: fmt.Sprintf("%d.enc.yaml", i), Format: "yaml"}
							cfg.Encryption.Files = append(cfg.Encryption.Files, pair)
							if action == "encrypt" {
								require.NoError(t, os.WriteFile(pair.PlaintextPath, plain, 0600))
							} else {
								require.NoError(t, os.WriteFile(pair.EncryptedPath, cipher, 0600))
							}
						}
						w := &failAtOutput{match: failOn, short: short}
						var body bytes.Buffer
						out := presentation.New(&body, w, true)
						if action == "encrypt" {
							err = EncryptFiles(cfg, EncryptRequest{Presentation: out, Parallel: workers, UpdateProjectMetadata: true})
						} else {
							err = DecryptFiles(cfg, DecryptRequest{Presentation: out, KeyFile: env.keyFile, Parallel: workers, UpdateProjectMetadata: true})
						}
						want := errReadOutput
						if short {
							want = io.ErrShortWrite
						}
						require.ErrorIs(t, err, want)
						require.True(t, presentation.DiagnosticsFailed(err))
						require.Zero(t, w.callsAfterFailure)
						require.Empty(t, body.String())
						for _, pair := range cfg.Encryption.Files {
							if action == "encrypt" {
								data, readErr := os.ReadFile(pair.EncryptedPath)
								require.NoError(t, readErr)
								decoded, decodeErr := sopsx.Decrypt(data, "yaml", mustTestBundle(t, env.keyFile).String())
								require.NoError(t, decodeErr)
								require.Equal(t, plain, decoded)
							} else {
								data, readErr := os.ReadFile(pair.PlaintextPath)
								require.NoError(t, readErr)
								require.Equal(t, plain, data)
							}
						}
					})
				}
			}
		}
	}
}

func TestDiffContentFailurePreservesBusinessFailure(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	cfg := &config.Config{}
	for _, name := range []string{"broken", "good", "later"} {
		cfg.Encryption.Files = append(cfg.Encryption.Files, config.FilePair{PlaintextPath: name + ".yaml", EncryptedPath: name + ".enc.yaml", Format: "yaml"})
		require.NoError(t, os.WriteFile(name+".yaml", []byte("token: local\n"), 0600))
		cipher, err := sopsx.Encrypt([]byte("token: saved\n"), "yaml", []string{env.publicKey})
		require.NoError(t, err)
		if name == "broken" {
			cipher = []byte("broken")
		}
		require.NoError(t, os.WriteFile(name+".enc.yaml", cipher, 0600))
	}
	result, err := DiffPlaintextAgainstEncryptedTargets(rejectedOutput{}, io.Discard, cfg, nil, env.keyFile, false, "never", false)
	require.ErrorIs(t, err, errReadOutput)
	require.Equal(t, 1, result.Summary.FailedCount)
	require.Equal(t, 1, result.Summary.ComparedCount)
	require.Len(t, result.Summary.Results, 2, "content failure must stop later comparisons")
	require.Contains(t, err.Error(), "failed to compare")
	var delivery *presentation.OutputError
	require.True(t, errors.As(err, &delivery))
}
