package presentation

import (
	"fmt"
	"strings"

	"github.com/YewFence/YewSeal/internal/verify"
)

// VerifyReport renders a verify report on the content stream.
func (o *Output) VerifyReport(report *verify.Report, asJSON bool) error {
	if asJSON {
		return o.verifyReportJSON(report)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Summary: %d passed, %d warnings, %d errors, %d skipped\n",
		report.PassCount, report.WarningCount, report.ErrorCount, report.SkipCount)
	for _, reason := range report.SkipReasons {
		fmt.Fprintf(&b, "SKIPPED %s\n", reason)
	}
	for _, f := range report.Findings {
		fmt.Fprintf(&b, "%s [%s]%s: %s\n", strings.ToUpper(string(f.Severity)), f.Code, o.findingLocation(f), f.Message)
		if f.Hint != "" {
			fmt.Fprintf(&b, "  hint: %s\n", f.Hint)
		}
	}
	_, err := o.Write([]byte(b.String()))
	return err
}

func (o *Output) findingLocation(f verify.Finding) string {
	switch {
	case f.PlaintextPath != "" && f.EncryptedPath != "":
		return " " + o.path(f.PlaintextPath) + " -> " + o.path(f.EncryptedPath)
	case f.PlaintextPath != "":
		return " " + o.path(f.PlaintextPath)
	case f.EncryptedPath != "":
		return " " + o.path(f.EncryptedPath)
	}
	return ""
}

func (o *Output) verifyReportJSON(report *verify.Report) error {
	findings := make([]verifyFindingJSON, 0, len(report.Findings))
	for _, f := range report.Findings {
		entry := verifyFindingJSON{
			Code:      f.Code,
			Severity:  string(f.Severity),
			Recipient: f.Recipient,
			Message:   f.Message,
			Hint:      f.Hint,
		}
		if f.PlaintextPath != "" {
			entry.PlaintextPath = o.path(f.PlaintextPath)
		}
		if f.EncryptedPath != "" {
			entry.EncryptedPath = o.path(f.EncryptedPath)
		}
		findings = append(findings, entry)
	}
	return encodeReportJSON(o, verifyReportJSON{
		OK: report.OK(),
		Summary: verifySummaryJSON{
			Pass:    report.PassCount,
			Warning: report.WarningCount,
			Error:   report.ErrorCount,
			Skipped: report.SkipCount,
		},
		Skipped:  append([]string{}, report.SkipReasons...),
		Findings: findings,
	})
}

type verifyReportJSON struct {
	OK       bool                `json:"ok"`
	Summary  verifySummaryJSON   `json:"summary"`
	Skipped  []string            `json:"skipped"`
	Findings []verifyFindingJSON `json:"findings"`
}

type verifySummaryJSON struct {
	Pass    int `json:"pass"`
	Warning int `json:"warning"`
	Error   int `json:"error"`
	Skipped int `json:"skipped"`
}

type verifyFindingJSON struct {
	Code          string `json:"code"`
	Severity      string `json:"severity"`
	PlaintextPath string `json:"plaintext_path,omitempty"`
	EncryptedPath string `json:"encrypted_path,omitempty"`
	Recipient     string `json:"recipient,omitempty"`
	Message       string `json:"message"`
	Hint          string `json:"hint,omitempty"`
}
