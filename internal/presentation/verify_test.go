package presentation

import (
	"bytes"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/verify"
	"github.com/stretchr/testify/require"
)

func TestVerifyReportJSONIncludesFullRecipient(t *testing.T) {
	var body bytes.Buffer
	report := &verify.Report{}
	report.Add(verify.Finding{Code: "recipient_extra", Severity: verify.SeverityError, Recipient: "age1completepublickey", Message: "unknown recipient"})
	require.NoError(t, New(&body, nil, false).VerifyReport(report, true))
	var result struct {
		Findings []struct {
			Recipient string `json:"recipient"`
		} `json:"findings"`
	}
	require.NoError(t, json.Unmarshal(body.Bytes(), &result))
	require.Len(t, result.Findings, 1)
	require.Equal(t, "age1completepublickey", result.Findings[0].Recipient)
}

func TestVerifyReportPreservesMappingModesAndFindingPaths(t *testing.T) {
	cwd := t.TempDir()
	plain := filepath.Join(cwd, "secrets", "prod.env")
	encrypted := filepath.Join(cwd, "secrets", "prod.enc.env")
	report := &verify.Report{FilePairs: []config.ResolvedFilePair{
		{PlaintextPath: plain, EncryptedPath: encrypted, PlaintextMode: "delivery"},
		{PlaintextPath: filepath.Join(cwd, "config.toml"), EncryptedPath: filepath.Join(cwd, "config.enc.toml"), PlaintextMode: "inplace"},
	}}
	for _, finding := range []verify.Finding{
		{Code: "plaintext_drift", Severity: verify.SeverityError, PlaintextPath: plain, EncryptedPath: encrypted, Message: "content differs", Hint: "inspect the difference"},
		{Code: "plaintext_not_ignored", Severity: verify.SeverityError, PlaintextPath: plain, Message: "not ignored"},
		{Code: "ciphertext_missing", Severity: verify.SeverityError, EncryptedPath: encrypted, Message: "missing"},
	} {
		report.Add(finding)
	}
	report.AddSkip("decryption disabled")
	for _, asJSON := range []bool{false, true} {
		var body, diagnostics bytes.Buffer
		out := New(&body, &diagnostics, false)
		out.SetDirectory(cwd)
		require.NoError(t, out.Finish(out.VerifyReport(report, asJSON)))
		require.Empty(t, diagnostics.String())
		if !asJSON {
			require.Equal(t, "Summary: 0 passed, 0 warnings, 3 errors, 1 skipped\n"+
				"secrets/prod.env -> secrets/prod.enc.env  plaintext_mode=delivery\n"+
				"config.toml -> config.enc.toml  plaintext_mode=inplace\n"+
				"SKIPPED decryption disabled\n"+
				"ERROR [plaintext_drift] secrets/prod.env -> secrets/prod.enc.env: content differs\n"+
				"  hint: inspect the difference\n"+
				"ERROR [plaintext_not_ignored] secrets/prod.env: not ignored\n"+
				"ERROR [ciphertext_missing] secrets/prod.enc.env: missing\n", filepath.ToSlash(body.String()))
			continue
		}
		var result verifyReportJSON
		require.NoError(t, json.Unmarshal(body.Bytes(), &result))
		require.False(t, result.OK)
		require.Equal(t, verifySummaryJSON{Error: 3, Skipped: 1}, result.Summary)
		require.Equal(t, []string{"decryption disabled"}, result.Skipped)
		require.Equal(t, []verifyPairJSON{
			{PlaintextPath: filepath.Join("secrets", "prod.env"), EncryptedPath: filepath.Join("secrets", "prod.enc.env"), PlaintextMode: "delivery"},
			{PlaintextPath: "config.toml", EncryptedPath: "config.enc.toml", PlaintextMode: "inplace"},
		}, result.FilePairs)
		require.Len(t, result.Findings, 3)
		require.Equal(t, filepath.Join("secrets", "prod.env"), result.Findings[0].PlaintextPath)
		require.Equal(t, filepath.Join("secrets", "prod.enc.env"), result.Findings[0].EncryptedPath)
		require.Equal(t, "inspect the difference", result.Findings[0].Hint)
		require.Equal(t, filepath.Join("secrets", "prod.env"), result.Findings[1].PlaintextPath)
		require.Empty(t, result.Findings[1].EncryptedPath)
		require.Empty(t, result.Findings[2].PlaintextPath)
		require.Equal(t, filepath.Join("secrets", "prod.enc.env"), result.Findings[2].EncryptedPath)
		require.NotContains(t, body.String(), cwd)
	}
}

func TestVerifyReportPropagatesOutputFailure(t *testing.T) {
	failure := errors.New("report stream closed")
	for _, asJSON := range []bool{false, true} {
		writer := &failingWriter{err: failure}
		out := New(writer, nil, false)
		err := out.Finish(out.VerifyReport(&verify.Report{FilePairs: []config.ResolvedFilePair{{PlaintextMode: "delivery"}}}, asJSON))
		require.ErrorIs(t, err, failure)
		require.Equal(t, 1, writer.calls)
		require.True(t, strings.Contains(err.Error(), "content"))
	}
}
