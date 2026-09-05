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
	Verbose        bool
	Force          bool
	FilePairs      []FilePair
}

type Result struct {
	SourceFile string
	TargetFile string
	Success    bool
	Error      error
}

type Summary struct {
	TotalFiles   int
	SuccessCount int
	FailedCount  int
	Results      []Result
}

func Encrypt(opts Options) (*Summary, error) {
	filePairs := opts.FilePairs
	if len(filePairs) == 0 {
		return nil, fmt.Errorf("no configured file pairs to encrypt")
	}
	fmt.Printf("Encrypting %d files from config...\n", len(filePairs))

	if err := ensureOutputDirs(filePairs, func(pair FilePair) string { return pair.EncryptedPath }); err != nil {
		return nil, err
	}

	processor := func(pair FilePair) error {
		return seal.Encrypt(seal.EncryptOptions{
			InputFile:      pair.PlaintextPath,
			OutputFile:     pair.EncryptedPath,
			Recipients:     pair.Recipients,
			FormatOverride: pair.Format,
			Verbose:        opts.Verbose,
		})
	}
	describe := func(pair FilePair) (string, string) {
		return pair.PlaintextPath, pair.EncryptedPath
	}

	summary := process(filePairs, opts.Parallel, describe, processor)
	printSummary(summary, "encrypted")

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
	fmt.Printf("Decrypting %d files from config...\n", len(filePairs))

	if err := ensureOutputDirs(filePairs, func(pair FilePair) string { return pair.PlaintextPath }); err != nil {
		return nil, err
	}

	processor := func(pair FilePair) error {
		return seal.Decrypt(seal.DecryptOptions{
			InputFile:      pair.EncryptedPath,
			OutputFile:     pair.PlaintextPath,
			IdentityBundle: opts.IdentityBundle,
			FormatOverride: pair.Format,
			Verbose:        opts.Verbose,
			Force:          opts.Force,
		})
	}
	describe := func(pair FilePair) (string, string) {
		return pair.EncryptedPath, pair.PlaintextPath
	}

	summary := process(filePairs, opts.Parallel, describe, processor)
	printSummary(summary, "decrypted")

	if summary.FailedCount > 0 {
		return summary, fmt.Errorf("%d of %d files failed to decrypt", summary.FailedCount, summary.TotalFiles)
	}
	return summary, nil
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
) *Summary {
	if parallel > 1 {
		return processParallel(pairs, parallel, describe, processor)
	}
	return processSequential(pairs, describe, processor)
}

func processSequential(
	pairs []FilePair,
	describe func(FilePair) (string, string),
	processor func(FilePair) error,
) *Summary {
	summary := &Summary{
		TotalFiles: len(pairs),
		Results:    make([]Result, 0, len(pairs)),
	}

	for i, pair := range pairs {
		source, target := describe(pair)
		fmt.Printf("  [%d/%d] %s -> %s ... ", i+1, summary.TotalFiles, filepath.Base(source), filepath.Base(target))

		err := processor(pair)
		result := Result{
			SourceFile: source,
			TargetFile: target,
			Success:    err == nil,
			Error:      err,
		}
		summary.Results = append(summary.Results, result)

		if err == nil {
			summary.SuccessCount++
			fmt.Println("OK")
		} else {
			summary.FailedCount++
			fmt.Printf("FAILED\n        Error: %v\n", err)
		}
	}

	return summary
}

func processParallel(
	pairs []FilePair,
	parallel int,
	describe func(FilePair) (string, string),
	processor func(FilePair) error,
) *Summary {
	summary := &Summary{
		TotalFiles: len(pairs),
		Results:    make([]Result, len(pairs)),
	}

	var wg sync.WaitGroup
	jobs := make(chan int, len(pairs))
	var mu sync.Mutex
	var completed int

	for range parallel {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobs {
				pair := pairs[idx]
				source, target := describe(pair)
				err := processor(pair)

				summary.Results[idx] = Result{
					SourceFile: source,
					TargetFile: target,
					Success:    err == nil,
					Error:      err,
				}

				mu.Lock()
				completed++
				current := completed
				mu.Unlock()

				if err == nil {
					fmt.Printf("  [%d/%d] %s -> %s ... OK\n", current, summary.TotalFiles, filepath.Base(source), filepath.Base(target))
				} else {
					fmt.Printf("  [%d/%d] %s -> %s ... FAILED\n        Error: %v\n", current, summary.TotalFiles, filepath.Base(source), filepath.Base(target), err)
				}
			}
		}()
	}

	for i := range pairs {
		jobs <- i
	}
	close(jobs)

	wg.Wait()

	for _, result := range summary.Results {
		if result.Success {
			summary.SuccessCount++
		} else {
			summary.FailedCount++
		}
	}

	return summary
}

func printSummary(summary *Summary, action string) {
	fmt.Println()
	if summary.FailedCount == 0 {
		fmt.Printf("✅ Summary: %d/%d files %s successfully\n", summary.SuccessCount, summary.TotalFiles, action)
	} else {
		fmt.Printf("⚠️  Summary: %d/%d succeeded, %d failed\n", summary.SuccessCount, summary.TotalFiles, summary.FailedCount)
		fmt.Println("\nFailed files:")
		for _, result := range summary.Results {
			if !result.Success {
				fmt.Printf("  - %s: %v\n", filepath.Base(result.SourceFile), result.Error)
			}
		}
	}
}
