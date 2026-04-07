package prompt

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockStdin replaces os.Stdin with a pipe containing the given input
// Returns a cleanup function that restores the original stdin
func mockStdin(input string) func() {
	r, w, _ := os.Pipe()
	oldStdin := os.Stdin
	os.Stdin = r

	go func() {
		w.Write([]byte(input))
		w.Close()
	}()

	return func() {
		os.Stdin = oldStdin
	}
}

func TestPromptWithDefault_UserInput(t *testing.T) {
	restore := mockStdin("custom_value\n")
	defer restore()

	result := PromptWithDefault("Enter value", "default")
	assert.Equal(t, "custom_value", result)
}

func TestPromptWithDefault_EmptyInput(t *testing.T) {
	restore := mockStdin("\n")
	defer restore()

	result := PromptWithDefault("Enter value", "default")
	assert.Equal(t, "default", result)
}

func TestPromptWithDefault_WhitespaceInput(t *testing.T) {
	restore := mockStdin("   \n")
	defer restore()

	result := PromptWithDefault("Enter value", "default")
	assert.Equal(t, "default", result)
}

func TestPromptOptional_UserInput(t *testing.T) {
	restore := mockStdin(" custom_value \n")
	defer restore()

	result := PromptOptional("Enter value")
	assert.Equal(t, "custom_value", result)
}

func TestPromptOptional_EmptyInput(t *testing.T) {
	restore := mockStdin("\n")
	defer restore()

	result := PromptOptional("Enter value")
	assert.Equal(t, "", result)
}

func TestPromptYesNo_Yes(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"y\n", true},
		{"Y\n", true},
		{"yes\n", true},
		{"YES\n", true},
		{"Yes\n", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			restore := mockStdin(tt.input)
			defer restore()

			result := PromptYesNo("Continue?", false)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPromptYesNo_No(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"n\n", false},
		{"N\n", false},
		{"no\n", false},
		{"NO\n", false},
		{"No\n", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			restore := mockStdin(tt.input)
			defer restore()

			result := PromptYesNo("Continue?", true)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPromptYesNo_DefaultYes(t *testing.T) {
	restore := mockStdin("\n")
	defer restore()

	result := PromptYesNo("Continue?", true)
	assert.True(t, result)
}

func TestPromptYesNo_DefaultNo(t *testing.T) {
	restore := mockStdin("\n")
	defer restore()

	result := PromptYesNo("Continue?", false)
	assert.False(t, result)
}

func TestPromptYesNo_InvalidInputUsesDefault(t *testing.T) {
	restore := mockStdin("maybe\n")
	defer restore()

	// Invalid input should not match "y" or "yes", so it returns false
	result := PromptYesNo("Continue?", true)
	assert.False(t, result)
}

func TestPromptYesNoConditional_FlagSet(t *testing.T) {
	// When flag is set, should return defaultValue without reading stdin
	result := PromptYesNoConditional(true, true, "Continue?")
	assert.True(t, result)

	result = PromptYesNoConditional(true, false, "Continue?")
	assert.False(t, result)
}

func TestPromptYesNoConditional_FlagNotSet(t *testing.T) {
	restore := mockStdin("y\n")
	defer restore()

	result := PromptYesNoConditional(false, false, "Continue?")
	assert.True(t, result)
}
