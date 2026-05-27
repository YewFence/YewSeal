package fileformat

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetect(t *testing.T) {
	tests := []struct {
		filename string
		want     Format
	}{
		{"config.toml", TOML},
		{"config.yaml", YAML},
		{"config.yml", YAML},
		{"config.json", JSON},
		{"config.env", ENV},
		{"config.ini", INI},
		// 非标准扩展名 → unknown
		{".dev.vars", Unknown},
		{"secrets.vars", Unknown},
		{"wrangler", Unknown},
		{"config.xml", Unknown},
		// 大小写无影响
		{"config.TOML", TOML},
		{"config.YAML", YAML},
		{"config.JSON", JSON},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			assert.Equal(t, tt.want, Detect(tt.filename))
		})
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		input string
		want  Format
	}{
		// 基本格式
		{"toml", TOML},
		{"yaml", YAML},
		{"yml", YAML},
		{"json", JSON},
		{"env", ENV},
		{"dotenv", ENV},
		{"ini", INI},
		// 大小写不敏感
		{"TOML", TOML},
		{"YAML", YAML},
		{"ENV", ENV},
		{"Dotenv", ENV},
		// 前后空格忽略
		{" env ", ENV},
		{"\tyaml\t", YAML},
		// 空字符串和未知值 → unknown
		{"", Unknown},
		{"xml", Unknown},
		{"vars", Unknown},
		{"binary", Unknown},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, Parse(tt.input))
		})
	}
}
