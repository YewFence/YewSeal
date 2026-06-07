package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/YewFence/YewSeal/internal/config"
)

var Version = "dev"

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to load config: %v\n", err)
		os.Exit(1)
	}

	rootCmd := rootCommand(cfg)
	if err := rootCmd.Execute(); err != nil {
		if strings.TrimSpace(err.Error()) != "" {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}
}
