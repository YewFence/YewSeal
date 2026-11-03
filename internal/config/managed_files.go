package config

import (
	"path/filepath"
	"sort"
)

// ManagedFileScope contains the mappings owned by configurations in one directory.
type ManagedFileScope struct {
	Dir       string
	FilePairs []ResolvedFilePair
}

// GroupManagedFilePairs groups mappings by their source configuration's logical
// directory, including configurations whose current policy has no mappings.
func GroupManagedFilePairs(pairs []ResolvedFilePair, files []LoadedFile, fallbackDir string) []ManagedFileScope {
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
