package prompt

import (
	"bufio"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setPromptStdin(t *testing.T, file *os.File) {
	t.Helper()

	oldStdin := os.Stdin
	os.Stdin = file
	cachedStdin = nil
	cachedStdinReader = nil

	t.Cleanup(func() {
		os.Stdin = oldStdin
		cachedStdin = nil
		cachedStdinReader = nil
	})
}

func mockPromptInput(t *testing.T, input string) {
	t.Helper()

	r, w, err := os.Pipe()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = r.Close()
		_ = w.Close()
	})
	setPromptStdin(t, r)

	go func() {
		_, _ = w.Write([]byte(input))
		_ = w.Close()
	}()
}

func setEmptyPromptInput(t *testing.T) {
	t.Helper()

	file, err := os.CreateTemp(t.TempDir(), "empty-stdin-*")
	require.NoError(t, err)
	setPromptStdin(t, file)
	t.Cleanup(func() {
		_ = file.Close()
	})
}

func TestStdinReader_CachesPerStdin(t *testing.T) {
	r1, w1, err := os.Pipe()
	require.NoError(t, err)
	defer func() {
		_ = r1.Close()
		_ = w1.Close()
	}()

	setPromptStdin(t, r1)
	first := stdinReader()
	second := stdinReader()

	assert.Same(t, first, second)

	r2, w2, err := os.Pipe()
	require.NoError(t, err)
	defer func() {
		_ = r2.Close()
		_ = w2.Close()
	}()

	os.Stdin = r2
	third := stdinReader()

	assert.NotSame(t, first, third)
	assert.IsType(t, &bufio.Reader{}, third)
}

func TestPromptRequired_RePromptsUntilValue(t *testing.T) {
	mockPromptInput(t, "   \nactual-value\n")

	result, err := PromptRequired("Enter value")
	require.NoError(t, err)
	assert.Equal(t, "actual-value", result)
}

func TestPromptRequired_ReadErrorReturnsError(t *testing.T) {
	setEmptyPromptInput(t)

	result, err := PromptRequired("Enter value")
	assert.Empty(t, result)
	assert.Error(t, err)
}

func TestPromptWithDefault_ReadErrorReturnsDefault(t *testing.T) {
	setEmptyPromptInput(t)

	result := PromptWithDefault("Enter value", "fallback")
	assert.Equal(t, "fallback", result)
}

func TestPromptOptional_ReadErrorReturnsEmpty(t *testing.T) {
	setEmptyPromptInput(t)

	result := PromptOptional("Enter optional value")
	assert.Equal(t, "", result)
}

func TestPromptYesNo_ReadErrorUsesDefault(t *testing.T) {
	t.Run("default yes", func(t *testing.T) {
		setEmptyPromptInput(t)
		assert.True(t, PromptYesNo("Continue?", true))
	})

	t.Run("default no", func(t *testing.T) {
		setEmptyPromptInput(t)
		assert.False(t, PromptYesNo("Continue?", false))
	})
}
