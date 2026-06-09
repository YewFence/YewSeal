package app

import (
	"fmt"

	"github.com/YewFence/YewSeal/internal/config"
)

func printResolvedSelection(verbose bool, cfg *config.Config, selection config.ResolvedSelection) {
	if !verbose {
		return
	}
	if len(selection.ConfigFiles) > 0 {
		fmt.Printf("Loaded %d config files\n", len(selection.ConfigFiles))
	}
	fmt.Printf("Selected %d file pairs\n", len(selection.FilePairs))
	if selection.CurrentDirScope != "" {
		fmt.Printf("Using current directory scope: %s\n", config.DisplayPath(config.CurrentDir(cfg), selection.CurrentDirScope))
	}
	for _, filePair := range selection.FilePairs {
		fmt.Printf("  %s -> %s\n", config.DisplayPath(config.CurrentDir(cfg), filePair.PlaintextPath), config.DisplayPath(config.CurrentDir(cfg), filePair.EncryptedPath))
	}
}
