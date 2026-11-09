package config

import (
	"sort"

	"github.com/YewFence/YewSeal/internal/task"
)

// CompletionPaths lists every registered path that target selection accepts
// for command, as display paths relative to the current directory: plaintext
// for encrypt, clean, and diff; encrypted for decrypt and view; both sides
// for plan and verify (directionless). It runs discovery only — no target
// selectors and no directory scope — so completion can suggest any registered
// path regardless of where the command itself would focus.
func CompletionPaths(cfg *Config, command string) ([]string, error) {
	filePairs, err := DiscoverFilePairs(cfg, command)
	if err != nil {
		return nil, err
	}
	cwd := CurrentDir(cfg)
	seen := make(map[string]struct{})
	paths := make([]string, 0, len(filePairs))
	add := func(path string) {
		display := DisplayPath(cwd, path)
		if _, duplicate := seen[display]; duplicate {
			return
		}
		seen[display] = struct{}{}
		paths = append(paths, display)
	}
	for _, pair := range filePairs {
		switch policyForCommand(command).scopeMode {
		case task.ModeDecrypt:
			add(pair.EncryptedPath)
		case task.ModePlan:
			add(pair.PlaintextPath)
			add(pair.EncryptedPath)
		default:
			add(pair.PlaintextPath)
		}
	}
	sort.Strings(paths)
	return paths, nil
}

// DiscoverFilePairs returns every mapping visible to command's discovery
// pipeline (explicit files plus group discovery), without applying target
// selectors or directory scope.
func DiscoverFilePairs(cfg *Config, command string) ([]FilePair, error) {
	var stats task.ScanStats
	return newSelectionDiscovery(cfg, &stats).filePairs(command, SelectionOptions{Command: command})
}
