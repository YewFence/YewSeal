package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/YewFence/YewSeal/internal/cli"
)

// 由构建期 -ldflags="-X main.version=..." 注入
var version = "dev"

func main() {
	if err := cli.NewRootCommand(version).Execute(); err != nil {
		if strings.TrimSpace(err.Error()) != "" {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}
}
