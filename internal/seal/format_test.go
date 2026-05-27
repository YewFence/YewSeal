package seal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		filename string
		want     format
	}{
		{"config.toml", formatTOML},
		{"config.yaml", formatYAML},
		{"config.yml", formatYAML},
		{"config.json", formatJSON},
		{"config.env", formatENV},
		{"config.ini", formatINI},
		{".dev.vars", formatUnknown},
		{"secrets.vars", formatUnknown},
		{"wrangler", formatUnknown},
		{"config.xml", formatUnknown},
		{"config.TOML", formatTOML},
		{"config.YAML", formatYAML},
		{"config.JSON", formatJSON},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			assert.Equal(t, tt.want, detectFormat(tt.filename))
		})
	}
}

func TestParseFormat(t *testing.T) {
	tests := []struct {
		input string
		want  format
	}{
		{"toml", formatTOML},
		{"yaml", formatYAML},
		{"yml", formatYAML},
		{"json", formatJSON},
		{"env", formatENV},
		{"dotenv", formatENV},
		{"ini", formatINI},
		{"TOML", formatTOML},
		{"YAML", formatYAML},
		{"ENV", formatENV},
		{"Dotenv", formatENV},
		{" env ", formatENV},
		{"\tyaml\t", formatYAML},
		{"", formatUnknown},
		{"xml", formatUnknown},
		{"vars", formatUnknown},
		{"binary", formatUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, parseFormat(tt.input))
		})
	}
}

func TestNormalizeFormatOverride(t *testing.T) {
	value, ok := NormalizeFormatOverride("dotenv")
	assert.True(t, ok)
	assert.Equal(t, "env", value)

	_, ok = NormalizeFormatOverride("xml")
	assert.False(t, ok)
}

func TestNeedsConversion(t *testing.T) {
	assert.True(t, needsConversion(formatTOML))
	assert.False(t, needsConversion(formatYAML))
	assert.False(t, needsConversion(formatJSON))
	assert.False(t, needsConversion(formatENV))
	assert.False(t, needsConversion(formatINI))
	assert.False(t, needsConversion(formatUnknown))
}

func TestSOPSFormat(t *testing.T) {
	tests := []struct {
		name   string
		format format
		want   string
	}{
		{name: "toml", format: formatTOML, want: "yaml"},
		{name: "yaml", format: formatYAML, want: "yaml"},
		{name: "json", format: formatJSON, want: "json"},
		{name: "env", format: formatENV, want: "dotenv"},
		{name: "ini", format: formatINI, want: "ini"},
		{name: "unknown", format: formatUnknown, want: "yaml"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, sopsFormat(tt.format))
		})
	}
}
