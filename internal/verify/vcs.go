package verify

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/YewFence/YewSeal/internal/execx"
)

// vcsAdapter abstracts the dual-list contract over git and jj.
type vcsAdapter interface {
	// committedFiles returns files that are already in version control history (list A → error).
	committedFiles() (map[string]bool, error)
	// trackedFiles returns files that are one step away from history (list B → warning).
	// For jj this is @; for git this is untracked-not-excluded files.
	pendingFiles() (map[string]bool, error)
}

// detectVCS walks up from dir looking for .jj or .git, preferring .jj.
// Returns the repo root and adapter, or nil if not in a VCS repo.
func detectVCS(dir string) (string, vcsAdapter, error) {
	current := dir
	for {
		// jj takes precedence over git (colocated repos have both, jj semantics apply)
		if _, err := os.Stat(filepath.Join(current, ".jj")); err == nil {
			return current, &jjAdapter{root: current}, nil
		}
		if _, err := os.Stat(filepath.Join(current, ".git")); err == nil {
			return current, &gitAdapter{root: current}, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", nil, nil
		}
		current = parent
	}
}

// checkVCS runs the VCS layer for a single plaintext path and optional key file path.
func checkVCS(report *Report, plaintextPath, keyFilePath, repoRoot string, adapter vcsAdapter) {
	committed, err := adapter.committedFiles()
	if err != nil {
		report.Add(Finding{
			Code:          "vcs_query_failed",
			Severity:      SeverityError,
			PlaintextPath: plaintextPath,
			Message:       fmt.Sprintf("VCS query failed: %v", err),
			Hint:          "ensure git or jj is installed and the repository is accessible",
		})
		return
	}
	pending, err := adapter.pendingFiles()
	if err != nil {
		report.Add(Finding{
			Code:          "vcs_query_failed",
			Severity:      SeverityError,
			PlaintextPath: plaintextPath,
			Message:       fmt.Sprintf("VCS query failed: %v", err),
			Hint:          "ensure git or jj is installed and the repository is accessible",
		})
		return
	}

	checkFileVCS(report, plaintextPath, repoRoot, committed, pending, "plaintext_tracked", "plaintext_not_ignored")
	if keyFilePath != "" {
		checkFileVCS(report, keyFilePath, repoRoot, committed, pending, "key_tracked", "key_not_ignored")
	}
}

func checkFileVCS(report *Report, absPath, repoRoot string, committed, pending map[string]bool, errorCode, warningCode string) {
	rel, err := filepath.Rel(repoRoot, absPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		// File is outside the repository; pass.
		report.AddPass()
		return
	}
	rel = filepath.ToSlash(rel)

	if committed[rel] {
		sev := SeverityError
		msg := fmt.Sprintf("%s is tracked in version control history", absPath)
		hint := "remove it from version control history; adding to .gitignore after the fact does not remove it from history"
		if warningCode == "key_not_ignored" || errorCode == "key_tracked" {
			msg = fmt.Sprintf("Age private key file %s is tracked in version control history", absPath)
		}
		report.Add(Finding{
			Code:          errorCode,
			Severity:      sev,
			PlaintextPath: absPath,
			Message:       msg,
			Hint:          hint,
		})
		return
	}
	if pending[rel] {
		msg := fmt.Sprintf("%s is not ignored and would enter version control on the next commit", absPath)
		hint := "add it to .gitignore before committing"
		if warningCode == "key_not_ignored" {
			msg = fmt.Sprintf("Age private key file %s is not ignored and could be committed accidentally", absPath)
		}
		report.Add(Finding{
			Code:          warningCode,
			Severity:      SeverityWarning,
			PlaintextPath: absPath,
			Message:       msg,
			Hint:          hint,
		})
		return
	}
	report.AddPass()
}

// gitAdapter implements vcsAdapter for plain git repos.
type gitAdapter struct{ root string }

func (g *gitAdapter) committedFiles() (map[string]bool, error) {
	stdout, stderr, err := execx.ExecCommand("git", "-C", g.root, "ls-files", "-z", "--")
	if err != nil {
		return nil, fmt.Errorf("git ls-files: %v\n%s", err, strings.TrimRight(stderr, "\n"))
	}
	return splitNullLines(stdout), nil
}

func (g *gitAdapter) pendingFiles() (map[string]bool, error) {
	stdout, stderr, err := execx.ExecCommand("git", "-C", g.root, "ls-files", "-z", "--others", "--exclude-standard", "--")
	if err != nil {
		return nil, fmt.Errorf("git ls-files --others: %v\n%s", err, strings.TrimRight(stderr, "\n"))
	}
	return splitNullLines(stdout), nil
}

// jjAdapter implements vcsAdapter for jj repos (including colocated).
type jjAdapter struct{ root string }

func (j *jjAdapter) committedFiles() (map[string]bool, error) {
	// @- is the most recent real commit; files here are already in history.
	stdout, stderr, err := execx.ExecCommand("jj", "--no-pager", "file", "list", "-r", "@-")
	if err != nil {
		return nil, fmt.Errorf("jj file list -r @-: %v\n%s", err, strings.TrimRight(stderr, "\n"))
	}
	return splitNewlines(stdout), nil
}

func (j *jjAdapter) pendingFiles() (map[string]bool, error) {
	// @ is the working-copy snapshot; files here are one describe away from history.
	stdout, stderr, err := execx.ExecCommand("jj", "--no-pager", "file", "list", "-r", "@")
	if err != nil {
		return nil, fmt.Errorf("jj file list -r @: %v\n%s", err, strings.TrimRight(stderr, "\n"))
	}
	return splitNewlines(stdout), nil
}

func splitNullLines(s string) map[string]bool {
	m := make(map[string]bool)
	for _, p := range strings.Split(s, "\x00") {
		p = strings.TrimSpace(p)
		if p != "" {
			m[p] = true
		}
	}
	return m
}

func splitNewlines(s string) map[string]bool {
	m := make(map[string]bool)
	for _, p := range strings.Split(s, "\n") {
		p = strings.TrimSpace(p)
		if p != "" {
			m[p] = true
		}
	}
	return m
}
