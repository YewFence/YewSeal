package diff

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/YewFence/YewSeal/internal/seal"
	"github.com/sergi/go-diff/diffmatchpatch"
)

// DiffResult describes a plaintext-vs-decrypted comparison.
type DiffResult struct {
	Diff      string
	Different bool
}

// DiffPlaintextAgainstEncrypted compares an existing plaintext file with the
// decrypted encrypted file without writing decrypted content to disk.
type Options struct {
	PlaintextFile  string
	EncryptedFile  string
	KeyFile        string
	FormatOverride string
	Verbose        bool
}

func PlaintextAgainstEncrypted(opts Options) (DiffResult, error) {
	currentData, err := os.ReadFile(opts.PlaintextFile)
	if err != nil {
		if os.IsNotExist(err) {
			return DiffResult{}, fmt.Errorf("plaintext file %s does not exist", opts.PlaintextFile)
		}
		return DiffResult{}, fmt.Errorf("failed to read plaintext file: %w", err)
	}

	decryptedData, err := seal.DecryptToBytes(seal.DecryptBytesOptions{
		InputFile:      opts.EncryptedFile,
		OutputFile:     opts.PlaintextFile,
		KeyFile:        opts.KeyFile,
		FormatOverride: opts.FormatOverride,
		Verbose:        opts.Verbose,
	})
	if err != nil {
		return DiffResult{}, err
	}

	if bytes.Equal(currentData, decryptedData) {
		return DiffResult{}, nil
	}

	diff := UnifiedDiff(opts.PlaintextFile, opts.EncryptedFile+" (decrypted)", currentData, decryptedData)
	return DiffResult{Diff: diff, Different: true}, nil
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
