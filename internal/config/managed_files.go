package config

import (
	"path/filepath"
	"sort"

	"github.com/YewFence/YewSeal/internal/task"
)

// ManagedFileScope contains the mappings owned by configurations in one directory.
type ManagedFileScope struct {
	Dir       string
	FilePairs []ResolvedFilePair
}

func (d *selectionDiscovery) managedScopes(opts SelectionOptions, allPairs []ResolvedFilePair) ([]ManagedFileScope, error) {
	metadataCommand := task.ModePlan
	if policyForCommand(opts.Command).historicalRecipients && !opts.StrictRecipients {
		metadataCommand = task.ModeClean
	}
	metadataPairs := allPairs
	if policyForCommand(opts.Command).discoveryMode != policyForCommand(metadataCommand).discoveryMode {
		pairs, err := d.filePairs(metadataCommand, opts)
		if err != nil {
			return nil, err
		}
		metadataPairs, err = resolveFilePairs(d.cfg, pairs, SelectionOptions{Command: metadataCommand})
		if err != nil {
			return nil, err
		}
		if policyForCommand(metadataCommand).writes {
			scopedPairs, err := filterCurrentDirectoryScope(pairs, metadataCommand, cwdFromConfig(d.cfg))
			if err != nil {
				return nil, err
			}
			resolvedScope, err := resolveFilePairs(d.cfg, scopedPairs, SelectionOptions{Command: metadataCommand})
			if err != nil {
				return nil, err
			}
			if err := checkWriteConflicts(metadataCommand, resolvedScope); err != nil {
				return nil, err
			}
		}
	}
	return groupManagedFilePairs(metadataPairs, d.cfg.LoadedFiles, cwdFromConfig(d.cfg)), nil
}

func groupManagedFilePairs(pairs []ResolvedFilePair, files []LoadedFile, fallbackDir string) []ManagedFileScope {
	directories := make(map[string]string, len(files))
	grouped := make(map[string][]ResolvedFilePair)
	for _, file := range files {
		directories[file.Path] = file.Dir
		grouped[file.Dir] = nil
	}
	for _, pair := range pairs {
		dir, found := directories[pair.ConfigPath]
		if !found {
			dir = fallbackDir
			if pair.ConfigPath != "" {
				dir = filepath.Dir(resolveCommandPath(fallbackDir, pair.ConfigPath))
			}
		}
		grouped[dir] = append(grouped[dir], pair)
	}
	scopes := make([]ManagedFileScope, 0, len(grouped))
	for dir, filePairs := range grouped {
		scopes = append(scopes, ManagedFileScope{Dir: dir, FilePairs: filePairs})
	}
	sort.Slice(scopes, func(i, j int) bool { return scopes[i].Dir < scopes[j].Dir })
	return scopes
}
