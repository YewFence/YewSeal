package task

import (
	"fmt"
	"sync"

	"github.com/YewFence/YewSeal/internal/agekey"
	"github.com/YewFence/YewSeal/internal/seal"
)

type FilePair struct {
	PlaintextPath string
	EncryptedPath string
	Format        string
	Recipients    []string
}

type Options struct {
	IdentityBundle agekey.IdentityBundle
	Parallel       int
	OnComplete     func(Result)
	Force          bool
	Strict         bool
	FilePairs      []FilePair
}

func Encrypt(opts Options) (*Summary, error) {
	filePairs := opts.FilePairs
	if len(filePairs) == 0 {
		return nil, fmt.Errorf("no configured file pairs to encrypt")
	}

	processor := func(pair FilePair) (Outcome, string, error) {
		result, err := seal.Update(seal.UpdateOptions{
			EncryptOptions: seal.EncryptOptions{
				InputFile:      pair.PlaintextPath,
				OutputFile:     pair.EncryptedPath,
				Recipients:     pair.Recipients,
				FormatOverride: pair.Format,
			},
			IdentityBundle: opts.IdentityBundle,
			Force:          opts.Force,
		})
		if err != nil {
			return OutcomeProcessed, result.Warning, err
		}
		outcome := OutcomeEncrypted
		switch result.Outcome {
		case seal.Unchanged:
			outcome = OutcomeUnchanged
		case seal.MissingPlaintext:
			outcome = OutcomeMissingPlaintext
		}
		return outcome, result.Warning, err
	}
	describe := func(pair FilePair) (string, string) {
		return pair.PlaintextPath, pair.EncryptedPath
	}

	summary := process(filePairs, opts.Parallel, describe, processor, opts.OnComplete)

	if summary.FailedCount > 0 {
		return summary, fmt.Errorf("%d of %d files failed to encrypt", summary.FailedCount, summary.TotalFiles)
	}
	return summary, nil
}

func Decrypt(opts Options) (*Summary, error) {
	filePairs := opts.FilePairs
	if len(filePairs) == 0 {
		return nil, fmt.Errorf("no configured file pairs to decrypt")
	}

	processor := func(pair FilePair) (Outcome, string, error) {
		err := seal.Decrypt(seal.DecryptOptions{
			InputFile:      pair.EncryptedPath,
			OutputFile:     pair.PlaintextPath,
			IdentityBundle: opts.IdentityBundle,
			FormatOverride: pair.Format,
			Force:          opts.Force,
		})
		return OutcomeProcessed, "", err
	}
	describe := func(pair FilePair) (string, string) {
		return pair.EncryptedPath, pair.PlaintextPath
	}

	summary := process(filePairs, opts.Parallel, describe, processor, opts.OnComplete)

	return summary, summary.Check("decrypt", opts.Strict)
}

func process(
	pairs []FilePair,
	parallel int,
	describe func(FilePair) (string, string),
	processor func(FilePair) (Outcome, string, error),
	completed func(Result),
) *Summary {
	if parallel > 1 {
		return processParallel(pairs, parallel, describe, processor, completed)
	}
	return processSequential(pairs, describe, processor, completed)
}

func processSequential(
	pairs []FilePair,
	describe func(FilePair) (string, string),
	processor func(FilePair) (Outcome, string, error),
	completed func(Result),
) *Summary {
	summary := &Summary{
		Results: make([]Result, 0, len(pairs)),
	}

	for _, pair := range pairs {
		source, target := describe(pair)

		outcome, warning, err := processor(pair)
		result := newOutcomeResult(source, target, outcome, warning, err)
		summary.Results = append(summary.Results, result)
		summary.TotalFiles++
		summary.count(result)
		if completed != nil {
			completed(result)
		}
	}

	return summary
}

func processParallel(
	pairs []FilePair,
	parallel int,
	describe func(FilePair) (string, string),
	processor func(FilePair) (Outcome, string, error),
	completed func(Result),
) *Summary {
	summary := &Summary{
		TotalFiles: len(pairs),
		Results:    make([]Result, len(pairs)),
	}

	var wg sync.WaitGroup
	jobs := make(chan int, len(pairs))
	var mu sync.Mutex

	for range parallel {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobs {
				pair := pairs[idx]
				source, target := describe(pair)
				outcome, warning, err := processor(pair)

				summary.Results[idx] = newOutcomeResult(source, target, outcome, warning, err)

				mu.Lock()
				if completed != nil {
					completed(summary.Results[idx])
				}
				mu.Unlock()
			}
		}()
	}

	for i := range pairs {
		jobs <- i
	}
	close(jobs)

	wg.Wait()

	for _, result := range summary.Results {
		summary.count(result)
	}

	return summary
}
