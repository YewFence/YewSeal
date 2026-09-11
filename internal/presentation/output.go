package presentation

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/diff"
	"github.com/YewFence/YewSeal/internal/prompt"
	"github.com/YewFence/YewSeal/internal/sopsx"
	"github.com/YewFence/YewSeal/internal/task"
)

// OutputError identifies delivery failure independently of file processing results.
type OutputError struct {
	Channel string
	Err     error
}

func (e *OutputError) Error() string { return fmt.Sprintf("failed to write %s: %v", e.Channel, e.Err) }
func (e *OutputError) Unwrap() error { return e.Err }

// Output owns one command's streams. Diagnostics are best-effort during execution;
// Finish reports their failure without cancelling file operations.
type Output struct {
	mu            sync.Mutex
	content       io.Writer
	diagnostics   io.Writer
	contentErr    error
	diagnosticErr error
	verbose       bool
	cwd           string
}

func New(content, diagnostics io.Writer, verbose bool) *Output {
	if content == nil {
		content = io.Discard
	}
	if diagnostics == nil {
		diagnostics = io.Discard
	}
	return &Output{content: content, diagnostics: diagnostics, verbose: verbose}
}

func OrDiscard(out *Output) *Output {
	if out == nil {
		return New(io.Discard, io.Discard, false)
	}
	return out
}

func (o *Output) SetDirectory(cwd string) { o.cwd = cwd }

func (o *Output) path(path string) string {
	if o.cwd != "" {
		return config.DisplayPath(o.cwd, path)
	}
	return path
}

// Write is the content-only interface used by plaintext, diff and plan renderers.
func (o *Output) Write(p []byte) (int, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.contentErr != nil {
		return 0, o.contentErr
	}
	n, err := checkedWrite(o.content, p)
	if err != nil {
		o.contentErr = &OutputError{Channel: "content", Err: err}
	}
	return n, o.contentErr
}

func checkedWrite(w io.Writer, p []byte) (int, error) {
	n, err := w.Write(p)
	if err == nil && n != len(p) {
		err = io.ErrShortWrite
	}
	return n, err
}

func (o *Output) diagnostic(text string) {
	_, _ = o.writeDiagnostic([]byte(text))
}

func (o *Output) writeDiagnostic(p []byte) (int, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.diagnosticErr != nil {
		return 0, o.diagnosticErr
	}
	n, err := checkedWrite(o.diagnostics, p)
	if err != nil {
		o.diagnosticErr = &OutputError{Channel: "diagnostics", Err: err}
	}
	return n, o.diagnosticErr
}

type promptWriter struct{ output *Output }

func (w promptWriter) Write(p []byte) (int, error) { return w.output.writeDiagnostic(p) }

// Prompts shares diagnostic failure state, but propagates it to stop interaction.
func (o *Output) Prompts(input io.Reader) *prompt.Session {
	return prompt.New(input, promptWriter{output: o})
}

func (o *Output) Finish(businessErr error) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	contentErr, diagnosticErr := o.contentErr, o.diagnosticErr
	if errors.Is(businessErr, contentErr) {
		contentErr = nil
	}
	if errors.Is(businessErr, diagnosticErr) {
		diagnosticErr = nil
	}
	return errors.Join(businessErr, contentErr, diagnosticErr)
}

func DiagnosticsFailed(err error) bool {
	if err == nil {
		return false
	}
	if e, ok := err.(*OutputError); ok && e.Channel == "diagnostics" {
		return true
	}
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		for _, child := range joined.Unwrap() {
			if DiagnosticsFailed(child) {
				return true
			}
		}
	}
	return DiagnosticsFailed(errors.Unwrap(err))
}

func (o *Output) Warning(message string) {
	message = strings.TrimPrefix(message, "warning: ")
	o.diagnostic("WARNING " + message + "\n")
}

