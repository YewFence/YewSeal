package seal

import (
	"fmt"
	"maps"
	"path/filepath"
	"slices"
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

// formatSpellings 是所有被接受的格式拼写(规范名与别名)到规范格式的映射,
// 是运行时格式契约的唯一事实来源;schema 同步测试以 FormatSpellings 为准。
var formatSpellings = map[string]format{
	"toml":   formatTOML,
	"yaml":   formatYAML,
	"yml":    formatYAML,
	"json":   formatJSON,
	"env":    formatENV,
	"dotenv": formatENV,
	"ini":    formatINI,
	"binary": formatBinary,
	"bin":    formatBinary,
}

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
	if parsed, ok := formatSpellings[strings.ToLower(strings.TrimSpace(value))]; ok {
		return parsed
	}
	return formatUnknown
}

// FormatSpellings 返回运行时接受的所有格式拼写(规范名与别名),按字典序排列。
// 每次调用返回新切片,调用方的修改不会影响包内状态。
func FormatSpellings() []string {
	return slices.Sorted(maps.Keys(formatSpellings))
}
