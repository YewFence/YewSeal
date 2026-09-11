package diff

import (
	"fmt"
	"io"

	"github.com/YewFence/YewSeal/internal/sopsx"
)

type SkipReason string

const (
	MissingPlaintext   SkipReason = "missing-plaintext"
	MissingCiphertext  SkipReason = "missing-ciphertext"
	MissingBothInputs  SkipReason = "missing-both-inputs"
	NoMatchingIdentity SkipReason = "no-matching-identity"
)

type Outcome struct {
	PlaintextFile string
	EncryptedFile string
	Skipped       SkipReason
	Error         error
}

type Summary struct {
	ComparedCount     int
	MissingInputCount int
	NoIdentityCount   int
	FailedCount       int
	Results           []Outcome
}

func (s *Summary) Add(plaintext, encrypted string, result DiffResult, err error) {
	s.Results = append(s.Results, Outcome{PlaintextFile: plaintext, EncryptedFile: encrypted, Skipped: result.Skipped, Error: err})
	switch {
	case err != nil:
		s.FailedCount++
	case result.Skipped == NoMatchingIdentity:
		s.NoIdentityCount++
	case result.Skipped != "":
		s.MissingInputCount++
	default:
		s.ComparedCount++
	}
}

func (s *Summary) Check(strict bool) error {
	if s.FailedCount > 0 {
		return fmt.Errorf("%d of %d files failed to compare", s.FailedCount, len(s.Results))
	}
	if strict && s.NoIdentityCount > 0 {
		return fmt.Errorf("strict mode requires comparison of files with both inputs present: %d skipped without matching identity", s.NoIdentityCount)
	}
	return nil
}

func (s *Summary) Report(w io.Writer) error {
	for _, result := range s.Results {
		if result.Error != nil {
			if _, err := fmt.Fprintf(w, "FAILED %s -> %s: %v\n", result.PlaintextFile, result.EncryptedFile, result.Error); err != nil {
				return err
			}
			continue
		}
		var reason string
		switch result.Skipped {
		case MissingPlaintext:
			reason = "plaintext file is missing; not compared"
		case MissingCiphertext:
			reason = "encrypted file is missing; not compared"
		case MissingBothInputs:
			reason = "plaintext and encrypted files are missing; not compared"
		case NoMatchingIdentity:
			reason = sopsx.ErrNoMatchingIdentity.Error()
		default:
			continue
		}
		if _, err := fmt.Fprintf(w, "SKIPPED %s -> %s: %s\n", result.PlaintextFile, result.EncryptedFile, reason); err != nil {
			return err
		}
	}
	if s.NoIdentityCount > 0 || s.FailedCount > 0 {
		if _, err := fmt.Fprintln(w, "Comparison incomplete: some files with both inputs present could not be compared."); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(w, "Summary: %d compared, %d skipped (%d missing input, %d no matching identity), %d failed (%d selected)\n",
		s.ComparedCount, s.MissingInputCount+s.NoIdentityCount, s.MissingInputCount, s.NoIdentityCount, s.FailedCount, len(s.Results))
	return err
}
