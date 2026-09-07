package project

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/YewFence/YewSeal/internal/presentation"
	"github.com/YewFence/YewSeal/internal/prompt"
	"github.com/stretchr/testify/require"
)

func testInitializer() *initializer {
	return &initializer{output: presentation.New(io.Discard, io.Discard, false), prompts: prompt.New(os.Stdin, io.Discard)}
}

type brokenDiagnostics struct{ calls int }

func (w *brokenDiagnostics) Write([]byte) (int, error) { w.calls++; return 0, io.ErrClosedPipe }

type unreadableInput struct{}

func (unreadableInput) Read([]byte) (int, error) { panic("must not wait for invisible question") }

func TestInitDiagnosticFailureStillCompletesForcedRebuild(t *testing.T) {
	t.Chdir(t.TempDir())
	w := &brokenDiagnostics{}
	var body bytes.Buffer
	out := presentation.New(&body, w, false)
	err := InitProject(true, "config.yaml", "", "", false, false, out, out.Prompts(unreadableInput{}))
	require.ErrorIs(t, err, io.ErrClosedPipe)
	require.True(t, presentation.DiagnosticsFailed(err))
	require.Equal(t, 1, w.calls)
	require.Empty(t, body.String())
	for _, file := range []string{".age/keys.txt", ".yewseal.toml", ".gitignore", ".sops.yaml"} {
		require.FileExists(t, file)
	}
}

func TestInitPromptFailureStopsBeforeWrites(t *testing.T) {
	t.Chdir(t.TempDir())
	w := &brokenDiagnostics{}
	out := presentation.New(nil, w, false)
	err := InitProject(false, "", "", "", false, false, out, out.Prompts(unreadableInput{}))
	require.ErrorIs(t, err, io.ErrClosedPipe)
	require.True(t, presentation.DiagnosticsFailed(err))
	out.Error(err)
	require.Equal(t, 1, w.calls)
	for _, file := range []string{".age", ".yewseal.toml", ".gitignore", ".sops.yaml"} {
		_, err := os.Stat(file)
		require.True(t, errors.Is(err, os.ErrNotExist))
	}
}

func TestInitStreamsAndCompletion(t *testing.T) {
	t.Chdir(t.TempDir())
	var body, diagnostics bytes.Buffer
	out := presentation.New(&body, &diagnostics, false)
	err := InitProject(false, "", "", "", false, false, out, out.Prompts(strings.NewReader("config.yaml\n\nn\nn\nn\n")))
	require.NoError(t, err)
	require.Empty(t, body.String())
	require.Contains(t, diagnostics.String(), "Enter plaintext config file name")
	require.Equal(t, 1, strings.Count(diagnostics.String(), "Initialized"))
	require.NotContains(t, diagnostics.String(), "Next steps")
}

func testInitProject(force bool, input, output, format string, example, skip bool) error {
	i := testInitializer()
	return InitProject(force, input, output, format, example, skip, i.output, i.prompts)
}
