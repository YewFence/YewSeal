package task

import (
	"fmt"
	"os"
	"path/filepath"
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

	if err := ensureOutputDirs(filePairs, func(pair FilePair) string { return pair.EncryptedPath }); err != nil {
		return nil, err
	}

	processor := func(pair FilePair) error {
		return seal.Encrypt(seal.EncryptOptions{
			InputFile:      pair.PlaintextPath,
			OutputFile:     pair.EncryptedPath,
			Recipients:     pair.Recipients,
			FormatOverride: pair.Format,
		})
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

	processor := func(pair FilePair) error {
		return seal.Decrypt(seal.DecryptOptions{
			InputFile:      pair.EncryptedPath,
			OutputFile:     pair.PlaintextPath,
			IdentityBundle: opts.IdentityBundle,
			FormatOverride: pair.Format,
			Force:          opts.Force,
		})
	}
	describe := func(pair FilePair) (string, string) {
		return pair.EncryptedPath, pair.PlaintextPath
	}

	summary := process(filePairs, opts.Parallel, describe, processor, opts.OnComplete)

	return summary, summary.Check("decrypt", opts.Strict)
}

func ensureOutputDirs(pairs []FilePair, target func(FilePair) string) error {
	seen := map[string]bool{}
	for _, pair := range pairs {
		dir := filepath.Dir(target(pair))
		if dir == "." || seen[dir] {
			continue
		}
		seen[dir] = true
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory %s: %w", dir, err)
		}
	}
	return nil
}

func process(
	pairs []FilePair,
	parallel int,
	describe func(FilePair) (string, string),
	processor func(FilePair) error,
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
	processor func(FilePair) error,
	completed func(Result),
) *Summary {
	summary := &Summary{
		Results: make([]Result, 0, len(pairs)),
	}

	for _, pair := range pairs {
		source, target := describe(pair)

		err := processor(pair)
		result := summary.Add(source, target, err)
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
	processor func(FilePair) error,
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
				err := processor(pair)

				summary.Results[idx] = newResult(source, target, err)

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
