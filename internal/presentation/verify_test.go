package presentation

import (
	"bytes"
	"encoding/json"
	"testing"

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
