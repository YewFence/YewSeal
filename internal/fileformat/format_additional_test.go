package fileformat

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNeedsConversion(t *testing.T) {
	assert.True(t, NeedsConversion(TOML))
	assert.False(t, NeedsConversion(YAML))
	assert.False(t, NeedsConversion(JSON))
	assert.False(t, NeedsConversion(ENV))
	assert.False(t, NeedsConversion(INI))
	assert.False(t, NeedsConversion(Unknown))
}

func TestSOPSFormat(t *testing.T) {
	tests := []struct {
		name   string
		format Format
		want   string
	}{
		{name: "toml", format: TOML, want: "yaml"},
		{name: "yaml", format: YAML, want: "yaml"},
		{name: "json", format: JSON, want: "json"},
		{name: "env", format: ENV, want: "dotenv"},
		{name: "ini", format: INI, want: "ini"},
		{name: "unknown", format: Unknown, want: "yaml"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, SOPSFormat(tt.format))
		})
	}
}

func TestIsSopsNative(t *testing.T) {
	assert.False(t, IsSopsNative(TOML))
	assert.True(t, IsSopsNative(YAML))
	assert.True(t, IsSopsNative(JSON))
	assert.True(t, IsSopsNative(ENV))
	assert.True(t, IsSopsNative(INI))
	assert.False(t, IsSopsNative(Unknown))
}
