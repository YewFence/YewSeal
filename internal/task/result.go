package task

import (
	"errors"
	"fmt"

	"github.com/YewFence/YewSeal/internal/sopsx"
)

type Status string

const (
	Succeeded Status = "succeeded"
	Skipped   Status = "skipped"
	Failed    Status = "failed"
)

type Result struct {
	SourceFile string
	TargetFile string
	Status     Status
	Error      error
}

type Summary struct {
	TotalFiles   int
	SuccessCount int
	SkippedCount int
	FailedCount  int
	Results      []Result
}

func newResult(source, target string, err error) Result {
	result := Result{SourceFile: source, TargetFile: target, Status: Succeeded, Error: err}
	if errors.Is(err, sopsx.ErrNoMatchingIdentity) {
		result.Status = Skipped
	} else if err != nil {
		result.Status = Failed
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
}

// Check separates per-file outcomes from the caller's completeness requirement.
func (s *Summary) Check(action string, strict bool) error {
	if s.FailedCount > 0 {
		return fmt.Errorf("%d of %d files failed to %s (%d skipped)", s.FailedCount, s.TotalFiles, action, s.SkippedCount)
	}
	if s.SuccessCount == 0 {
		return fmt.Errorf("no files successfully processed by %s (%d skipped)", action, s.SkippedCount)
	}
	if strict && s.SkippedCount > 0 {
		return fmt.Errorf("strict mode requires complete processing: %d of %d files skipped", s.SkippedCount, s.TotalFiles)
	}
	return nil
}
