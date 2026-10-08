package vcs

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/YewFence/YewSeal/internal/execx"
)

// Status describes how a path participates in the current repository state.
type Status uint8

const (
	Unmanaged Status = iota
	Pending
	History
)

// Repository identifies a Git or jj working copy without invoking its CLI.
type Repository struct {
	root    string
	adapter adapter
}

// Snapshot contains the queried repository history and current working-copy
// state used to classify paths.
type Snapshot struct {
	root    string
	history map[string]bool
	pending map[string]bool
}

// Detect locates the repository containing dir without invoking Git or jj. It
// returns nil when dir is outside a repository. A colocated jj repository takes
// precedence over Git.
func Detect(dir string) (*Repository, error) {
	root, adapter, err := detect(dir)
	if err != nil || adapter == nil {
		return nil, err
	}
	return &Repository{root: root, adapter: adapter}, nil
}

func (r *Repository) Root() string {
	return r.root
}

func (r *Repository) Name() string {
	return r.adapter.name()
}

// DiscoveryFiles returns repository files eligible for config discovery
// without querying history or persisting a jj working-copy snapshot.
func (r *Repository) DiscoveryFiles() ([]string, error) {
	files, err := r.adapter.discoveryFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to query %s repository %s: %w", r.Name(), r.root, err)
	}
	return sortedPaths(files), nil
}

func (r *Repository) Snapshot() (*Snapshot, error) {
	history, pending, err := r.adapter.snapshot()
	if err != nil {
		return nil, fmt.Errorf("failed to query %s repository %s: %w", r.Name(), r.root, err)
	}
	return &Snapshot{
		root:    r.root,
		history: history,
		pending: pending,
	}, nil
}

func (s *Snapshot) Root() string {
	return s.root
}

func (s *Snapshot) Status(path string) Status {
	rel, err := filepath.Rel(s.root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return Unmanaged
	}
	rel = filepath.ToSlash(rel)
	if s.history[rel] {
		return History
	}
	if s.pending[rel] {
		return Pending
	}
	return Unmanaged
}

type adapter interface {
	name() string
	discoveryFiles() (map[string]bool, error)
	snapshot() (history, pending map[string]bool, err error)
}

func detect(dir string) (string, adapter, error) {
	current, err := filepath.Abs(dir)
	if err != nil {
		return "", nil, fmt.Errorf("failed to resolve repository search directory %s: %w", dir, err)
	}
	for {
		jjFound, err := markerExists(filepath.Join(current, ".jj"))
		if err != nil {
			return "", nil, err
		}
		if jjFound {
			return current, &jjAdapter{root: current}, nil
		}
		gitFound, err := markerExists(filepath.Join(current, ".git"))
		if err != nil {
			return "", nil, err
		}
		if gitFound {
			return current, &gitAdapter{root: current}, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", nil, nil
		}
		current = parent
	}
}

func markerExists(path string) (bool, error) {
	if _, err := os.Stat(path); err == nil {
		return true, nil
	} else if os.IsNotExist(err) {
		return false, nil
	} else {
		return false, fmt.Errorf("failed to inspect VCS marker %s: %w", path, err)
	}
}

type gitAdapter struct{ root string }

func (g *gitAdapter) name() string { return "git" }

func (g *gitAdapter) snapshot() (history, pending map[string]bool, err error) {
	history, err = g.list("ls-files", "-z", "--")
	if err != nil {
		return nil, nil, err
	}
	pending, err = g.list("ls-files", "-z", "--others", "--exclude-standard", "--")
	if err != nil {
		return nil, nil, err
	}
	return history, pending, nil
}

func (g *gitAdapter) discoveryFiles() (map[string]bool, error) {
	files, pending, err := g.snapshot()
	if err != nil {
		return nil, err
	}
	for path := range pending {
		files[path] = true
	}
	return files, nil
}

func (g *gitAdapter) list(args ...string) (map[string]bool, error) {
	stdout, stderr, err := execx.ExecCommand("git", append([]string{"-C", g.root}, args...)...)
	if err != nil {
		return nil, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr))
	}
	return splitNUL(stdout), nil
}

type jjAdapter struct{ root string }

func (j *jjAdapter) name() string { return "jj" }

func (j *jjAdapter) discoveryFiles() (map[string]bool, error) {
	const configAutoTrack = `snapshot.auto-track="root-glob:\"**/.yewseal.toml\""`
	stdout, stderr, err := execx.ExecCommand("jj", "--no-pager", "--no-integrate-operation", "--config", configAutoTrack, "-R", j.root, "file", "list", "-r", "@", "-T", `path ++ "\0"`)
	if err != nil {
		return nil, fmt.Errorf("jj file list -r @: %w: %s", err, strings.TrimSpace(stderr))
	}
	return splitNUL(stdout), nil
}

func (j *jjAdapter) snapshot() (history, pending map[string]bool, err error) {
	const template = `parents.map(|p| p.files().map(|f| "H" ++ f.path() ++ "\0").join("")).join("") ++ self.files().map(|f| "P" ++ f.path() ++ "\0").join("")`
	stdout, stderr, err := execx.ExecCommand("jj", "--no-pager", "--no-integrate-operation", "-R", j.root, "log", "--no-graph", "-r", "@", "-T", template)
	if err != nil {
		return nil, nil, fmt.Errorf("jj log -r @: %w: %s", err, strings.TrimSpace(stderr))
	}
	history = make(map[string]bool)
	pending = make(map[string]bool)
	for _, record := range strings.Split(stdout, "\x00") {
		if record == "" {
			continue
		}
		if len(record) < 2 {
			return nil, nil, fmt.Errorf("invalid jj snapshot record: missing path")
		}
		switch record[0] {
		case 'H':
			history[record[1:]] = true
		case 'P':
			pending[record[1:]] = true
		default:
			return nil, nil, fmt.Errorf("invalid jj snapshot record tag %q", record[0])
		}
	}
	return history, pending, nil
}

func sortedPaths(paths map[string]bool) []string {
	sorted := make([]string, 0, len(paths))
	for path := range paths {
		sorted = append(sorted, path)
	}
	sort.Strings(sorted)
	return sorted
}

func splitNUL(output string) map[string]bool {
	paths := make(map[string]bool)
	for _, path := range strings.Split(output, "\x00") {
		if path != "" {
			paths[path] = true
		}
	}
	return paths
}