func (o *Output) Selection(selection config.ResolvedSelection) {
	for _, pair := range selection.FilePairs {
		if pair.RecipientWarning != "" {
			o.Warning(pair.RecipientWarning)
		}
	}
	if !o.verbose {
		return
	}
	o.diagnostic(fmt.Sprintf("Selected %d file pairs from %d config files\n", len(selection.FilePairs), len(selection.ConfigFiles)))
	for _, pair := range selection.FilePairs {
		o.diagnostic(fmt.Sprintf("  %s -> %s\n", o.path(pair.PlaintextPath), o.path(pair.EncryptedPath)))
	}
}

func (o *Output) FileCompleted(result task.Result) {
	switch result.Status {
	case task.Skipped:
		o.diagnostic(fmt.Sprintf("SKIPPED %s: %s\n", o.path(result.SourceFile), sopsx.ErrNoMatchingIdentity))
	case task.Failed:
		o.diagnostic(fmt.Sprintf("FAILED %s: %v\n", o.path(result.SourceFile), result.Error))
	case task.Succeeded:
		if o.verbose {
			o.diagnostic(fmt.Sprintf("SUCCEEDED %s -> %s\n", o.path(result.SourceFile), o.path(result.TargetFile)))
		}
	}
}

func (o *Output) BatchSummary(summary *task.Summary, action string) {
	if summary == nil {
		return
	}
	o.diagnostic(fmt.Sprintf("Summary (%s): %d succeeded, %d skipped, %d failed (%d selected)\n", action, summary.SuccessCount, summary.SkippedCount, summary.FailedCount, summary.TotalFiles))
}

func (o *Output) ComparisonCompleted(result diff.Outcome) {
	paths := o.path(result.PlaintextFile) + " -> " + o.path(result.EncryptedFile)
	if result.Error != nil {
		o.diagnostic(fmt.Sprintf("FAILED %s: %v\n", paths, result.Error))
		return
	}
	var reason string
	switch result.Skipped {
	case diff.MissingPlaintext:
		reason = "plaintext file is missing; not compared"
	case diff.MissingCiphertext:
		reason = "encrypted file is missing; not compared"
	case diff.MissingBothInputs:
		reason = "plaintext and encrypted files are missing; not compared"
	case diff.NoMatchingIdentity:
		reason = sopsx.ErrNoMatchingIdentity.Error()
	}
	if reason != "" {
		o.diagnostic(fmt.Sprintf("SKIPPED %s: %s\n", paths, reason))
	} else if o.verbose {
		o.diagnostic("COMPARED " + paths + "\n")
	}
}

func (o *Output) ComparisonSummary(summary diff.Summary) {
	if summary.NoIdentityCount > 0 || summary.FailedCount > 0 {
		o.diagnostic("Comparison incomplete: some files with both inputs present could not be compared.\n")
	}
	o.diagnostic(fmt.Sprintf("Summary: %d compared, %d skipped (%d missing input, %d no matching identity), %d failed (%d selected)\n", summary.ComparedCount, summary.MissingInputCount+summary.NoIdentityCount, summary.MissingInputCount, summary.NoIdentityCount, summary.FailedCount, len(summary.Results)))
}

func (o *Output) Edited(path string, changed bool) {
	if changed {
		o.diagnostic("Updated " + o.path(path) + "\n")
	} else {
		o.diagnostic("Unchanged " + o.path(path) + "\n")
	}
}

func (o *Output) Initialized(files int, sops bool) {
	paths := ".yewseal.toml, .age/keys.txt, .gitignore"
	if sops {
		paths += ", .sops.yaml"
	}
	o.diagnostic(fmt.Sprintf("Initialized %d file mappings: %s\n", files, paths))
}

func (o *Output) InitKept() { o.diagnostic("Unchanged: existing project configuration kept\n") }

// Error prints the final command error only when the diagnostic stream is usable.
func (o *Output) Error(err error) {
	if err != nil && !DiagnosticsFailed(err) {
		o.diagnostic("Error: " + err.Error() + "\n")
	}
}
