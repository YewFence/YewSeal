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
	default:
		return formatUnknown
	}
}

func needsConversion(format format) bool {
	return format == formatTOML
}

func sopsFormat(format format) string {
	switch format {
	case formatTOML:
		return "yaml"
	case formatYAML:
		return "yaml"
	case formatJSON:
		return "json"
	case formatENV:
		return "dotenv"
	case formatINI:
		return "ini"
	default:
		return "yaml"
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
	default:
		return formatUnknown
	}
}

func resolveFormat(path, override string) format {
	if parsed := parseFormat(override); parsed != formatUnknown {
		return parsed
	}
	return detectFormat(path)
}

func supportedExtensions() []string {
	return []string{".toml", ".yaml", ".yml", ".json", ".env", ".ini"}
}
