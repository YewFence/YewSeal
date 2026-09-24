package verify

import (
	"fmt"
	"os"
	"sort"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/sopsx"
)

// checkCiphertext runs the static ciphertext checks, which need no private
// key, and reports whether the ciphertext is usable for the decrypt layer.
func checkCiphertext(report *Report, pair config.ResolvedFilePair, labels recipientLabels) bool {
	fail := func(code, message, hint string) bool {
		report.Add(Finding{
			Code:          code,
			Severity:      SeverityError,
			PlaintextPath: pair.PlaintextPath,
			EncryptedPath: pair.EncryptedPath,
			Message:       message,
			Hint:          hint,
		})
		return false
	}

	info, err := os.Lstat(pair.EncryptedPath)
	switch {
	case os.IsNotExist(err):
		return fail("ciphertext_missing", "encrypted file does not exist", "run 'yews encrypt' to create it")
	case err != nil:
		return fail("ciphertext_stat_error", fmt.Sprintf("failed to inspect encrypted file: %v", err), "")
	case !info.Mode().IsRegular():
		return fail("ciphertext_not_regular", "encrypted path is not a regular file", "replace it with the encrypted file itself")
	}
	encData, err := os.ReadFile(pair.EncryptedPath)
	if err != nil {
		return fail("ciphertext_read_error", fmt.Sprintf("failed to read encrypted file: %v", err), "")
	}
	inspected, err := sopsx.Inspect(encData, pair.Format)
	if err != nil {
		return fail("ciphertext_parse_error", fmt.Sprintf("encrypted file is not valid %s SOPS ciphertext: %v", pair.Format, err), "restore it from version control or re-encrypt it with 'yews encrypt --force'")
	}
	if len(inspected.AgeRecipients) == 0 {
		return fail("ciphertext_no_recipients", "SOPS metadata contains no Age recipient", "re-encrypt it with 'yews encrypt --force'")
	}
	metadataValid := true
	keyGroupsSupported := inspected.KeyGroupCount == 1
	if !keyGroupsSupported {
		fail("key_groups_unsupported", "SOPS metadata contains unsupported multiple key groups", "re-encrypt it with 'yews encrypt --force' to normalize the key group layout")
		metadataValid = false
	}
	if inspected.HasNonAgeKeys {
		fail("recipient_unsupported", "SOPS metadata contains non-Age decryption keys", "re-encrypt it with 'yews encrypt --force' to remove unauthorized keys")
		metadataValid = false
	}
	recipientsValid := checkRecipients(report, pair, inspected.AgeRecipients, labels)
	if metadataValid && recipientsValid {
		report.AddPass()
	}
	return keyGroupsSupported
}

// checkRecipients compares the metadata recipient list with the configured
// canonical set. Missing, extra, and duplicate recipients are reported
// separately because each needs a different repair.
func checkRecipients(report *Report, pair config.ResolvedFilePair, actual []string, labels recipientLabels) bool {
	configured := make(map[string]bool, len(pair.Recipients))
	for _, recipient := range pair.Recipients {
		configured[recipient] = true
	}
	counts := make(map[string]int, len(actual))
	for _, recipient := range actual {
		counts[recipient]++
	}

	var findings []Finding
	add := func(code, recipient, message string) {
		findings = append(findings, Finding{
			Code:          code,
			Severity:      SeverityError,
			PlaintextPath: pair.PlaintextPath,
			EncryptedPath: pair.EncryptedPath,
			Recipient:     recipient,
			Message:       message,
			Hint:          "run 'yews encrypt' to rewrap the data key for the configured recipients",
		})
	}
	for _, recipient := range sortedKeys(configured) {
		if counts[recipient] == 0 {
			add("recipient_missing", recipient, fmt.Sprintf("configured recipient %s cannot decrypt this file", labels.label(recipient)))
		}
	}
	for _, recipient := range sortedKeys(counts) {
		if !configured[recipient] {
			add("recipient_extra", recipient, fmt.Sprintf("recipient %s can decrypt this file but is not configured", labels.label(recipient)))
		}
		if counts[recipient] > 1 {
			add("recipient_duplicate", recipient, fmt.Sprintf("recipient %s appears %d times in SOPS metadata", labels.label(recipient), counts[recipient]))
		}
	}
	if len(findings) == 0 {
		return true
	}
	for _, finding := range findings {
		report.Add(finding)
	}
	return false
}

// recipientLabels maps canonical public keys to registry aliases for display.
type recipientLabels map[string]string

// label shows the alias when known, otherwise a short public-key fingerprint;
// full keys stay in the Recipient field for machine consumers.
func (l recipientLabels) label(recipient string) string {
	if alias, ok := l[recipient]; ok {
		return alias
	}
	const edge = 8
	if len(recipient) <= 2*edge {
		return recipient
	}
	return recipient[:edge] + "…" + recipient[len(recipient)-edge:]
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
