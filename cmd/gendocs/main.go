// gendocs 生成 yews CLI 的 Markdown 参考文档，每个命令一页，供 VitePress 站点使用。
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/YewFence/YewSeal/internal/cli"
	"github.com/YewFence/YewSeal/internal/config"

	"github.com/spf13/cobra/doc"
)

func main() {
	var output string
	flag.StringVar(&output, "o", "docs/references", "Output directory path")
	flag.Parse()

	if err := generate(output); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("文档已生成: %s\n", output)
}

func generate(output string) error {
	rootCmd := cli.NewRootCommand(&config.Config{}, "dev")
	rootCmd.InitDefaultCompletionCmd()

	if err := os.MkdirAll(output, 0o755); err != nil {
		return err
	}
	// 清掉上次生成的页面，避免已删除或改名的命令残留
	stale, err := filepath.Glob(filepath.Join(output, "yews*.md"))
	if err != nil {
		return err
	}
	for _, path := range stale {
		if err := os.Remove(path); err != nil {
			return err
		}
	}

	if err := doc.GenMarkdownTreeCustom(rootCmd, output, filePrepender, func(link string) string { return link }); err != nil {
		return err
	}

	// cobra 生成的页面末尾有多余空行，修剪为单个换行符以通过 newline lint
	pages, err := filepath.Glob(filepath.Join(output, "yews*.md"))
	if err != nil {
		return err
	}
	for _, path := range pages {
		if err := trimTrailingBlankLines(path); err != nil {
			return err
		}
	}
	return nil
}

func trimTrailingBlankLines(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	trimmed := append(bytes.TrimRight(content, "\n"), '\n')
	if bytes.Equal(content, trimmed) {
		return nil
	}
	return os.WriteFile(path, trimmed, 0o644)
}

// filePrepender 为每个生成的页面注入 VitePress frontmatter，
// 用命令路径（如 "yews encrypt"）作为页面标题。
func filePrepender(filename string) string {
	name := strings.TrimSuffix(filepath.Base(filename), ".md")
	return fmt.Sprintf("---\ntitle: %s\n---\n\n", strings.ReplaceAll(name, "_", " "))
}
