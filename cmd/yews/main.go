package main

import (
	"os"

	"github.com/YewFence/YewSeal/internal/cli"
	"github.com/YewFence/YewSeal/internal/presentation"
)

// 由构建期 -ldflags="-X main.version=..." 注入
var version = "dev"

func main() {
	cmd, err := cli.NewRootCommand(version).ExecuteC()
	if err != nil {
		presentation.New(os.Stdout, os.Stderr, false).Error(err)
		os.Exit(cli.ExitCode(cmd, err))
	}
}
