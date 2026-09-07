package diff

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/YewFence/YewSeal/internal/agekey"
	"github.com/YewFence/YewSeal/internal/seal"
	"github.com/YewFence/YewSeal/internal/sopsx"
	"github.com/fatih/color"
	"github.com/sergi/go-diff/diffmatchpatch"
)

// DiffResult describes a plaintext-vs-decrypted comparison.
type DiffResult struct {
	Diff      string
	Different bool
	Skipped   SkipReason
}

// DiffPlaintextAgainstEncrypted compares an existing plaintext file with the
// decrypted encrypted file without writing decrypted content to disk.
type Options struct {
	PlaintextFile  string
	EncryptedFile  string
	PlaintextLabel string
	EncryptedLabel string
	IdentityBundle agekey.IdentityBundle
	FormatOverride string
	Verbose        bool
	Diagnostics    io.Writer
}

func PlaintextAgainstEncrypted(opts Options) (DiffResult, error) {
	missing, err := missingInputs(opts.PlaintextFile, opts.EncryptedFile)
	if err != nil || missing != "" {
		return DiffResult{Skipped: missing}, err
	}
	diagnostics := opts.Diagnostics
	if diagnostics == nil {
		diagnostics = io.Discard
	}
	decryptedData, err := seal.DecryptToBytes(seal.DecryptBytesOptions{
		InputFile:      opts.EncryptedFile,
		OutputFile:     opts.PlaintextFile,
		IdentityBundle: opts.IdentityBundle,
		FormatOverride: opts.FormatOverride,
		Verbose:        opts.Verbose,
		Output:         diagnostics,
	})
	if errors.Is(err, sopsx.ErrNoMatchingIdentity) {
		return DiffResult{Skipped: NoMatchingIdentity}, nil
	}
	if err != nil {
		return DiffResult{}, err
	}
	currentData, err := os.ReadFile(opts.PlaintextFile)
	if err != nil {
		if os.IsNotExist(err) {
			return DiffResult{Skipped: MissingPlaintext}, nil
		}
		return DiffResult{}, fmt.Errorf("failed to read plaintext file: %w", err)
	}

	if bytes.Equal(currentData, decryptedData) {
		return DiffResult{}, nil
	}

	from, to := opts.PlaintextLabel, opts.EncryptedLabel
	if from == "" {
		from = opts.PlaintextFile
	}
	if to == "" {
		to = opts.EncryptedFile
	}
	diff := UnifiedDiff(from, to+" (decrypted)", currentData, decryptedData)
	return DiffResult{Diff: diff, Different: true}, nil
}

func missingInputs(plaintext, encrypted string) (SkipReason, error) {
	var missingPlaintext, missingCiphertext bool
	for i, path := range []string{plaintext, encrypted} {
		info, err := os.Stat(path)
		if os.IsNotExist(err) {
			if i == 0 {
				missingPlaintext = true
			} else {
				missingCiphertext = true
			}
			continue
		}
		if err != nil {
			return "", fmt.Errorf("failed to inspect comparison input %s: %w", path, err)
		}
		if !info.Mode().IsRegular() {
			return "", fmt.Errorf("comparison input %s is not a regular file", path)
		}
	}
	switch {
	case missingPlaintext && missingCiphertext:
		return MissingBothInputs, nil
	case missingPlaintext:
		return MissingPlaintext, nil
	case missingCiphertext:
		return MissingCiphertext, nil
	default:
		return "", nil
	}
}

// UnifiedDiff returns a unified diff-like text using a line-oriented go-diff comparison.
func UnifiedDiff(fromName, toName string, fromData, toData []byte) string {
	from := string(fromData)
	to := string(toData)

	dmp := diffmatchpatch.New()
	fromChars, toChars, lineArray := dmp.DiffLinesToChars(from, to)
	diffs := dmp.DiffMain(fromChars, toChars, false)
	diffs = dmp.DiffCharsToLines(diffs, lineArray)

	var out strings.Builder
	fmt.Fprintf(&out, "--- %s\n", fromName)
	fmt.Fprintf(&out, "+++ %s\n", toName)
	out.WriteString("@@\n")

	for _, diff := range diffs {
		prefix := " "
		switch diff.Type {
		case diffmatchpatch.DiffDelete:
			prefix = "-"
		case diffmatchpatch.DiffInsert:
			prefix = "+"
		}
		writePrefixedLines(&out, prefix, diff.Text)
	}

	return out.String()
}

// HighlightUnifiedDiff applies terminal colors to a unified diff without changing its text shape.
func HighlightUnifiedDiff(unifiedDiff string, enabled bool) string {
	if !enabled || unifiedDiff == "" {
		return unifiedDiff
	}

	headerColor := color.New(color.FgCyan, color.Bold)
	hunkColor := color.New(color.FgMagenta)
	deleteColor := color.New(color.FgRed)
	insertColor := color.New(color.FgGreen)
	for _, c := range []*color.Color{headerColor, hunkColor, deleteColor, insertColor} {
		c.EnableColor()
	}

	var out strings.Builder
	lines := strings.SplitAfter(unifiedDiff, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		switch {
		case strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "+++ "):
			out.WriteString(headerColor.Sprint(line))
		case strings.HasPrefix(line, "@@"):
			out.WriteString(hunkColor.Sprint(line))
		case strings.HasPrefix(line, "-"):
			out.WriteString(deleteColor.Sprint(line))
		case strings.HasPrefix(line, "+"):
			out.WriteString(insertColor.Sprint(line))
		default:
			out.WriteString(line)
		}
	}
	return out.String()
}

func writePrefixedLines(out *strings.Builder, prefix, text string) {
	if text == "" {
		return
	}

	lines := strings.SplitAfter(text, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		out.WriteString(prefix)
		out.WriteString(line)
		if !strings.HasSuffix(line, "\n") {
			out.WriteByte('\n')
		}
	}
}
