package clean

import "fmt"

type Status string

const (
	StatusRemoved       Status = "removed"
	StatusAlreadyAbsent Status = "already-absent"
	StatusRetained      Status = "retained"
	StatusFailed        Status = "failed"
)

type Result struct {
	PlaintextPath string
	Status        Status
	Error         error
}

type Summary struct {
	RemovedCount       int
	AlreadyAbsentCount int
	RetainedCount      int
	FailedCount        int
	Results            []Result
}

func (s *Summary) Add(result Result) {
	s.Results = append(s.Results, result)
	switch result.Status {
	case StatusRemoved:
		s.RemovedCount++
	case StatusAlreadyAbsent:
		s.AlreadyAbsentCount++
	case StatusRetained:
		s.RetainedCount++
	case StatusFailed:
		s.FailedCount++
	}
}

func (s Summary) Check() error {
	if s.FailedCount == 0 {
		return nil
	}
	return fmt.Errorf("%d of %d plaintext files failed to clean", s.FailedCount, len(s.Results))
}
