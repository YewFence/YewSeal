package presentation

import (
	"fmt"
	"strings"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/verify"
)

// VerifyPrintOptions controls how the verify report is rendered.
type VerifyPrintOptions struct {
	JSON    bool
	Verbose bool
	CWD     string
}

// VerifyReport renders the verify report to the appropriate output stream.
func (o *Output) VerifyReport(report *verify.Report, opts VerifyPrintOptions) error {
	if report == nil {
		return nil
	}
	if opts.JSON {
		return o.verifyReportJSON(report, opts.CWD)
	}
	return o.verifyReportText(report, opts)
}

func (o *Output) verifyReportText(report *verify.Report, opts VerifyPrintOptions) error {
	// Summary line first.
	summary := fmt.Sprintf("Summary: %d passed, %d warnings, %d errors, %d skipped",
		report.PassCount, report.WarningCount, report.ErrorCount, report.SkipCount)
	if _, err := fmt.Fprintln(o, summary); err != nil {
		return err
	}

	// Skip reasons (always shown so CI knows what was not checked).
	for _, reason := range report.SkipReasons {
		if _, err := fmt.Fprintf(o, "SKIPPED: %s\n", reason); err != nil {
			return err
		}
	}

	if len(report.Findings) == 0 {
		return nil
	}

	if _, err := fmt.Fprintln(o, ""); err != nil {
		return err
	}

	for _, f := range report.Findings {
		label := strings.ToUpper(string(f.Severity))
		path := displayFindingPath(f, opts.CWD)
		line := fmt.Sprintf("%s [%s]%s: %s", label, f.Code, path, f.Message)
		if _, err := fmt.Fprintln(o, line); err != nil {
			return err
		}
		if f.Hint != "" {
			if _, err := fmt.Fprintf(o, "  Hint: %s\n", f.Hint); err != nil {
				return err
			}
		}
	}
	return nil
}

func displayFindingPath(f verify.Finding, cwd string) string {
	path := f.PlaintextPath
	if path == "" {
		path = f.EncryptedPath
	}
	if path == "" {
		return ""
	}
	if cwd != "" {
		path = config.DisplayPath(cwd, path)
	}
	return " " + path
}

// verifyReportJSON renders the report as JSON on stdout; errors go to stderr.
func (o *Output) verifyReportJSON(report *verify.Report, cwd string) error {
	findings := make([]verifyFindingJSON, 0, len(report.Findings))
	for _, f := range report.Findings {
		plain := f.PlaintextPath
		enc := f.EncryptedPath
		if cwd != "" {
			if plain != "" {
				plain = config.DisplayPath(cwd, plain)
			}
			if enc != "" {
				enc = config.DisplayPath(cwd, enc)
			}
		}
		findings = append(findings, verifyFindingJSON{
			Code:          f.Code,
			Severity:      string(f.Severity),
			PlaintextPath: plain,
			EncryptedPath: enc,
			Message:       f.Message,
			Hint:          f.Hint,
		})
	}
	return encodeReportJSON(o, verifyReportJSON{
		OK: report.OK(),
		Summary: verifySummaryJSON{
			Pass:    report.PassCount,
			Warning: report.WarningCount,
			Error:   report.ErrorCount,
			Skipped: report.SkipCount,
		},
		Findings: findings,
	})
}

type verifyReportJSON struct {
	OK       bool                `json:"ok"`
	Summary  verifySummaryJSON   `json:"summary"`
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
	Message       string `json:"message"`
	Hint          string `json:"hint,omitempty"`
}
