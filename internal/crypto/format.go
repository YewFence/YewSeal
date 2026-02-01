package crypto

import (
	"path/filepath"
	"strings"
)

// FileFormat represents the configuration file format
type FileFormat string

const (
	FormatTOML    FileFormat = "toml"
	FormatYAML    FileFormat = "yaml"
	FormatJSON    FileFormat = "json"
	FormatENV     FileFormat = "env"
	FormatINI     FileFormat = "ini"
	FormatUnknown FileFormat = "unknown"
)

// DetectFormat detects the file format based on the file extension
func DetectFormat(filename string) FileFormat {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".toml":
		return FormatTOML
	case ".yaml", ".yml":
		return FormatYAML
	case ".json":
		return FormatJSON
	case ".env":
		return FormatENV
	case ".ini":
		return FormatINI
	default:
		return FormatUnknown
	}
}

// NeedsConversion returns true if the format needs conversion before SOPS can process it
// Currently only TOML needs conversion (to YAML)
func NeedsConversion(format FileFormat) bool {
	return format == FormatTOML
}

// GetSopsType returns the SOPS type string for the given format
func GetSopsType(format FileFormat) string {
	switch format {
	case FormatTOML:
		return "yaml" // TOML is converted to YAML before SOPS processing
	case FormatYAML:
		return "yaml"
	case FormatJSON:
		return "json"
	case FormatENV:
		return "dotenv"
	case FormatINI:
		return "ini"
	default:
		return "yaml"
	}
}

// IsSopsNativeFormat returns true if SOPS can directly process this format
func IsSopsNativeFormat(format FileFormat) bool {
	switch format {
	case FormatYAML, FormatJSON, FormatENV, FormatINI:
		return true
	default:
		return false
	}
}
