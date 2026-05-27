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
		{"config.bin", formatBinary},
		{"config.binary", formatBinary},
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
		{"binary", formatBinary},
		{"bin", formatBinary},
		{"TOML", formatTOML},
		{"YAML", formatYAML},
		{"ENV", formatENV},
		{"Dotenv", formatENV},
		{" env ", formatENV},
		{"\tyaml\t", formatYAML},
		{"", formatUnknown},
		{"xml", formatUnknown},
		{"vars", formatUnknown},
		{"BINARY", formatBinary},
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
