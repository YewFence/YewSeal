package app

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

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
	cfg := &config.Config{CurrentDir: tempDir, Encryption: config.EncryptionConfig{Files: []config.FilePair{{PlaintextPath: ".dev.vars", EncryptedPath: ".dev.vars.enc.yaml", Format: "env"}}}}
	var out bytes.Buffer
	err = PrintPlan(&out, cfg, PlanRequest{
		Target: ".dev.vars",
		Format: "env",
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
			SelectedBy string `json:"selected_by"`
		} `json:"file_pairs"`
	}
	require.NoError(t, json.Unmarshal(out.Bytes(), &payload))
	assert.Equal(t, "plan", payload.Command)
	require.Len(t, payload.FilePairs, 1)
	assert.Equal(t, ".dev.vars", payload.FilePairs[0].Plaintext.Display)
	assert.Equal(t, "exact", payload.FilePairs[0].Plaintext.Source.Kind)
	assert.Equal(t, "env", payload.FilePairs[0].Format.Value)
	assert.Equal(t, "argument", payload.FilePairs[0].Format.Source.Kind)
	assert.Equal(t, "path-target", payload.FilePairs[0].SelectedBy)
}
