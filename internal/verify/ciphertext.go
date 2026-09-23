package verify

import (
	"fmt"
	"os"
	"sort"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/sopsx"
)

// checkCiphertext performs the static ciphertext layer: file existence,
// regular-file check, SOPS parseability, and recipient-set comparison.
// No private key is required.
func checkCiphertext(report *Report, pair config.ResolvedFilePair) {
	encPath := pair.EncryptedPath

	info, err := os.Lstat(encPath)
	if os.IsNotExist(err) {
		report.Add(Finding{
			Code:          "ciphertext_missing",
			Severity:      SeverityError,
			PlaintextPath: pair.PlaintextPath,
			EncryptedPath: encPath,
			Message:       fmt.Sprintf("encrypted file %s does not exist", encPath),
			Hint:          "run 'yews encrypt' to create it",
		})
		return
	}
	if err != nil {
		report.Add(Finding{
			Code:          "ciphertext_stat_error",
			Severity:      SeverityError,
			PlaintextPath: pair.PlaintextPath,
			EncryptedPath: encPath,
			Message:       fmt.Sprintf("failed to inspect encrypted file %s: %v", encPath, err),
		})
		return
	}
	if !info.Mode().IsRegular() {
		report.Add(Finding{
			Code:          "ciphertext_not_regular",
			Severity:      SeverityError,
			PlaintextPath: pair.PlaintextPath,
			EncryptedPath: encPath,
			Message:       fmt.Sprintf("encrypted path %s is not a regular file", encPath),
		})
		return
	}

	encData, err := os.ReadFile(encPath)
	if err != nil {
		report.Add(Finding{
			Code:          "ciphertext_read_error",
			Severity:      SeverityError,
			PlaintextPath: pair.PlaintextPath,
			EncryptedPath: encPath,
			Message:       fmt.Sprintf("failed to read encrypted file %s: %v", encPath, err),
		})
		return
	}

	inspected, err := sopsx.Inspect(encData, pair.Format)
	if err != nil {
		report.Add(Finding{
			Code:          "ciphertext_parse_error",
			Severity:      SeverityError,
			PlaintextPath: pair.PlaintextPath,
			EncryptedPath: encPath,
			Message:       fmt.Sprintf("encrypted file %s cannot be parsed by SOPS: %v", encPath, err),
			Hint:          "the file may be corrupted or not a SOPS-encrypted file",
		})
		return
	}
	if len(inspected.AgeRecipients) == 0 {
		report.Add(Finding{
			Code:          "ciphertext_no_recipients",
			Severity:      SeverityError,
			PlaintextPath: pair.PlaintextPath,
			EncryptedPath: encPath,
			Message:       fmt.Sprintf("encrypted file %s contains no Age recipients in its SOPS metadata", encPath),
			Hint:          "re-encrypt the file with 'yews encrypt'",
		})
		return
	}

	checkRecipientDrift(report, pair, inspected.AgeRecipients)
}

func checkRecipientDrift(report *Report, pair config.ResolvedFilePair, actual []string) {
	if len(pair.Recipients) == 0 {
		report.AddPass()
		return
	}

	configured := canonicalSet(pair.Recipients)
	present := canonicalSet(actual)

	var missing, extra []string
	for r := range configured {
		if !present[r] {
			missing = append(missing, r)
		}
	}
	for r := range present {
		if !configured[r] {
			extra = append(extra, r)
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)

	if len(missing) == 0 && len(extra) == 0 {
		report.AddPass()
		return
	}
	for _, r := range missing {
		report.Add(Finding{
			Code:          "recipient_missing",
			Severity:      SeverityError,
			PlaintextPath: pair.PlaintextPath,
			EncryptedPath: pair.EncryptedPath,
			Message:       fmt.Sprintf("recipient %s is declared in config but absent from ciphertext metadata", r),
			Hint:          "run 'yews encrypt' to rewrap the data key for all configured recipients",
		})
	}
	for _, r := range extra {
		report.Add(Finding{
			Code:          "recipient_extra",
			Severity:      SeverityError,
			PlaintextPath: pair.PlaintextPath,
			EncryptedPath: pair.EncryptedPath,
			Message:       fmt.Sprintf("recipient %s is present in ciphertext metadata but not declared in config", r),
			Hint:          "run 'yews encrypt' to rewrap with only the configured recipients",
		})
	}
}

func canonicalSet(recipients []string) map[string]bool {
	m := make(map[string]bool, len(recipients))
	for _, r := range recipients {
		m[r] = true
	}
	return m
}
