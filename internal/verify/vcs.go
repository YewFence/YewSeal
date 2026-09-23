package verify

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/YewFence/YewSeal/internal/execx"
)

// vcsAdapter exposes the dual-list contract shared by git and jj. Paths are
// slash-separated and relative to the repository root.
type vcsAdapter interface {
	name() string
	// historyFiles are already in (or bound for) version-control history: error.
	historyFiles() (map[string]bool, error)
	// pendingFiles are one step away from history: warning.
	pendingFiles() (map[string]bool, error)
}

// vcsState is the result of querying the repository once per verify run.
type vcsState struct {
	root    string
	history map[string]bool
	pending map[string]bool
}

// detectVCS walks up from dir looking for .jj or .git. .jj wins because jj owns
// the working copy of a colocated repository.
func detectVCS(dir string) (string, vcsAdapter) {
	current := dir
	for {
		if _, err := os.Stat(filepath.Join(current, ".jj")); err == nil {
			return current, &jjAdapter{root: current}
		}
		if _, err := os.Stat(filepath.Join(current, ".git")); err == nil {
			return current, &gitAdapter{root: current}
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", nil
		}
		current = parent
	}
}

func queryVCS(root string, adapter vcsAdapter) (vcsState, error) {
	history, err := adapter.historyFiles()
	if err != nil {
		return vcsState{}, err
	}
	pending, err := adapter.pendingFiles()
	if err != nil {
		return vcsState{}, err
	}
	return vcsState{root: root, history: history, pending: pending}, nil
}

// vcsSubject names what is classified, so plaintext and key findings share one
// classification path but keep distinct codes and wording.
type vcsSubject struct {
	trackedCode    string
	notIgnoredCode string
	noun           string
}

var (
	plaintextSubject = vcsSubject{trackedCode: "plaintext_tracked", notIgnoredCode: "plaintext_not_ignored", noun: "plaintext file"}
	keySubject       = vcsSubject{trackedCode: "key_tracked", notIgnoredCode: "key_not_ignored", noun: "Age private key file"}
)

// classify applies the dual-list contract to one absolute path. finding
// carries the mapping paths for plaintext subjects and stays path-less for keys.
func (s vcsState) classify(report *Report, absPath string, subject vcsSubject, finding Finding) {
	rel, err := filepath.Rel(s.root, absPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		report.AddPass()
		return
	}
	rel = filepath.ToSlash(rel)
	switch {
	case s.history[rel]:
		finding.Code = subject.trackedCode
		finding.Severity = SeverityError
		finding.Message = fmt.Sprintf("%s %s is tracked by version control", subject.noun, rel)
		finding.Hint = "remove it from version control and review repository history; ignoring it now does not remove it from history"
	case s.pending[rel]:
		finding.Code = subject.notIgnoredCode
		finding.Severity = SeverityWarning
		finding.Message = fmt.Sprintf("%s %s is not ignored and would enter history with the next commit", subject.noun, rel)
		finding.Hint = "add it to .gitignore"
	default:
		report.AddPass()
		return
	}
	report.Add(finding)
}

// gitAdapter uses the index as history: a staged file enters the next commit
// even when it is ignored afterwards.
type gitAdapter struct{ root string }

func (g *gitAdapter) name() string { return "git" }

func (g *gitAdapter) historyFiles() (map[string]bool, error) {
	return g.list("ls-files", "-z", "--")
}

func (g *gitAdapter) pendingFiles() (map[string]bool, error) {
	return g.list("ls-files", "-z", "--others", "--exclude-standard", "--")
}

func (g *gitAdapter) list(args ...string) (map[string]bool, error) {
	stdout, stderr, err := execx.ExecCommand("git", append([]string{"-C", g.root}, args...)...)
	if err != nil {
		return nil, fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr))
	}
	return splitNUL(stdout), nil
}

// jjAdapter has no untracked state: @ absorbs every non-ignored file, so @-
// (the latest real commit) is history and @ is the scratch snapshot.
type jjAdapter struct{ root string }

func (j *jjAdapter) name() string { return "jj" }

func (j *jjAdapter) historyFiles() (map[string]bool, error) { return j.list("@-") }

func (j *jjAdapter) pendingFiles() (map[string]bool, error) { return j.list("@") }

// list uses a template because jj's default output is relative to the process
// working directory, while classification compares repository-root paths.
// NUL separation keeps file names containing newlines unambiguous.
func (j *jjAdapter) list(revision string) (map[string]bool, error) {
	stdout, stderr, err := execx.ExecCommand("jj", "--no-pager", "-R", j.root, "file", "list", "-r", revision, "-T", `path ++ "\0"`)
	if err != nil {
		return nil, fmt.Errorf("jj file list -r %s: %v: %s", revision, err, strings.TrimSpace(stderr))
	}
	return splitNUL(stdout), nil
}

func splitNUL(s string) map[string]bool {
	m := make(map[string]bool)
	for _, p := range strings.Split(s, "\x00") {
		if p != "" {
			m[p] = true
		}
	}
	return m
}
