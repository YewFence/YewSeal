package diff

import (
	"fmt"
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
