package main

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

func docsCommand(rootCmd *cobra.Command) *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:    "docs",
		Short:  "Generate CLI documentation in Markdown format",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			var buf bytes.Buffer
			if err := writeMarkdownDocs(&buf, rootCmd); err != nil {
				return err
			}
			if err := os.WriteFile(output, buf.Bytes(), 0644); err != nil {
				return err
			}
			fmt.Printf("文档已生成: %s\n", output)
			return nil
		},
	}
	cmd.Flags().StringVarP(&output, "output", "o", "CLI.md", "Output file path")
	return cmd
}

func writeMarkdownDocs(w io.Writer, rootCmd *cobra.Command) error {
	rootCmd.InitDefaultHelpCmd()
	rootCmd.InitDefaultCompletionCmd()
	rootCmd.DisableAutoGenTag = true
	return walkCommands(rootCmd, func(cmd *cobra.Command) error {
		if cmd.Hidden || cmd.IsAdditionalHelpTopicCommand() {
			return nil
		}
		cmd.DisableAutoGenTag = true
		return doc.GenMarkdownCustom(cmd, w, func(link string) string { return link })
	})
}

func walkCommands(cmd *cobra.Command, fn func(*cobra.Command) error) error {
	if err := fn(cmd); err != nil {
		return err
	}
	for _, child := range cmd.Commands() {
		if err := walkCommands(child, fn); err != nil {
			return err
		}
	}
	return nil
}
