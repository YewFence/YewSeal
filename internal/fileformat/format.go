package fileformat

import (
	"path/filepath"
	"slices"
	"strings"
)

type formatDefinition struct {
	name       string
	aliases    []string
	extensions []string
	suffix     string
}

var formats = []formatDefinition{
	{name: "toml", extensions: []string{".toml"}, suffix: "toml"},
	{name: "yaml", aliases: []string{"yml"}, extensions: []string{".yaml", ".yml"}, suffix: "yaml"},
	{name: "json", extensions: []string{".json"}, suffix: "json"},
	{name: "env", aliases: []string{"dotenv"}, extensions: []string{".env"}, suffix: "env"},
	{name: "ini", extensions: []string{".ini"}, suffix: "ini"},
	{name: "binary", aliases: []string{"bin"}, extensions: []string{".bin", ".binary"}, suffix: "bin"},
}

// Normalize returns the canonical format for an accepted name or alias.
func Normalize(value string) (string, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	for _, format := range formats {
		if value == format.name || slices.Contains(format.aliases, value) {
			return format.name, true
		}
	}
	return "", false
}

// Detect returns the canonical format inferred from a filename's extension.
func Detect(path string) (string, bool) {
	ext := strings.ToLower(filepath.Ext(path))
	for _, format := range formats {
		if slices.Contains(format.extensions, ext) {
			return format.name, true
		}
	}
	return "", false
}

// Spellings returns a sorted copy of all accepted format names and aliases.
func Spellings() []string {
	var spellings []string
	for _, format := range formats {
		spellings = append(spellings, format.name)
		spellings = append(spellings, format.aliases...)
	}
	slices.Sort(spellings)
	return spellings
}

// Supported returns the canonical format names for user-facing choices.
func Supported() string {
	names := make([]string, 0, len(formats))
	for _, format := range formats {
		names = append(names, format.name)
	}
	return strings.Join(names, ", ")
}
