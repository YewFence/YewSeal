// Package verify implements the four-layer health check for a YewSeal project.
// It is read-only: it never modifies project files, ciphertext, or plaintext.
package verify

import "sort"

// Severity classifies a Finding's urgency.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

// Finding describes one issue discovered during verification.
// Pass findings are not represented; only errors and warnings carry a Finding.
type Finding struct {
	Code          string   `json:"code"`
	Severity      Severity `json:"severity"`
	PlaintextPath string   `json:"plaintext_path,omitempty"`
	EncryptedPath string   `json:"encrypted_path,omitempty"`
	Recipient     string   `json:"recipient,omitempty"`
	Message       string   `json:"message"`
	Hint          string   `json:"hint,omitempty"`
}

// Report is the full result of a verify run.
type Report struct {
	Findings    []Finding
	SkipReasons []string

	PassCount    int
	WarningCount int
	ErrorCount   int
	SkipCount    int
}

// OK returns true when no error-severity findings were produced.
func (r *Report) OK() bool { return r.ErrorCount == 0 }

func (r *Report) Add(f Finding) {
	r.Findings = append(r.Findings, f)
	if f.Severity == SeverityError {
		r.ErrorCount++
	} else {
		r.WarningCount++
	}
}

func (r *Report) AddPass() { r.PassCount++ }

// AddSkip counts one skipped check. Identical reasons are recorded once so the
// report explains each skipped layer without repeating it per file.
func (r *Report) AddSkip(reason string) {
	r.SkipCount++
	for _, existing := range r.SkipReasons {
		if existing == reason {
			return
		}
	}
	r.SkipReasons = append(r.SkipReasons, reason)
}

// Sort orders findings deterministically by (plaintext_path, encrypted_path, code).
func (r *Report) Sort() {
	sort.Slice(r.Findings, func(i, j int) bool {
		a, b := r.Findings[i], r.Findings[j]
		if a.PlaintextPath != b.PlaintextPath {
			return a.PlaintextPath < b.PlaintextPath
		}
		if a.EncryptedPath != b.EncryptedPath {
			return a.EncryptedPath < b.EncryptedPath
		}
		return a.Code < b.Code
	})
}
