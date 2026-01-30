package crypto

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// BatchOptions 批量操作的配置选项
type BatchOptions struct {
	InputDir     string // 输入目录
	Pattern      string // 文件匹配模式 (glob)
	OutputDir    string // 输出目录（空则与输入同目录）
	OutputSuffix string // 输出文件后缀
	KeyFile      string // 密钥文件路径
	PublicKey    string // Age 公钥（加密时使用）
	Parallel     int    // 并行度（1=顺序执行）
	Verbose      bool   // 详细输出
	// FilePairs 直接指定的文件对列表（来自配置文件）
	FilePairs []FilePair
}

// FilePair 文件对（输入和输出）
type FilePair struct {
	Input  string
	Output string
}

// BatchResult 单个文件的处理结果
type BatchResult struct {
	InputFile  string // 输入文件路径
	OutputFile string // 输出文件路径
	Success    bool   // 是否成功
	Error      error  // 错误信息（如果失败）
}

// BatchSummary 批量操作的汇总结果
type BatchSummary struct {
	TotalFiles   int           // 总文件数
	SuccessCount int           // 成功数
	FailedCount  int           // 失败数
	Results      []BatchResult // 详细结果
}

// collectFiles 根据目录和模式收集待处理的文件列表
func collectFiles(dir, pattern string) ([]string, error) {
	// 构建完整的 glob 模式
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

// GenerateOutputFilename 根据输入文件生成输出文件名
// inputFile: 输入文件路径
// outputDir: 输出目录（空则使用输入文件所在目录）
// outputSuffix: 输出文件后缀
// mode: "encrypt" 或 "decrypt"
func GenerateOutputFilename(inputFile, outputDir, outputSuffix, mode string) string {
	dir := filepath.Dir(inputFile)
	if outputDir != "" {
		dir = outputDir
	}

	base := filepath.Base(inputFile)

	if mode == "encrypt" {
		// 加密模式：config.toml -> config.enc.toml.yaml
		ext := filepath.Ext(base) // .toml
		name := strings.TrimSuffix(base, ext)
		return filepath.Join(dir, name+outputSuffix)
	}

	// 解密模式：config.enc.toml.yaml -> config.toml
	// 尝试匹配常见加密后缀模式
	encSuffixes := []string{".enc.toml.yaml", ".enc.yaml", ".encrypted.yaml"}
	for _, encSuffix := range encSuffixes {
		if strings.HasSuffix(base, encSuffix) {
			name := strings.TrimSuffix(base, encSuffix)
			return filepath.Join(dir, name+outputSuffix)
		}
	}

	// 回退：直接替换扩展名
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	return filepath.Join(dir, name+outputSuffix)
}

// BatchEncrypt 批量加密文件
func BatchEncrypt(opts BatchOptions) (*BatchSummary, error) {
	// 确定文件列表来源
	var filePairs []FilePair

	if len(opts.FilePairs) > 0 {
		// 使用配置文件中的文件对
		filePairs = opts.FilePairs
		fmt.Printf("Encrypting %d files from config...\n", len(filePairs))
	} else {
		// 使用目录扫描
		files, err := collectFiles(opts.InputDir, opts.Pattern)
		if err != nil {
			return nil, err
		}
		fmt.Printf("Encrypting %d files from %s...\n", len(files), opts.InputDir)

		// 转换为文件对
		for _, f := range files {
			filePairs = append(filePairs, FilePair{
				Input:  f,
				Output: GenerateOutputFilename(f, opts.OutputDir, opts.OutputSuffix, "encrypt"),
			})
		}
	}

	// 确保输出目录存在
	if opts.OutputDir != "" {
		if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	// 定义处理器函数
	processor := func(input, output string) error {
		return Encrypt(input, output, opts.KeyFile, opts.PublicKey, opts.Verbose)
	}

	// 根据并行度选择处理方式
	var summary *BatchSummary
	if opts.Parallel > 1 {
		summary = processFilePairsParallel(filePairs, opts, processor)
	} else {
		summary = processFilePairsSequential(filePairs, opts, processor)
	}

	// 打印汇总
	printSummary(summary, "encrypted")

	if summary.FailedCount > 0 {
		return summary, fmt.Errorf("%d of %d files failed to encrypt", summary.FailedCount, summary.TotalFiles)
	}

	return summary, nil
}

// BatchDecrypt 批量解密文件
func BatchDecrypt(opts BatchOptions) (*BatchSummary, error) {
	// 确定文件列表来源
	var filePairs []FilePair

	if len(opts.FilePairs) > 0 {
		// 使用配置文件中的文件对（解密时输入输出互换）
		for _, fp := range opts.FilePairs {
			filePairs = append(filePairs, FilePair{
				Input:  fp.Output, // 加密后的文件作为输入
				Output: fp.Input,  // 原文件作为输出
			})
		}
		fmt.Printf("Decrypting %d files from config...\n", len(filePairs))
	} else {
		// 使用目录扫描
		files, err := collectFiles(opts.InputDir, opts.Pattern)
		if err != nil {
			return nil, err
		}
		fmt.Printf("Decrypting %d files from %s...\n", len(files), opts.InputDir)

		// 转换为文件对
		for _, f := range files {
			filePairs = append(filePairs, FilePair{
				Input:  f,
				Output: GenerateOutputFilename(f, opts.OutputDir, opts.OutputSuffix, "decrypt"),
			})
		}
	}

	// 确保输出目录存在
	if opts.OutputDir != "" {
		if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	// 定义处理器函数
	processor := func(input, output string) error {
		return Decrypt(input, output, opts.KeyFile, opts.Verbose)
	}

	// 根据并行度选择处理方式
	var summary *BatchSummary
	if opts.Parallel > 1 {
		summary = processFilePairsParallel(filePairs, opts, processor)
	} else {
		summary = processFilePairsSequential(filePairs, opts, processor)
	}

	// 打印汇总
	printSummary(summary, "decrypted")

	if summary.FailedCount > 0 {
		return summary, fmt.Errorf("%d of %d files failed to decrypt", summary.FailedCount, summary.TotalFiles)
	}

	return summary, nil
}

// processFilePairsSequential 顺序处理文件对
func processFilePairsSequential(pairs []FilePair, opts BatchOptions, processor func(input, output string) error) *BatchSummary {
	summary := &BatchSummary{
		TotalFiles: len(pairs),
		Results:    make([]BatchResult, 0, len(pairs)),
	}

	for i, pair := range pairs {
		// 打印进度
		fmt.Printf("  [%d/%d] %s -> %s ... ", i+1, summary.TotalFiles, filepath.Base(pair.Input), filepath.Base(pair.Output))

		err := processor(pair.Input, pair.Output)

		result := BatchResult{
			InputFile:  pair.Input,
			OutputFile: pair.Output,
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

// processFilePairsParallel 并行处理文件对
func processFilePairsParallel(pairs []FilePair, opts BatchOptions, processor func(input, output string) error) *BatchSummary {
	summary := &BatchSummary{
		TotalFiles: len(pairs),
		Results:    make([]BatchResult, len(pairs)),
	}

	var wg sync.WaitGroup
	jobs := make(chan int, len(pairs))
	var mu sync.Mutex
	var completed int

	// 启动 workers
	for w := 0; w < opts.Parallel; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobs {
				pair := pairs[idx]

				err := processor(pair.Input, pair.Output)

				summary.Results[idx] = BatchResult{
					InputFile:  pair.Input,
					OutputFile: pair.Output,
					Success:    err == nil,
					Error:      err,
				}

				// 线程安全地更新进度
				mu.Lock()
				completed++
				current := completed
				mu.Unlock()

				// 打印进度
				if err == nil {
					fmt.Printf("  [%d/%d] %s -> %s ... OK\n", current, summary.TotalFiles, filepath.Base(pair.Input), filepath.Base(pair.Output))
				} else {
					fmt.Printf("  [%d/%d] %s -> %s ... FAILED\n        Error: %v\n", current, summary.TotalFiles, filepath.Base(pair.Input), filepath.Base(pair.Output), err)
				}
			}
		}()
	}

	// 分发任务
	for i := range pairs {
		jobs <- i
	}
	close(jobs)

	wg.Wait()

	// 统计结果
	for _, r := range summary.Results {
		if r.Success {
			summary.SuccessCount++
		} else {
			summary.FailedCount++
		}
	}

	return summary
}

// printSummary 打印处理汇总
func printSummary(summary *BatchSummary, action string) {
	fmt.Println()
	if summary.FailedCount == 0 {
		fmt.Printf("✅ Summary: %d/%d files %s successfully\n", summary.SuccessCount, summary.TotalFiles, action)
	} else {
		fmt.Printf("⚠️  Summary: %d/%d succeeded, %d failed\n", summary.SuccessCount, summary.TotalFiles, summary.FailedCount)
		fmt.Println("\nFailed files:")
		for _, r := range summary.Results {
			if !r.Success {
				fmt.Printf("  - %s: %v\n", filepath.Base(r.InputFile), r.Error)
			}
		}
	}
}
