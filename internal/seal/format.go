package seal

import (
	"path/filepath"
	"strings"
)

type format string

const (
	formatTOML    format = "toml"
	formatYAML    format = "yaml"
	formatJSON    format = "json"
	formatENV     format = "env"
	formatINI     format = "ini"
	formatBinary  format = "binary"
	formatUnknown format = "unknown"
)

func NormalizeFormatOverride(value string) (string, bool) {
	parsed := parseFormat(value)
	if parsed == formatUnknown {
		return "", false
	}
	return string(parsed), true
}

func detectFormat(filename string) format {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".toml":
		return formatTOML
	case ".yaml", ".yml":
		return formatYAML
	case ".json":
		return formatJSON
	case ".env":
		return formatENV
	case ".ini":
		return formatINI
	case ".bin", ".binary":
		return formatBinary
	default:
		return formatUnknown
	}
}

func parseFormat(value string) format {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "toml":
		return formatTOML
	case "yaml", "yml":
		return formatYAML
	case "json":
		return formatJSON
	case "env", "dotenv":
		return formatENV
	case "ini":
		return formatINI
	case "binary", "bin":
		return formatBinary
	default:
		return formatUnknown
	}
}
