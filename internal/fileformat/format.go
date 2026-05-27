package fileformat

import (
	"path/filepath"
	"strings"
)

// Format represents the configuration file format.
type Format string

const (
	TOML    Format = "toml"
	YAML    Format = "yaml"
	JSON    Format = "json"
	ENV     Format = "env"
	INI     Format = "ini"
	Unknown Format = "unknown"
)

// Detect detects the file format based on the file extension.
func Detect(filename string) Format {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".toml":
		return TOML
	case ".yaml", ".yml":
		return YAML
	case ".json":
		return JSON
	case ".env":
		return ENV
	case ".ini":
		return INI
	default:
		return Unknown
	}
}

// NeedsConversion returns true if the format needs conversion before SOPS can process it.
// Currently only TOML needs conversion to YAML.
func NeedsConversion(format Format) bool {
	return format == TOML
}

// SOPSFormat returns the SOPS type string for the given format.
func SOPSFormat(format Format) string {
	switch format {
	case TOML:
		return "yaml" // TOML is converted to YAML before SOPS processing
	case YAML:
		return "yaml"
	case JSON:
		return "json"
	case ENV:
		return "dotenv"
	case INI:
		return "ini"
	default:
		return "yaml"
	}
}

// IsSopsNative returns true if SOPS can directly process this format.
func IsSopsNative(format Format) bool {
	switch format {
	case YAML, JSON, ENV, INI:
		return true
	default:
		return false
	}
}

// Parse converts a user-provided format string to Format.
// It returns Unknown if the string is empty or unrecognized.
func Parse(s string) Format {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "toml":
		return TOML
	case "yaml", "yml":
		return YAML
	case "json":
		return JSON
	case "env", "dotenv":
		return ENV
	case "ini":
		return INI
	default:
		return Unknown
	}
}

func Resolve(path, override string) Format {
	if parsed := Parse(override); parsed != Unknown {
		return parsed
	}
	return Detect(path)
}

func SupportedExtensions() []string {
	return []string{".toml", ".yaml", ".yml", ".json", ".env", ".ini"}
}
