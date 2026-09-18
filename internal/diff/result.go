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
	Diff          string
	Different     bool
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
	s.Results = append(s.Results, Outcome{
		PlaintextFile: plaintext,
		EncryptedFile: encrypted,
		Skipped:       result.Skipped,
		Diff:          result.Diff,
		Different:     result.Different,
		Error:         err,
	})
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

func (s *Summary) Check() error {
	if s.FailedCount > 0 {
		return fmt.Errorf("%d of %d files failed to compare", s.FailedCount, len(s.Results))
	}
	return nil
}
