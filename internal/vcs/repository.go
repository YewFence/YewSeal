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

// Repository is a queried Git or jj working copy. Files returns Git's tracked
// and untracked non-ignored files, or the files in jj's working-copy commit.
// Returned paths are slash-separated and relative to Root.
type Repository struct {
	root    string
	files   map[string]bool
	history map[string]bool
	pending map[string]bool
}

// Open locates and queries the repository containing dir. It returns nil when
// dir is outside a repository. A colocated jj repository takes precedence over
// Git.
func Open(dir string) (*Repository, error) {
	root, adapter, err := detect(dir)
	if err != nil || adapter == nil {
		return nil, err
	}

	history, err := adapter.historyFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to query %s repository %s: %w", adapter.name(), root, err)
	}
	pending, err := adapter.pendingFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to query %s repository %s: %w", adapter.name(), root, err)
	}
	return &Repository{
		root:    root,
		files:   adapter.discoveryFiles(history, pending),
		history: history,
		pending: pending,
	}, nil
}

func (r *Repository) Root() string {
	return r.root
}

func (r *Repository) Files() []string {
	files := make([]string, 0, len(r.files))
	for path := range r.files {
		files = append(files, path)
	}
	sort.Strings(files)
	return files
}

func (r *Repository) Status(path string) Status {
	rel, err := filepath.Rel(r.root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return Unmanaged
	}
	rel = filepath.ToSlash(rel)
	if r.history[rel] {
		return History
	}
	if r.pending[rel] {
		return Pending
	}
	return Unmanaged
}

type adapter interface {
	name() string
	historyFiles() (map[string]bool, error)
	pendingFiles() (map[string]bool, error)
	discoveryFiles(history, pending map[string]bool) map[string]bool
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

func (g *gitAdapter) historyFiles() (map[string]bool, error) {
	return g.list("ls-files", "-z", "--")
}

func (g *gitAdapter) pendingFiles() (map[string]bool, error) {
	return g.list("ls-files", "-z", "--others", "--exclude-standard", "--")
}

func (g *gitAdapter) discoveryFiles(history, pending map[string]bool) map[string]bool {
	files := make(map[string]bool, len(history)+len(pending))
	for path := range history {
		files[path] = true
	}
	for path := range pending {
		files[path] = true
	}
	return files
}

func (g *gitAdapter) list(args ...string) (map[string]bool, error) {
	stdout, stderr, err := execx.ExecCommand("git", append([]string{"-C", g.root}, args...)...)
	if err != nil {
		return nil, fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr))
	}
	return splitNUL(stdout), nil
}

type jjAdapter struct{ root string }

func (j *jjAdapter) name() string { return "jj" }

func (j *jjAdapter) historyFiles() (map[string]bool, error) {
	stdout, stderr, err := execx.ExecCommand("jj", "--no-pager", "-R", j.root, "log", "--no-graph", "-r", "@-", "-T", `commit_id ++ "\0"`)
	if err != nil {
		return nil, fmt.Errorf("jj log -r @-: %v: %s", err, strings.TrimSpace(stderr))
	}
	history := make(map[string]bool)
	for parent := range splitNUL(stdout) {
		files, err := j.list(parent)
		if err != nil {
			return nil, err
		}
		for path := range files {
			history[path] = true
		}
	}
	return history, nil
}

func (j *jjAdapter) pendingFiles() (map[string]bool, error) {
	return j.list("@")
}

func (j *jjAdapter) discoveryFiles(_, pending map[string]bool) map[string]bool {
	return pending
}

func (j *jjAdapter) list(revision string) (map[string]bool, error) {
	stdout, stderr, err := execx.ExecCommand("jj", "--no-pager", "-R", j.root, "file", "list", "-r", revision, "-T", `path ++ "\0"`)
	if err != nil {
		return nil, fmt.Errorf("jj file list -r %s: %v: %s", revision, err, strings.TrimSpace(stderr))
	}
	return splitNUL(stdout), nil
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
