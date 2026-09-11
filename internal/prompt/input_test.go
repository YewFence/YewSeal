package prompt

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSessionPreservesBufferedAnswers(t *testing.T) {
	var out bytes.Buffer
	s := New(strings.NewReader("custom\n\n  optional  \n\nrequired\nYES\nno\n"), &out)
	require.Equal(t, "custom", s.PromptWithDefault("First", "fallback"))
	require.Equal(t, "fallback", s.PromptWithDefault("Second", "fallback"))
	require.Equal(t, "optional", s.PromptOptional("Optional"))
	value, err := s.PromptRequired("Required")
	require.NoError(t, err)
	require.Equal(t, "required", value)
	require.True(t, s.PromptYesNo("Continue?", false))
	require.False(t, s.PromptYesNo("Continue?", true))
	require.NoError(t, s.Err())
	require.Contains(t, out.String(), "First [fallback]: ")
}

func TestConfirmationAnswers(t *testing.T) {
	for _, tc := range []struct {
		input          string
		fallback, want bool
	}{
		{"y\n", false, true}, {"Y\n", false, true}, {"yes\n", false, true},
		{"n\n", true, false}, {"NO\n", true, false}, {"maybe\n", true, false},
		{"\n", true, true}, {"  \n", false, false}, {"", true, false},
	} {
		s := New(strings.NewReader(tc.input), io.Discard)
		require.Equal(t, tc.want, s.PromptYesNo("Continue?", tc.fallback))
		if tc.input == "" {
			require.ErrorIs(t, s.Err(), io.EOF)
		}
	}
}

func TestReadDefaultsAndRequiredError(t *testing.T) {
	s := New(strings.NewReader(""), io.Discard)
	require.Empty(t, s.PromptWithDefault("Value", "fallback"))
	require.Empty(t, s.PromptOptional("Optional"))
	_, err := s.PromptRequired("Required")
	require.ErrorIs(t, err, io.EOF)
	require.ErrorIs(t, s.Err(), io.EOF)
}

type unavailableInput struct{}

func (unavailableInput) Read([]byte) (int, error) { panic("must not read after prompt failure") }

type failedWriter struct {
	err   error
	calls int
}

func (w *failedWriter) Write([]byte) (int, error) { w.calls++; return 0, w.err }

func TestPromptFailureDoesNotReadOrChooseDefault(t *testing.T) {
	for _, failure := range []error{errors.New("closed"), nil} {
		w := &failedWriter{err: failure}
		s := New(unavailableInput{}, w)
		require.False(t, s.PromptYesNo("Destroy?", true))
		require.Empty(t, s.PromptWithDefault("Value", "fallback"))
		require.Empty(t, s.PromptOptional("Value"))
		_, err := s.PromptRequired("Value")
		require.Error(t, err)
		require.Error(t, s.Err())
		require.Equal(t, 1, w.calls)
		if failure == nil {
			require.ErrorIs(t, s.Err(), io.ErrShortWrite)
		}
	}
}

func TestConditionalQuestionDoesNotTouchStreams(t *testing.T) {
	w := &failedWriter{err: errors.New("closed")}
	s := New(unavailableInput{}, w)
	require.True(t, s.PromptYesNoConditional(true, true, "Continue?"))
	require.False(t, s.PromptYesNoConditional(true, false, "Continue?"))
	require.Zero(t, w.calls)
	require.NoError(t, s.Err())
}
