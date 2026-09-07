package presentation

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/YewFence/YewSeal/internal/diff"
	"github.com/YewFence/YewSeal/internal/task"
	"github.com/stretchr/testify/require"
)

type failingWriter struct {
	calls int
	err   error
}

func (w *failingWriter) Write([]byte) (int, error) { w.calls++; return 0, w.err }

func TestOutputFaultsAreIndependentAndSticky(t *testing.T) {
	for _, short := range []bool{false, true} {
		t.Run(fmt.Sprint(short), func(t *testing.T) {
			failure := errors.New("unavailable")
			w := &failingWriter{err: failure}
			if short {
				w.err = nil
				failure = io.ErrShortWrite
			}
			var body bytes.Buffer
			out := New(&body, w, true)
			out.Warning("first")
			out.Warning("second")
			n, err := out.Write([]byte("secret content"))
			require.NoError(t, err)
			require.Equal(t, len("secret content"), n)
			require.Equal(t, "secret content", body.String())
			business := errors.New("file failed")
			err = out.Finish(business)
			require.ErrorIs(t, err, business)
			require.ErrorIs(t, err, failure)
			require.True(t, DiagnosticsFailed(err))
			out.Error(err)
			require.Equal(t, 1, w.calls)
		})
	}
}

func TestContentFailureAndFinalError(t *testing.T) {
	failure := errors.New("closed")
	w := &failingWriter{err: failure}
	var diagnostics bytes.Buffer
	out := New(w, &diagnostics, false)
	_, err := out.Write([]byte("content"))
	require.ErrorIs(t, err, failure)
	_, again := out.Write([]byte("more content"))
	require.Same(t, err, again)
	out.Error(out.Finish(err))
	require.Equal(t, 1, w.calls)
	require.Equal(t, 1, strings.Count(diagnostics.String(), "failed to write content"))
	require.NotContains(t, diagnostics.String(), "more content")
}

func TestConcurrentResultsAndSummaryAreNotDuplicated(t *testing.T) {
	for _, verbose := range []bool{false, true} {
		var body, diagnostics bytes.Buffer
		out := New(&body, &diagnostics, verbose)
		var wg sync.WaitGroup
		for i := range 32 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				out.FileCompleted(task.Result{SourceFile: fmt.Sprintf("%d.enc.yaml", i), TargetFile: fmt.Sprintf("%d.yaml", i), Status: task.Succeeded})
			}()
		}
		wg.Wait()
		out.FileCompleted(task.Result{SourceFile: "unavailable.enc.yaml", Status: task.Skipped})
		out.FileCompleted(task.Result{SourceFile: "broken.enc.yaml", Status: task.Failed, Error: errors.New("invalid")})
		out.BatchSummary(&task.Summary{TotalFiles: 34, SuccessCount: 32, SkippedCount: 1, FailedCount: 1}, "decrypted")
		require.NoError(t, out.Finish(nil))
		require.Empty(t, body.String())
		require.Equal(t, 1, strings.Count(diagnostics.String(), "SKIPPED"))
		require.Equal(t, 1, strings.Count(diagnostics.String(), "FAILED"))
		if verbose {
			require.Equal(t, 35, strings.Count(diagnostics.String(), "\n"))
			for i := range 32 {
				require.Contains(t, diagnostics.String(), fmt.Sprintf("SUCCEEDED %d.enc.yaml -> %d.yaml\n", i, i))
			}
		} else {
			require.Equal(t, 3, strings.Count(diagnostics.String(), "\n"))
		}
	}
}

func TestComparisonKeepsDistinctSkipReasons(t *testing.T) {
	var diagnostics bytes.Buffer
	out := New(nil, &diagnostics, false)
	out.ComparisonCompleted(diff.Outcome{PlaintextFile: "missing.yaml", EncryptedFile: "missing.enc.yaml", Skipped: diff.MissingPlaintext})
	out.ComparisonCompleted(diff.Outcome{PlaintextFile: "secret.yaml", EncryptedFile: "secret.enc.yaml", Skipped: diff.NoMatchingIdentity})
	out.ComparisonSummary(diff.Summary{MissingInputCount: 1, NoIdentityCount: 1, Results: make([]diff.Outcome, 2)})
	require.Contains(t, diagnostics.String(), "plaintext file is missing; not compared")
	require.Contains(t, diagnostics.String(), "encrypted content was not verified")
	require.Contains(t, diagnostics.String(), "1 missing input, 1 no matching identity")
	require.Contains(t, diagnostics.String(), "Comparison incomplete")
}
