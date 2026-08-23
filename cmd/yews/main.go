package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/YewFence/YewSeal/internal/cli"
	"github.com/YewFence/YewSeal/internal/config"
)

// 由构建期 -ldflags="-X main.version=..." 注入
var version = "dev"

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to load config: %v\n", err)
		os.Exit(1)
	}

	if err := cli.NewRootCommand(cfg, version).Execute(); err != nil {
		if strings.TrimSpace(err.Error()) != "" {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}
}
