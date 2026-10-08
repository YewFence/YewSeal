package verify

import (
	"fmt"
	"path/filepath"

	"github.com/YewFence/YewSeal/internal/vcs"
)

type vcsSubject struct {
	trackedCode    string
	notIgnoredCode string
	noun           string
}

var (
	plaintextSubject = vcsSubject{trackedCode: "plaintext_tracked", notIgnoredCode: "plaintext_not_ignored", noun: "plaintext file"}
	keySubject       = vcsSubject{trackedCode: "key_tracked", notIgnoredCode: "key_not_ignored", noun: "Age private key file"}
)

func classifyVCS(snapshot *vcs.Snapshot, report *Report, absPath string, subject vcsSubject, finding Finding) {
	classified := classifyVCSPath(snapshot, report, absPath, subject, finding)
	target, err := filepath.EvalSymlinks(absPath)
	if err == nil && target != absPath {
		classified = classifyVCSPath(snapshot, report, target, subject, finding) || classified
	}
	if !classified {
		report.AddPass()
	}
}

func classifyVCSPath(snapshot *vcs.Snapshot, report *Report, path string, subject vcsSubject, finding Finding) bool {
	relative, err := filepath.Rel(snapshot.Root(), path)
	if err != nil {
		return false
	}
	switch snapshot.Status(path) {
	case vcs.History:
		finding.Code = subject.trackedCode
		finding.Severity = SeverityError
		finding.Message = fmt.Sprintf("%s %s is tracked by version control", subject.noun, filepath.ToSlash(relative))
		finding.Hint = "remove it from version control and review repository history; ignoring it now does not remove it from history" + hintAgentHandoff
	case vcs.Pending:
		finding.Code = subject.notIgnoredCode
		finding.Severity = SeverityError
		finding.Message = fmt.Sprintf("%s %s is not ignored and would enter history with the next commit", subject.noun, filepath.ToSlash(relative))
		finding.Hint = "add it to .gitignore" + hintAgentAutoOK
	default:
		return false
	}
	report.Add(finding)
	return true
}
