// gendocs 生成 yews CLI 的 Markdown 参考文档，每个命令一页，供 Astro Starlight 站点使用。
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/YewFence/YewSeal/internal/cli"

	"github.com/spf13/cobra/doc"
)

func main() {
	var output string
	flag.StringVar(&output, "o", "docs/src/content/docs/references", "Output directory path")
	flag.Parse()

	if err := generate(output); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("文档已生成: %s\n", output)
}

func generate(output string) error {
	rootCmd := cli.NewRootCommand("dev")
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

	if err := doc.GenMarkdownTreeCustom(rootCmd, output, filePrepender, linkRewriter); err != nil {
		return err
	}

	// Starlight 将 frontmatter title 渲染为页面大标题，删除正文开头重复的命令标题行；
	// cobra 生成的页面末尾有多余空行，修剪为单个换行符以通过 newline lint
	pages, err := filepath.Glob(filepath.Join(output, "yews*.md"))
	if err != nil {
		return err
	}
	for _, path := range pages {
		if err := stripDuplicateHeading(path); err != nil {
			return err
		}
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

// linkRewriter 把 cobra 生成的 "yews_init.md" 文件链接改写为 Starlight 的
// "yews_init/" 目录路由链接。
func linkRewriter(link string) string {
	return strings.TrimSuffix(link, ".md") + "/"
}

// filePrepender 为每个生成的页面注入 Starlight frontmatter，
// 用命令路径（如 "yews encrypt"）作为页面标题。
func filePrepender(filename string) string {
	return fmt.Sprintf("---\ntitle: %s\n---\n\n", commandTitle(filename))
}

// commandTitle 把页面文件名（如 "yews_encrypt.md"）转换为命令路径标题（如 "yews encrypt"）。
func commandTitle(path string) string {
	return strings.ReplaceAll(strings.TrimSuffix(filepath.Base(path), ".md"), "_", " ")
}

// stripDuplicateHeading 删除正文开头与 frontmatter title 相同的 "## <title>" 标题行，
// 避免页面出现重复标题。
func stripDuplicateHeading(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	title := commandTitle(path)
	frontmatter := "---\ntitle: " + title + "\n---\n\n"
	trimmed := strings.TrimPrefix(string(content), frontmatter+"## "+title+"\n\n")
	if trimmed == string(content) {
		return nil
	}
	return os.WriteFile(path, []byte(frontmatter+trimmed), 0o644)
}
