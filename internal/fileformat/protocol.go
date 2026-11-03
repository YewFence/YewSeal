package fileformat

import (
	"fmt"
	"path/filepath"
	"strings"
)

// EncryptPathForPlaintext generates a ciphertext path using the naming protocol.
// An empty format is inferred from the plaintext extension.
func EncryptPathForPlaintext(path, formatOverride string) (string, error) {
	format := formatOverride
	detected, _ := Detect(path)
	if format == "" {
		if detected == "" {
			return "", fmt.Errorf("could not detect format for %s", path)
		}
		format = detected
	}
	canonical, ok := Normalize(format)
	if !ok {
		return "", fmt.Errorf("unsupported format %q", format)
	}
	base := filepath.Base(path)
	name := base
	if detected == canonical {
		name = strings.TrimSuffix(base, filepath.Ext(base))
	}
	return filepath.Join(filepath.Dir(path), name+".enc."+formatSuffix(canonical)), nil
}

// PlaintextPathForEncrypted infers a plaintext path and canonical format.
// The naming protocol cannot recover every original plaintext filename.
func PlaintextPathForEncrypted(path, formatOverride string) (string, string, error) {
	if formatOverride != "" {
		format, ok := Normalize(formatOverride)
		if !ok {
			return "", "", fmt.Errorf("unsupported format %q", formatOverride)
		}
		return plaintextPathForEncryptedWithFormat(path, format), format, nil
	}
	stem, format, ok := EncryptedStemAndFormat(filepath.Base(path))
	if !ok {
		return "", "", fmt.Errorf("encrypted filename %s does not follow the yewseal group protocol; declare an explicit file mapping for a custom ciphertext name", path)
	}
	return filepath.Join(filepath.Dir(path), stem+"."+formatSuffix(format)), format, nil
}

// PlaintextStemPathForEncrypted removes a recognized ciphertext protocol suffix.
func PlaintextStemPathForEncrypted(path string) (string, string, bool) {
	stem, format, ok := EncryptedStemAndFormat(filepath.Base(path))
	if !ok {
		return "", "", false
	}
	return filepath.Join(filepath.Dir(path), stem), format, true
}

// EncryptedPatterns returns the filename patterns used for Group discovery.
func EncryptedPatterns() []string {
	patterns := make([]string, 0, len(formats))
	for _, format := range formats {
		patterns = append(patterns, "*.enc."+format.suffix)
	}
	return patterns
}

// EncryptedStemAndFormat recognizes the protocol suffix of a ciphertext filename.
func EncryptedStemAndFormat(base string) (string, string, bool) {
	lowerBase := strings.ToLower(base)
	for _, format := range formats {
		suffix := ".enc." + format.suffix
		if strings.HasSuffix(lowerBase, suffix) {
			return base[:len(base)-len(suffix)], format.name, true
		}
	}
	return "", "", false
}

func plaintextPathForEncryptedWithFormat(path, format string) string {
	dir, base := filepath.Dir(path), filepath.Base(path)
	suffix := ".enc." + formatSuffix(format)
	if strings.HasSuffix(strings.ToLower(base), suffix) {
		name := base[:len(base)-len(suffix)]
		if filepath.Ext(name) != "" {
			return filepath.Join(dir, name)
		}
		return filepath.Join(dir, name+"."+formatSuffix(format))
	}
	if stem, _, ok := EncryptedStemAndFormat(base); ok {
		if filepath.Ext(stem) != "" {
			return filepath.Join(dir, stem)
		}
		return filepath.Join(dir, stem+"."+formatSuffix(format))
	}
	name := strings.TrimSuffix(base, filepath.Ext(base))
	if strings.HasSuffix(strings.ToLower(name), ".enc") {
		name = name[:len(name)-len(".enc")]
	}
	return filepath.Join(dir, name+"."+formatSuffix(format))
}

func formatSuffix(name string) string {
	for _, format := range formats {
		if format.name == name {
			return format.suffix
		}
	}
	panic("non-canonical format: " + name)
}
