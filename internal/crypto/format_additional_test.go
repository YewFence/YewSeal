package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNeedsConversion(t *testing.T) {
	assert.True(t, NeedsConversion(FormatTOML))
	assert.False(t, NeedsConversion(FormatYAML))
	assert.False(t, NeedsConversion(FormatJSON))
	assert.False(t, NeedsConversion(FormatENV))
	assert.False(t, NeedsConversion(FormatINI))
	assert.False(t, NeedsConversion(FormatUnknown))
}

func TestGetSopsType(t *testing.T) {
	tests := []struct {
		name   string
		format FileFormat
		want   string
	}{
		{name: "toml", format: FormatTOML, want: "yaml"},
		{name: "yaml", format: FormatYAML, want: "yaml"},
		{name: "json", format: FormatJSON, want: "json"},
		{name: "env", format: FormatENV, want: "dotenv"},
		{name: "ini", format: FormatINI, want: "ini"},
		{name: "unknown", format: FormatUnknown, want: "yaml"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, GetSopsType(tt.format))
		})
	}
}

func TestIsSopsNativeFormat(t *testing.T) {
	assert.False(t, IsSopsNativeFormat(FormatTOML))
	assert.True(t, IsSopsNativeFormat(FormatYAML))
	assert.True(t, IsSopsNativeFormat(FormatJSON))
	assert.True(t, IsSopsNativeFormat(FormatENV))
	assert.True(t, IsSopsNativeFormat(FormatINI))
	assert.False(t, IsSopsNativeFormat(FormatUnknown))
}
