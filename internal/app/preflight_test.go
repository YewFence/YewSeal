package app

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"filippo.io/age"
	"github.com/YewFence/YewSeal/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrintPlanJSONDoesNotRequireKeys(t *testing.T) {
	tempDir := t.TempDir()
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tempDir))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(oldWd))
	})
	require.NoError(t, os.WriteFile(".dev.vars", []byte("TOKEN=secret\n"), 0644))
	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	defaults := []string{"owner"}
	cfg := &config.Config{CurrentDir: tempDir, Recipients: config.RecipientConfig{Defaults: &defaults, Registry: map[string]string{"owner": identity.Recipient().String()}}, Encryption: config.EncryptionConfig{Files: []config.FilePair{{PlaintextPath: ".dev.vars", EncryptedPath: ".dev.vars.enc.yaml", Format: "env", ConfigPath: ".yewseal.toml"}}}}
	var out bytes.Buffer
	err = PrintPlan(&out, cfg, PlanRequest{
		Target: ".dev.vars",
	}, PreflightPrintOptions{JSON: true})
	require.NoError(t, err)

	var payload struct {
		Command   string `json:"command"`
		FilePairs []struct {
			Plaintext struct {
				Display string `json:"display"`
				Source  struct {
					Kind string `json:"kind"`
				} `json:"source"`
			} `json:"plaintext"`
			Format struct {
				Value  string `json:"value"`
				Source struct {
					Kind string `json:"kind"`
				} `json:"source"`
			} `json:"format"`
			RecipientAliases []string `json:"recipient_aliases"`
			Recipients       []string `json:"recipients"`
			Authorization    struct {
				Kind            string `json:"kind"`
				EffectiveSource struct {
					Kind string `json:"kind"`
				} `json:"effective_source"`
			} `json:"authorization"`
			SelectedBy string `json:"selected_by"`
		} `json:"file_pairs"`
	}
	require.NoError(t, json.Unmarshal(out.Bytes(), &payload))
	assert.Equal(t, "plan", payload.Command)
	require.Len(t, payload.FilePairs, 1)
	assert.Equal(t, ".dev.vars", payload.FilePairs[0].Plaintext.Display)
	assert.Equal(t, "exact", payload.FilePairs[0].Plaintext.Source.Kind)
	assert.Equal(t, "env", payload.FilePairs[0].Format.Value)
	assert.Equal(t, "config-format", payload.FilePairs[0].Format.Source.Kind)
	assert.Equal(t, "path-target", payload.FilePairs[0].SelectedBy)
	assert.Equal(t, []string{"owner"}, payload.FilePairs[0].RecipientAliases)
	assert.Equal(t, []string{identity.Recipient().String()}, payload.FilePairs[0].Recipients)
	assert.Equal(t, "defaults", payload.FilePairs[0].Authorization.Kind)
	assert.Equal(t, "defaults", payload.FilePairs[0].Authorization.EffectiveSource.Kind)
}

func TestOutputOverridePreservesInferredFormatProvenance(t *testing.T) {
	env := newAppCryptoTestEnv(t)
	require.NoError(t, os.WriteFile("config.yaml", []byte("token: value\n"), 0600))
	require.NoError(t, os.WriteFile("config.enc.yaml", []byte("unused in preflight"), 0600))
	cfg := configWithOwnerRecipient(&config.Config{Encryption: config.EncryptionConfig{Files: []config.FilePair{{PlaintextPath: "config.yaml", EncryptedPath: "config.enc.yaml", ConfigPath: ".yewseal.toml"}}}}, env.publicKey)
	for _, target := range []string{"config.yaml", "config.enc.yaml"} {
		selection, err := config.ResolvePlanSelection(cfg, config.SelectionOptions{Target: target, Output: "export.json", OutputSet: true})
		require.NoError(t, err)
		require.Len(t, selection.FilePairs, 1)
		require.Equal(t, "yaml", selection.FilePairs[0].Format)
		require.Equal(t, config.ValueSourceFilename, selection.FilePairs[0].FormatSource.Kind)
	}
}
