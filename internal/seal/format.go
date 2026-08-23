package seal

import (
	"fmt"
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

// resolveFormat determines the YewSeal format name for a path, honoring an
// explicit format override when given.
func resolveFormat(path, override string) (string, error) {
	if strings.TrimSpace(override) != "" {
		userFormat := parseFormat(override)
		if userFormat == formatUnknown {
			return "", fmt.Errorf("unsupported format override %q (supported: %s)", override, supportedFormats())
		}
		return string(userFormat), nil
	}

	userFormat := detectFormat(path)
	if userFormat == formatUnknown {
		return "", fmt.Errorf("could not detect format for %s (supported: %s). Hint: pass --format binary if this should be encrypted as a binary file", path, supportedFormats())
	}
	return string(userFormat), nil
}

func supportedFormats() string {
	return "toml, yaml, json, env, ini, binary"
}

func NormalizeFormatForPath(filename string) (string, bool) {
	detected := detectFormat(filename)
	if detected == formatUnknown {
		return "", false
	}
	return string(detected), true
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
