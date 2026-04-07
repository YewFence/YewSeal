package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		filename string
		want     FileFormat
	}{
		{"config.toml", FormatTOML},
		{"config.yaml", FormatYAML},
		{"config.yml", FormatYAML},
		{"config.json", FormatJSON},
		{"config.env", FormatENV},
		{"config.ini", FormatINI},
		// 非标准扩展名 → unknown
		{".dev.vars", FormatUnknown},
		{"secrets.vars", FormatUnknown},
		{"wrangler", FormatUnknown},
		{"config.xml", FormatUnknown},
		// 大小写无影响
		{"config.TOML", FormatTOML},
		{"config.YAML", FormatYAML},
		{"config.JSON", FormatJSON},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			assert.Equal(t, tt.want, DetectFormat(tt.filename))
		})
	}
}

func TestParseFormat(t *testing.T) {
	tests := []struct {
		input string
		want  FileFormat
	}{
		// 基本格式
		{"toml", FormatTOML},
		{"yaml", FormatYAML},
		{"yml", FormatYAML},
		{"json", FormatJSON},
		{"env", FormatENV},
		{"dotenv", FormatENV},
		{"ini", FormatINI},
		// 大小写不敏感
		{"TOML", FormatTOML},
		{"YAML", FormatYAML},
		{"ENV", FormatENV},
		{"Dotenv", FormatENV},
		// 前后空格忽略
		{" env ", FormatENV},
		{"\tyaml\t", FormatYAML},
		// 空字符串和未知值 → unknown
		{"", FormatUnknown},
		{"xml", FormatUnknown},
		{"vars", FormatUnknown},
		{"binary", FormatUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, ParseFormat(tt.input))
		})
	}
}
