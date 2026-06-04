package batch

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/YewFence/YewSeal/internal/seal"
)

type FilePair struct {
	PlaintextPath string
	EncryptedPath string
	Format        string
}

type Options struct {
	InputDir     string
	Pattern      string
	OutputDir    string
	OutputSuffix string
	KeyFile      string
	PublicKey    string
	Parallel     int
	Verbose      bool
	Force        bool
	FilePairs    []FilePair
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
	filePairs, err := encryptFilePairs(opts)
	if err != nil {
		return nil, err
	}

	if len(opts.FilePairs) > 0 {
		fmt.Printf("Encrypting %d files from config...\n", len(filePairs))
	} else {
		fmt.Printf("Encrypting %d files from %s...\n", len(filePairs), opts.InputDir)
	}

	if err := ensureOutputDir(opts.OutputDir); err != nil {
		return nil, err
	}

	processor := func(pair FilePair) error {
		return seal.Encrypt(seal.EncryptOptions{
			InputFile:      pair.PlaintextPath,
			OutputFile:     pair.EncryptedPath,
			KeyFile:        opts.KeyFile,
			PublicKey:      opts.PublicKey,
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
	filePairs, err := decryptFilePairs(opts)
	if err != nil {
		return nil, err
	}

	if len(opts.FilePairs) > 0 {
		fmt.Printf("Decrypting %d files from config...\n", len(filePairs))
	} else {
		fmt.Printf("Decrypting %d files from %s...\n", len(filePairs), opts.InputDir)
	}

	if err := ensureOutputDir(opts.OutputDir); err != nil {
		return nil, err
	}

	processor := func(pair FilePair) error {
		return seal.Decrypt(seal.DecryptOptions{
			InputFile:      pair.EncryptedPath,
			OutputFile:     pair.PlaintextPath,
			KeyFile:        opts.KeyFile,
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

func encryptFilePairs(opts Options) ([]FilePair, error) {
	if len(opts.FilePairs) > 0 {
		return opts.FilePairs, nil
	}

	files, err := collectFiles(opts.InputDir, opts.Pattern)
	if err != nil {
		return nil, err
	}

	filePairs := make([]FilePair, 0, len(files))
	for _, file := range files {
		filePairs = append(filePairs, FilePair{
			PlaintextPath: file,
			EncryptedPath: GenerateOutputFilename(file, opts.OutputDir, opts.OutputSuffix, "encrypt"),
		})
	}
	return filePairs, nil
}

func decryptFilePairs(opts Options) ([]FilePair, error) {
	if len(opts.FilePairs) > 0 {
		return opts.FilePairs, nil
	}

	files, err := collectFiles(opts.InputDir, opts.Pattern)
	if err != nil {
		return nil, err
	}

	filePairs := make([]FilePair, 0, len(files))
	for _, file := range files {
		filePairs = append(filePairs, FilePair{
			PlaintextPath: GenerateOutputFilename(file, opts.OutputDir, opts.OutputSuffix, "decrypt"),
			EncryptedPath: file,
		})
	}
	return filePairs, nil
}

func collectFiles(dir, pattern string) ([]string, error) {
	fullPattern := filepath.Join(dir, pattern)

	files, err := filepath.Glob(fullPattern)
	if err != nil {
		return nil, fmt.Errorf("invalid pattern %s: %w", pattern, err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no files found matching pattern %s in %s", pattern, dir)
	}

	return files, nil
}

func GenerateOutputFilename(inputFile, outputDir, outputSuffix, mode string) string {
	dir := filepath.Dir(inputFile)
	if outputDir != "" {
		dir = outputDir
	}

	base := filepath.Base(inputFile)

	if mode == "encrypt" {
		ext := filepath.Ext(base)
		name := strings.TrimSuffix(base, ext)
		return filepath.Join(dir, name+outputSuffix)
	}

	encSuffixes := []string{".enc.toml.yaml", ".enc.yaml", ".encrypted.yaml"}
	for _, encSuffix := range encSuffixes {
		if strings.HasSuffix(base, encSuffix) {
			name := strings.TrimSuffix(base, encSuffix)
			return filepath.Join(dir, name+outputSuffix)
		}
	}

	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	return filepath.Join(dir, name+outputSuffix)
}

func ensureOutputDir(outputDir string) error {
	if outputDir == "" {
		return nil
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
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
