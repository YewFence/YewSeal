package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractPublicKey_Valid(t *testing.T) {
	output := "# public key: age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p"
	result := ExtractPublicKey(output)
	assert.Equal(t, "age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p", result)
}

func TestExtractPublicKey_MultiLine(t *testing.T) {
	output := `# created: 2024-01-01T00:00:00Z
# public key: age1abc123xyz456
AGE-SECRET-KEY-1QQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQ`

	result := ExtractPublicKey(output)
	assert.Equal(t, "age1abc123xyz456", result)
}

func TestExtractPublicKey_WithWhitespace(t *testing.T) {
	output := "  # public key: age1testkey  "
	result := ExtractPublicKey(output)
	assert.Equal(t, "age1testkey", result)
}

func TestExtractPublicKey_NoKey(t *testing.T) {
	output := `# created: 2024-01-01T00:00:00Z
AGE-SECRET-KEY-1QQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQ`

	result := ExtractPublicKey(output)
	assert.Empty(t, result)
}

func TestExtractPublicKey_Empty(t *testing.T) {
	result := ExtractPublicKey("")
	assert.Empty(t, result)
}

func TestExtractPublicKey_OnlyComments(t *testing.T) {
	output := `# some comment
# another comment
# not a public key line`

	result := ExtractPublicKey(output)
	assert.Empty(t, result)
}

func TestExtractPublicKey_MalformedPrefix(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{
			name:   "wrong prefix",
			input:  "# publickey: age1test",
			expect: "",
		},
		{
			name:   "missing colon",
			input:  "# public key age1test",
			expect: "",
		},
		{
			name:   "correct prefix",
			input:  "# public key: age1correct",
			expect: "age1correct",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractPublicKey(tt.input)
			assert.Equal(t, tt.expect, result)
		})
	}
}
