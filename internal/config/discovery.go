package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/YewFence/YewSeal/internal/vcs"
)

var configLocations = []string{
	filepath.Join(".yewseal", ".yewseal.toml"),
	filepath.Join(".config", ".yewseal.toml"),
	".yewseal.toml",
}

type configCandidate struct {
	file     LoadedFile
	priority int
}

func discoverConfigFiles(cwd string) ([]LoadedFile, error) {
	repository, err := vcs.Open(cwd)
	if err != nil {
		return nil, err
	}
	if repository == nil {
		path, err := highestPriorityConfigPath(cwd)
		if err != nil || path == "" {
			return nil, err
		}
		return []LoadedFile{{Path: path, Dir: cwd}}, nil
	}

	selected := make(map[string]configCandidate)
	directDirs, err := configSearchDirs(repository.Root(), cwd)
	if err != nil {
		return nil, err
	}
	for _, dir := range directDirs {
		path, err := highestPriorityConfigPath(dir)
		if err != nil {
			return nil, err
		}
		if path == "" {
			continue
		}
		priority := configLocationPriority(dir, path)
		selected[dir] = configCandidate{file: LoadedFile{Path: path, Dir: dir}, priority: priority}
	}

	for _, relativePath := range repository.Files() {
		candidate, ok := repositoryConfigCandidate(repository.Root(), relativePath)
		if !ok {
			continue
		}
		if _, err := os.Stat(candidate.file.Path); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("failed to stat config file %s: %w", candidate.file.Path, err)
		}
		existing, found := selected[candidate.file.Dir]
		if !found || candidate.priority < existing.priority {
			selected[candidate.file.Dir] = candidate
		}
	}

	files := make([]LoadedFile, 0, len(selected))
	for _, candidate := range selected {
		files = append(files, candidate.file)
	}
	sort.Slice(files, func(i, j int) bool {
		leftDepth := directoryDepth(repository.Root(), files[i].Dir)
		rightDepth := directoryDepth(repository.Root(), files[j].Dir)
		if leftDepth != rightDepth {
			return leftDepth < rightDepth
		}
		return filepath.ToSlash(files[i].Path) < filepath.ToSlash(files[j].Path)
	})
	return files, nil
}

func configSearchDirs(root, cwd string) ([]string, error) {
	relative, err := filepath.Rel(root, cwd)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve current directory relative to repository root: %w", err)
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("current directory %s is outside repository root %s", cwd, root)
	}
	dirs := []string{root}
	if relative == "." {
		return dirs, nil
	}
	for _, part := range strings.Split(relative, string(filepath.Separator)) {
		if part == "" || part == "." {
			continue
		}
		dirs = append(dirs, filepath.Join(dirs[len(dirs)-1], part))
	}
	return dirs, nil
}

func highestPriorityConfigPath(dir string) (string, error) {
	for _, relativePath := range configLocations {
		path := filepath.Join(dir, relativePath)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("failed to stat config file %s: %w", path, err)
		}
	}
	return "", nil
}

func configLocationPriority(dir, path string) int {
	relative, err := filepath.Rel(dir, path)
	if err != nil {
		return len(configLocations)
	}
	for priority, location := range configLocations {
		if relative == location {
			return priority
		}
	}
	return len(configLocations)
}

func repositoryConfigCandidate(root, relativePath string) (configCandidate, bool) {
	path := filepath.Clean(filepath.FromSlash(relativePath))
	if !filepath.IsLocal(path) || filepath.Base(path) != ".yewseal.toml" {
		return configCandidate{}, false
	}

	owner := filepath.Dir(path)
	priority := 2
	switch filepath.Base(owner) {
	case ".yewseal":
		owner = filepath.Dir(owner)
		priority = 0
	case ".config":
		owner = filepath.Dir(owner)
		priority = 1
	}
	if owner == "." {
		owner = ""
	}
	dir := filepath.Clean(filepath.Join(root, owner))
	return configCandidate{
		file: LoadedFile{
			Path: filepath.Clean(filepath.Join(root, path)),
			Dir:  dir,
		},
		priority: priority,
	}, true
}

func directoryDepth(root, dir string) int {
	relative, err := filepath.Rel(root, dir)
	if err != nil || relative == "." {
		return 0
	}
	return len(strings.Split(relative, string(filepath.Separator)))
}
