package presentation

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/YewFence/YewSeal/internal/agekey"
	cleaner "github.com/YewFence/YewSeal/internal/clean"
	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/diff"
	"github.com/YewFence/YewSeal/internal/prompt"
	"github.com/YewFence/YewSeal/internal/seal"
	"github.com/YewFence/YewSeal/internal/sopsx"
	"github.com/YewFence/YewSeal/internal/task"
	"github.com/fatih/color"
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
	mu                 sync.Mutex
	content            io.Writer
	diagnostics        io.Writer
	contentErr         error
	diagnosticErr      error
	verbose            bool
	cwd                string
	diagnosticsPalette colorPalette
}

func New(content, diagnostics io.Writer, verbose bool) *Output {
	if content == nil {
		content = io.Discard
	}
	if diagnostics == nil {
		diagnostics = io.Discard
	}
	return &Output{content: content, diagnostics: diagnostics, verbose: verbose, diagnosticsPalette: diagnosticsPaletteFor(diagnostics)}
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

// statusf writes a diagnostic line with a colored status label followed by
// the formatted remainder.
func (o *Output) statusf(role *color.Color, label, format string, args ...any) {
	o.diagnostic(paint(role, label) + fmt.Sprintf(format, args...))
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
	o.statusf(o.diagnosticsPalette.warning, "WARNING", " %s\n", message)
}

// IdentityBundle reports redacted diagnostics collected while parsing identities.
func (o *Output) IdentityBundle(bundle agekey.IdentityBundle) {
	for _, warning := range bundle.Warnings() {
		o.Warning(warning)
	}
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
	muted := o.diagnosticsPalette.muted
	o.diagnostic(paint(muted, fmt.Sprintf("Selected %s from %s\n", countNoun(len(selection.FilePairs), "file pair"), countNoun(len(selection.ConfigFiles), "config file"))))
	for _, pair := range selection.FilePairs {
		o.diagnostic(paint(muted, fmt.Sprintf("  %s -> %s\n", o.path(pair.PlaintextPath), o.path(pair.EncryptedPath))))
	}
}

func (o *Output) FileCompleted(result task.Result) {
	pal := o.diagnosticsPalette
	if result.Warning != "" {
		o.Warning(o.path(result.SourceFile) + ": " + result.Warning)
	}
	switch result.Status {
	case task.Skipped:
		reason := sopsx.ErrNoMatchingIdentity.Error()
		switch result.Outcome {
		case task.OutcomeMissingPlaintext:
			reason = "plaintext file is missing; not encrypted"
		case task.OutcomeNoIdentity:
			reason = seal.ErrNoIdentity.Error()
		}
		o.statusf(pal.muted, "SKIPPED", " %s: %s\n", o.path(result.SourceFile), reason)
	case task.Failed:
		o.statusf(pal.error, "FAILED", " %s: %v\n", o.path(result.SourceFile), result.Error)
	case task.Succeeded:
		if o.verbose {
			verb := "SUCCEEDED"
			role := pal.success
			switch result.Outcome {
			case task.OutcomeEncrypted:
				verb = "ENCRYPTED"
			case task.OutcomeUnchanged:
				verb = "UNCHANGED"
				role = pal.muted
			}
			o.statusf(role, verb, " %s -> %s\n", o.path(result.SourceFile), o.path(result.TargetFile))
		}
	}
}

func (o *Output) BatchSummary(summary *task.Summary, action string) {
	if summary == nil {
		return
	}
	if action == "encrypted" {
		o.statusf(o.diagnosticsPalette.muted, "Summary (encrypted):", " %d encrypted, %d unchanged, %d missing plaintext, %d failed (%d selected)\n", summary.EncryptedCount, summary.UnchangedCount, summary.MissingPlaintextCount, summary.FailedCount, summary.TotalFiles)
		return
	}
	o.statusf(o.diagnosticsPalette.muted, fmt.Sprintf("Summary (%s):", action), " %d succeeded, %d skipped, %d failed (%d selected)\n", summary.SuccessCount, summary.SkippedCount, summary.FailedCount, summary.TotalFiles)
}

func (o *Output) ConfirmCleanDifference(prompts *prompt.Session, path string) (bool, error) {
	displayPath := o.path(path)
	pal := o.diagnosticsPalette
	message := paint(pal.warning, "Plaintext differs from encrypted content:") + " " + displayPath + "\n" +
		paint(pal.muted, "Hint:") + " run yews diff -- " + shellQuote(displayPath) + " to view the diff.\n"
	if _, err := o.writeDiagnostic([]byte(message)); err != nil {
		return false, err
	}
	confirmed := prompts.PromptYesNo("Delete the local plaintext anyway?", false)
	if err := prompts.Err(); err != nil {
		return false, err
	}
	return confirmed, nil
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func (o *Output) CleanCompleted(result cleaner.Result) {
	pal := o.diagnosticsPalette
	path := o.path(result.PlaintextPath)
	switch result.Status {
	case cleaner.StatusRemoved:
		o.statusf(pal.success, "REMOVED", " %s\n", path)
	case cleaner.StatusAlreadyAbsent:
		if o.verbose {
			o.statusf(pal.muted, "ALREADY ABSENT", " %s\n", path)
		}
	case cleaner.StatusRetained:
		o.statusf(pal.warning, "RETAINED", " %s: plaintext differs from encrypted content\n", path)
	case cleaner.StatusFailed:
		o.statusf(pal.error, "FAILED", " %s: %v\n", path, result.Error)
	}
}

func (o *Output) CleanSummary(summary cleaner.Summary) {
	o.statusf(o.diagnosticsPalette.muted, "Summary (cleaned):", " %d removed, %d already absent, %d retained, %d failed (%d selected)\n", summary.RemovedCount, summary.AlreadyAbsentCount, summary.RetainedCount, summary.FailedCount, len(summary.Results))
}

func (o *Output) ComparisonCompleted(result diff.Outcome) {
	pal := o.diagnosticsPalette
	paths := o.path(result.PlaintextFile) + " -> " + o.path(result.EncryptedFile)
	if result.Error != nil {
		o.statusf(pal.error, "FAILED", " %s: %v\n", paths, result.Error)
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
		o.statusf(pal.muted, "SKIPPED", " %s: %s\n", paths, reason)
	} else if o.verbose {
		o.statusf(pal.success, "COMPARED", " %s\n", paths)
	}
}

func (o *Output) ComparisonSummary(summary diff.Summary) {
	if summary.NoIdentityCount > 0 || summary.FailedCount > 0 {
		o.diagnostic(paint(o.diagnosticsPalette.warning, "Comparison incomplete: some files with both inputs present could not be compared.\n"))
	}
	o.statusf(o.diagnosticsPalette.muted, "Summary:", " %d compared, %d skipped (%d missing input, %d no matching identity), %d failed (%d selected)\n", summary.ComparedCount, summary.MissingInputCount+summary.NoIdentityCount, summary.MissingInputCount, summary.NoIdentityCount, summary.FailedCount, len(summary.Results))
}

func (o *Output) Edited(path string, changed bool) {
	pal := o.diagnosticsPalette
	if changed {
		o.statusf(pal.success, "Updated", " %s\n", o.path(path))
	} else {
		o.statusf(pal.muted, "Unchanged", " %s\n", o.path(path))
	}
}

func (o *Output) Initialized(files int, sops, gitignore bool) {
	paths := ".yewseal.toml, .age/keys.txt"
	if gitignore {
		paths += ", .gitignore"
	}
	if sops {
		paths += ", .sops.yaml"
	}
	o.statusf(o.diagnosticsPalette.success, "Initialized", " %s: %s\n", countNoun(files, "file mapping"), paths)
}

func (o *Output) InitKept() {
	o.statusf(o.diagnosticsPalette.muted, "Unchanged:", " existing project configuration kept\n")
}

// Error prints the final command error only when the diagnostic stream is usable.
func (o *Output) Error(err error) {
	if err != nil && !DiagnosticsFailed(err) {
		o.statusf(o.diagnosticsPalette.error, "Error:", " %v\n", err)
	}
}
