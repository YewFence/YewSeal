package task

import (
	"errors"
	"fmt"

	"github.com/YewFence/YewSeal/internal/seal"
	"github.com/YewFence/YewSeal/internal/sopsx"
)

type Status string
type Outcome string

const (
	Succeeded Status = "succeeded"
	Skipped   Status = "skipped"
	Failed    Status = "failed"

	OutcomeProcessed          Outcome = "processed"
	OutcomeEncrypted          Outcome = "encrypted"
	OutcomeUnchanged          Outcome = "unchanged"
	OutcomeMissingPlaintext   Outcome = "missing-plaintext"
	OutcomeNoIdentity         Outcome = "no-identity"
	OutcomeNoMatchingIdentity Outcome = "no-matching-identity"
)

type Result struct {
	SourceFile string
	TargetFile string
	Status     Status
	Outcome    Outcome
	Warning    string
	Error      error
}

type Summary struct {
	TotalFiles            int
	SuccessCount          int
	SkippedCount          int
	FailedCount           int
	EncryptedCount        int
	UnchangedCount        int
	MissingPlaintextCount int
	Results               []Result
}

func newResult(source, target string, err error) Result {
	return newOutcomeResult(source, target, OutcomeProcessed, "", err)
}

func newOutcomeResult(source, target string, outcome Outcome, warning string, err error) Result {
	result := Result{SourceFile: source, TargetFile: target, Status: Succeeded, Outcome: outcome, Warning: warning, Error: err}
	if errors.Is(err, seal.ErrNoIdentity) {
		result.Status = Skipped
		result.Outcome = OutcomeNoIdentity
	} else if errors.Is(err, sopsx.ErrNoMatchingIdentity) {
		result.Status = Skipped
		result.Outcome = OutcomeNoMatchingIdentity
	} else if err != nil {
		result.Status = Failed
	} else if outcome == OutcomeMissingPlaintext {
		result.Status = Skipped
	}
	return result
}

func (s *Summary) Add(source, target string, err error) Result {
	result := newResult(source, target, err)
	s.Results = append(s.Results, result)
	s.TotalFiles++
	s.count(result)
	return result
}

func (s *Summary) count(result Result) {
	switch result.Status {
	case Succeeded:
		s.SuccessCount++
	case Skipped:
		s.SkippedCount++
	case Failed:
		s.FailedCount++
	}
	switch result.Outcome {
	case OutcomeEncrypted:
		s.EncryptedCount++
	case OutcomeUnchanged:
		s.UnchangedCount++
	case OutcomeMissingPlaintext:
		s.MissingPlaintextCount++
	}
}

// Check separates per-file outcomes from the caller's completeness requirement.
// Lenient mode tolerates any number of skips, including a fully skipped batch.
func (s *Summary) Check(action string, strict bool) error {
	if s.FailedCount > 0 {
		return fmt.Errorf("%d of %d files failed to %s (%d skipped)", s.FailedCount, s.TotalFiles, action, s.SkippedCount)
	}
	if strict && s.SkippedCount > 0 {
		return fmt.Errorf("strict mode requires complete processing: %d of %d files skipped", s.SkippedCount, s.TotalFiles)
	}
	return nil
}
