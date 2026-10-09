package config

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
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

type configDiscovery struct {
	root     string
	files    []LoadedFile
	warnings []string
	degraded bool
}

func discoverConfigFiles(cwd string) (configDiscovery, error) {
	repository, err := vcs.Detect(cwd)
	if err != nil {
		return discoverCurrentConfig(cwd, repositoryDetectionWarning(err))
	}
	if repository == nil {
		return discoverCurrentConfig(cwd, "")
	}

	directDirs, err := configSearchDirs(repository.Root(), cwd)
	if err != nil {
		return discoverCurrentConfig(cwd, repositoryDetectionWarning(err))
	}
	selected, err := discoverDirectConfigs(directDirs)
	if err != nil {
		return configDiscovery{}, err
	}

	relativePaths, err := repository.DiscoveryFiles()
	if err != nil {
		return configDiscovery{
			root:     repository.Root(),
			files:    sortedConfigFiles(repository.Root(), selected),
			warnings: []string{repositoryQueryWarning(repository.Name(), err)},
			degraded: true,
		}, nil
	}
	for _, relativePath := range relativePaths {
		candidate, ok := repositoryConfigCandidate(repository.Root(), relativePath)
		if !ok {
			continue
		}
		if _, err := os.Stat(candidate.file.Path); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return configDiscovery{}, fmt.Errorf("failed to stat config file %s: %w", candidate.file.Path, err)
		}
		existing, found := selected[candidate.file.Dir]
		if !found || candidate.priority < existing.priority {
			selected[candidate.file.Dir] = candidate
		}
	}
	return configDiscovery{root: repository.Root(), files: sortedConfigFiles(repository.Root(), selected)}, nil
}

func discoverCurrentConfig(cwd, warning string) (configDiscovery, error) {
	path, err := highestPriorityConfigPath(cwd)
	if err != nil {
		return configDiscovery{}, err
	}
	discovery := configDiscovery{root: cwd}
	if path != "" {
		discovery.files = []LoadedFile{{Path: path, Dir: cwd}}
	}
	if warning != "" {
		discovery.warnings = []string{warning}
		discovery.degraded = true
	}
	return discovery, nil
}

func discoverDirectConfigs(dirs []string) (map[string]configCandidate, error) {
	selected := make(map[string]configCandidate)
	for _, dir := range dirs {
		path, err := highestPriorityConfigPath(dir)
		if err != nil {
			return nil, err
		}
		if path == "" {
			continue
		}
		selected[dir] = configCandidate{
			file:     LoadedFile{Path: path, Dir: dir},
			priority: configLocationPriority(dir, path),
		}
	}
	return selected, nil
}

func sortedConfigFiles(root string, selected map[string]configCandidate) []LoadedFile {
	files := make([]LoadedFile, 0, len(selected))
	for _, candidate := range selected {
		files = append(files, candidate.file)
	}
	sort.Slice(files, func(i, j int) bool {
		leftDepth := directoryDepth(root, files[i].Dir)
		rightDepth := directoryDepth(root, files[j].Dir)
		if leftDepth != rightDepth {
			return leftDepth < rightDepth
		}
		return filepath.ToSlash(files[i].Path) < filepath.ToSlash(files[j].Path)
	})
	return files
}

func repositoryQueryWarning(name string, err error) string {
	var unavailable *exec.Error
	if errors.As(err, &unavailable) {
		return fmt.Sprintf("%s is unavailable; continuing with configs found from the repository root to the current directory only. Install %s to enable repository-wide config discovery", name, name)
	}
	return fmt.Sprintf("could not enumerate repository-wide configs with %s; continuing with configs found from the repository root to the current directory only: %v. Other repository configs may be missing. If %s can read this repository normally, report this at https://github.com/YewFence/YewSeal/issues", name, err, name)
}

func repositoryDetectionWarning(err error) string {
	return fmt.Sprintf("could not determine the repository config search path; continuing with the current-directory config only: %v. Other repository configs may be missing. If this directory belongs to a healthy Git or jj repository, report this at https://github.com/YewFence/YewSeal/issues", err)
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
