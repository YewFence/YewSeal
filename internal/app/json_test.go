package app

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/presentation"
	"github.com/stretchr/testify/require"
)

func TestBatchViewDiffJSONReportsOnContentStreamOnly(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	require.NoError(t, os.WriteFile("config.yaml", []byte("token: value\n"), 0600))
	cfg := configWithOwnerRecipient(&config.Config{Encryption: config.EncryptionConfig{Files: []config.FilePair{{PlaintextPath: "config.yaml", EncryptedPath: "config.enc.yaml", Format: "yaml"}}}}, env.publicKey)

	var stdout, stderr bytes.Buffer
	out := presentation.New(&stdout, &stderr, false)
	require.NoError(t, EncryptFiles(cfg, EncryptRequest{Presentation: out, KeyFile: env.keyFile, Targets: []string{"config.yaml"}, Parallel: 1, JSON: true}))
	var encryptPayload struct {
		Command string `json:"command"`
		Files   []struct {
			Status  string `json:"status"`
			Outcome string `json:"outcome"`
		} `json:"files"`
	}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &encryptPayload))
	require.Equal(t, "encrypt", encryptPayload.Command)
	require.Len(t, encryptPayload.Files, 1)
	require.Equal(t, "succeeded", encryptPayload.Files[0].Status)
	require.Contains(t, stderr.String(), "Summary (encrypted)")

	require.NoError(t, os.Remove("config.yaml"))
	stdout.Reset()
	require.NoError(t, DecryptFiles(cfg, DecryptRequest{Presentation: out, KeyFile: env.keyFile, Targets: []string{"config.enc.yaml"}, Parallel: 1, JSON: true}))
	var decryptPayload struct {
		Command string `json:"command"`
		Files   []struct {
			Plaintext string `json:"plaintext"`
			Encrypted string `json:"encrypted"`
		} `json:"files"`
		Summary struct {
			Succeeded int `json:"succeeded"`
		} `json:"summary"`
	}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &decryptPayload))
	require.Equal(t, "decrypt", decryptPayload.Command)
	require.Equal(t, 1, decryptPayload.Summary.Succeeded)
	require.Equal(t, "config.yaml", decryptPayload.Files[0].Plaintext)
	require.Equal(t, "config.enc.yaml", decryptPayload.Files[0].Encrypted)

	stdout.Reset()
	require.NoError(t, ViewTarget(&stdout, &stderr, cfg, ViewRequest{Target: "config.enc.yaml", KeyFile: env.keyFile, JSON: true}))
	var viewPayload struct {
		Path    string `json:"path"`
		Format  string `json:"format"`
		Content string `json:"content"`
	}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &viewPayload))
	require.Equal(t, "config.enc.yaml", viewPayload.Path)
	require.Equal(t, "yaml", viewPayload.Format)
	require.Equal(t, "token: value\n", viewPayload.Content)

	require.NoError(t, os.WriteFile("config.yaml", []byte("token: changed\n"), 0600))
	stdout.Reset()
	_, err := DiffTargets(&stdout, &stderr, cfg, DiffRequest{KeyFile: env.keyFile, ColorMode: "never", JSON: true})
	require.NoError(t, err)
	var diffPayload struct {
		Files []struct {
			Changed *bool  `json:"changed"`
			Diff    string `json:"diff"`
		} `json:"files"`
	}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &diffPayload))
	require.Len(t, diffPayload.Files, 1)
	require.NotNil(t, diffPayload.Files[0].Changed)
	require.True(t, *diffPayload.Files[0].Changed)
	require.Contains(t, diffPayload.Files[0].Diff, "token")
}
