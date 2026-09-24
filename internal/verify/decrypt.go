package verify

import (
	"errors"
	"fmt"
	"os"

	"github.com/YewFence/YewSeal/internal/agekey"
	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/seal"
	"github.com/YewFence/YewSeal/internal/sopsx"
)

// DecryptBehavior controls whether the decrypt layer runs.
type DecryptBehavior int

const (
	DecryptAuto     DecryptBehavior = iota // attempt if identity available; skip gracefully otherwise
	DecryptRequired                        // --decrypt: missing or non-matching identity = error
	DecryptDisabled                        // --no-decrypt: skip the entire layer
)

func checkDecrypt(report *Report, pair config.ResolvedFilePair, bundle agekey.IdentityBundle, mode DecryptBehavior) {
	plainData, decryptErr := seal.DecryptToBytes(seal.DecryptBytesOptions{
		InputFile:      pair.EncryptedPath,
		OutputFile:     pair.PlaintextPath,
		IdentityBundle: bundle,
		FormatOverride: pair.Format,
	})

	if errors.Is(decryptErr, sopsx.ErrMACMismatch) {
		report.Add(Finding{
			Code:          "mac_mismatch",
			Severity:      SeverityError,
			PlaintextPath: pair.PlaintextPath,
			EncryptedPath: pair.EncryptedPath,
			Message:       fmt.Sprintf("MAC verification failed for %s: file may have been tampered with", pair.EncryptedPath),
			Hint:          "the ciphertext may be corrupted or modified; re-encrypt from a trusted plaintext source",
		})
		return
	}
	if errors.Is(decryptErr, sopsx.ErrNoMatchingIdentity) {
		if mode == DecryptRequired {
			report.Add(Finding{
				Code:          "decrypt_no_matching_identity",
				Severity:      SeverityError,
				PlaintextPath: pair.PlaintextPath,
				EncryptedPath: pair.EncryptedPath,
				Message:       fmt.Sprintf("no available identity can decrypt %s", pair.EncryptedPath),
				Hint:          "provide the correct Age private key with --key-file or SOPS_AGE_KEY",
			})
		} else {
			report.Add(Finding{
				Code:          "decrypt_no_matching_identity",
				Severity:      SeverityWarning,
				PlaintextPath: pair.PlaintextPath,
				EncryptedPath: pair.EncryptedPath,
				Message:       fmt.Sprintf("no available identity can decrypt %s; plaintext consistency not checked", pair.EncryptedPath),
				Hint:          "provide the correct Age private key to enable content verification",
			})
		}
		return
	}
	if decryptErr != nil {
		report.Add(Finding{
			Code:          "decrypt_failed",
			Severity:      SeverityError,
			PlaintextPath: pair.PlaintextPath,
			EncryptedPath: pair.EncryptedPath,
			Message:       fmt.Sprintf("failed to decrypt %s: %v", pair.EncryptedPath, decryptErr),
		})
		return
	}

	report.AddPass() // decrypt + MAC OK

	currentData, err := os.ReadFile(pair.PlaintextPath)
	if os.IsNotExist(err) {
		report.AddSkip("local plaintext absent; plaintext consistency not checked for those files")
		return
	}
	if err != nil {
		report.Add(Finding{
			Code:          "plaintext_read_error",
			Severity:      SeverityError,
			PlaintextPath: pair.PlaintextPath,
			EncryptedPath: pair.EncryptedPath,
			Message:       fmt.Sprintf("failed to read plaintext %s: %v", pair.PlaintextPath, err),
		})
		return
	}
	equal, err := sopsx.PlaintextEqual(plainData, currentData, pair.Format)
	if err != nil {
		report.Add(Finding{
			Code:          "plaintext_parse_error",
			Severity:      SeverityError,
			PlaintextPath: pair.PlaintextPath,
			EncryptedPath: pair.EncryptedPath,
			Message:       fmt.Sprintf("failed to compare plaintext %s: %v", pair.PlaintextPath, err),
		})
		return
	}
	if equal {
		report.AddPass()
	} else {
		report.Add(Finding{
			Code:          "plaintext_drift",
			Severity:      SeverityError,
			PlaintextPath: pair.PlaintextPath,
			EncryptedPath: pair.EncryptedPath,
			Message:       fmt.Sprintf("plaintext %s differs from decrypted content of %s", pair.PlaintextPath, pair.EncryptedPath),
			Hint:          "run 'yews encrypt' to sync the ciphertext, or 'yews decrypt --force' to restore from ciphertext",
		})
	}
}
