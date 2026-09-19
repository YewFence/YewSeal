package presentation

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"testing"

	"github.com/YewFence/YewSeal/internal/diff"
	"github.com/YewFence/YewSeal/internal/task"
	"github.com/stretchr/testify/require"
)

func TestBatchReportJSONRendersSummaryAndFiles(t *testing.T) {
	summary := &task.Summary{
		TotalFiles:            3,
		SuccessCount:          1,
		SkippedCount:          1,
		FailedCount:           1,
		EncryptedCount:        1,
		MissingPlaintextCount: 1,
		Results: []task.Result{
			{SourceFile: "configs/app.toml", TargetFile: "configs/app.enc.toml", Status: task.Succeeded, Outcome: task.OutcomeEncrypted},
			{SourceFile: "gone.yaml", TargetFile: "gone.enc.yaml", Status: task.Skipped, Outcome: task.OutcomeMissingPlaintext},
			{SourceFile: "bad.json", TargetFile: "bad.enc.json", Status: task.Failed, Outcome: task.OutcomeProcessed, Error: errors.New("failed to encrypt")},
		},
	}

	var out bytes.Buffer
	require.NoError(t, New(&out, &bytes.Buffer{}, false).BatchReportJSON("encrypt", summary))

	var payload struct {
		Command string `json:"command"`
		Summary struct {
			Total     int `json:"total"`
			Succeeded int `json:"succeeded"`
			Skipped   int `json:"skipped"`
			Failed    int `json:"failed"`
			Encrypted int `json:"encrypted"`
		} `json:"summary"`
		Files []struct {
			Plaintext string `json:"plaintext"`
			Encrypted string `json:"encrypted"`
			Status    string `json:"status"`
			Outcome   string `json:"outcome"`
			Error     string `json:"error"`
		} `json:"files"`
	}
	require.NoError(t, json.Unmarshal(out.Bytes(), &payload))
	require.Equal(t, "encrypt", payload.Command)
	require.Equal(t, 3, payload.Summary.Total)
	require.Equal(t, 1, payload.Summary.Succeeded)
	require.Equal(t, 1, payload.Summary.Skipped)
	require.Equal(t, 1, payload.Summary.Failed)
	require.Equal(t, 1, payload.Summary.Encrypted)
	require.Len(t, payload.Files, 3)
	require.Equal(t, "configs/app.toml", payload.Files[0].Plaintext)
	require.Equal(t, "succeeded", payload.Files[0].Status)
	require.Equal(t, "encrypted", payload.Files[0].Outcome)
	require.Equal(t, "failed", payload.Files[2].Status)
	require.Contains(t, payload.Files[2].Error, "failed to encrypt")
}

func TestViewReportJSONEncodesBinaryAsBase64(t *testing.T) {
	text := []byte("token: value\n")
	var out bytes.Buffer
	require.NoError(t, New(&out, &bytes.Buffer{}, false).ViewReportJSON("config.enc.toml", "toml", text))
	var envelope struct {
		Format   string `json:"format"`
		Encoding string `json:"encoding"`
		Content  string `json:"content"`
	}
	require.NoError(t, json.Unmarshal(out.Bytes(), &envelope))
	require.Equal(t, "toml", envelope.Format)
	require.Equal(t, "utf-8", envelope.Encoding)
	require.Equal(t, string(text), envelope.Content)

	out.Reset()
	blob := []byte{0x89, 0x50, 0x0d}
	require.NoError(t, New(&out, &bytes.Buffer{}, false).ViewReportJSON("logo.enc.bin", "binary", blob))
	require.NoError(t, json.Unmarshal(out.Bytes(), &envelope))
	require.Equal(t, "binary", envelope.Format)
	require.Equal(t, "base64", envelope.Encoding)
	require.Equal(t, base64.StdEncoding.EncodeToString(blob), envelope.Content)
}

func TestDiffReportJSONEmbedsBodiesAndSkipReasons(t *testing.T) {
	var summary diff.Summary
	summary.Add("config.yaml", "config.enc.yaml", diff.DiffResult{Diff: "--- old\n+++ new", Different: true}, nil)
	summary.Add("same.yaml", "same.enc.yaml", diff.DiffResult{}, nil)
	summary.Add("gone.yaml", "gone.enc.yaml", diff.DiffResult{Skipped: diff.MissingPlaintext}, nil)
	summary.Add("bad.yaml", "bad.enc.yaml", diff.DiffResult{}, errors.New("read failure"))

	var out bytes.Buffer
	require.NoError(t, New(&out, &bytes.Buffer{}, false).DiffReportJSON(summary))

	var payload struct {
		Summary struct {
			Total        int `json:"total"`
			Compared     int `json:"compared"`
			MissingInput int `json:"missing_input"`
			Failed       int `json:"failed"`
		} `json:"summary"`
		Files []struct {
			Changed *bool  `json:"changed"`
			Diff    string `json:"diff"`
			Skipped string `json:"skipped"`
			Error   string `json:"error"`
		} `json:"files"`
	}
	require.NoError(t, json.Unmarshal(out.Bytes(), &payload))
	require.Equal(t, 4, payload.Summary.Total)
	require.Equal(t, 2, payload.Summary.Compared)
	require.Equal(t, 1, payload.Summary.MissingInput)
	require.Equal(t, 1, payload.Summary.Failed)
	require.Len(t, payload.Files, 4)
	require.NotNil(t, payload.Files[0].Changed)
	require.True(t, *payload.Files[0].Changed)
	require.Contains(t, payload.Files[0].Diff, "+++ new")
	require.NotNil(t, payload.Files[1].Changed)
	require.False(t, *payload.Files[1].Changed)
	require.Equal(t, "missing-plaintext", payload.Files[2].Skipped)
	require.Nil(t, payload.Files[2].Changed)
	require.Contains(t, payload.Files[3].Error, "read failure")
}

func TestInitReportJSONDistinguishesKeptFromFresh(t *testing.T) {
	var out bytes.Buffer
	require.NoError(t, New(&out, &bytes.Buffer{}, false).InitReportJSON(InitReport{Kept: true}))
	var kept struct {
		Kept     bool  `json:"kept"`
		Mappings []any `json:"mappings"`
	}
	require.NoError(t, json.Unmarshal(out.Bytes(), &kept))
	require.True(t, kept.Kept)
	require.Empty(t, kept.Mappings)

	out.Reset()
	report := InitReport{
		ConfigFile: ".yewseal.toml",
		KeyFile:    ".age/keys.txt",
		SOPSConfig: true,
		Mappings:   []InitMapping{{Plaintext: "config.toml", Encrypted: "config.enc.toml", Format: "toml", Recipients: []string{"owner"}}},
	}
	require.NoError(t, New(&out, &bytes.Buffer{}, false).InitReportJSON(report))
	var fresh struct {
		Kept     bool `json:"kept"`
		SOPS     bool `json:"sops_config"`
		Mappings []struct {
			Plaintext  string   `json:"plaintext"`
			Encrypted  string   `json:"encrypted"`
			Format     string   `json:"format"`
			Recipients []string `json:"recipients"`
		} `json:"mappings"`
	}
	require.NoError(t, json.Unmarshal(out.Bytes(), &fresh))
	require.False(t, fresh.Kept)
	require.True(t, fresh.SOPS)
	require.Len(t, fresh.Mappings, 1)
	require.Equal(t, "config.enc.toml", fresh.Mappings[0].Encrypted)
	require.Equal(t, []string{"owner"}, fresh.Mappings[0].Recipients)
}
