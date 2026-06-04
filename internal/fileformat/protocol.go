package fileformat

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/YewFence/YewSeal/internal/seal"
)

func EncryptPathForPlaintext(path, formatOverride string) (string, error) {
	format := formatOverride
	detected := ""
	if format == "" {
		pathFormat, ok := seal.NormalizeFormatForPath(path)
		if !ok {
			return "", fmt.Errorf("could not detect format for %s", path)
		}
		format = pathFormat
		detected = pathFormat
	} else {
		detected, _ = seal.NormalizeFormatForPath(path)
	}

	dir := filepath.Dir(path)
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	if detected != format {
		name = base
	}

	switch format {
	case "toml":
		if ext == "" {
			name = base
		}
		return filepath.Join(dir, name+".enc.toml.yaml"), nil
	case "yaml":
		if strings.EqualFold(ext, ".yml") {
			name = strings.TrimSuffix(base, ext)
		}
		return filepath.Join(dir, name+".enc.yaml"), nil
	case "json":
		return filepath.Join(dir, name+".enc.json"), nil
	case "env":
		return filepath.Join(dir, name+".enc.env"), nil
	case "ini":
		return filepath.Join(dir, name+".enc.ini"), nil
	case "binary":
		if ext == "" {
			name = base
		}
		return filepath.Join(dir, name+".enc.bin"), nil
	default:
		return "", fmt.Errorf("unsupported format %q", format)
	}
}

func PlaintextPathForEncrypted(path, formatOverride string) (string, string, error) {
	if formatOverride != "" {
		plaintextPath, err := plaintextPathForEncryptedWithFormat(path, formatOverride)
		return plaintextPath, formatOverride, err
	}

	dir := filepath.Dir(path)
	base := filepath.Base(path)
	suffixes := []struct {
		encryptedSuffix string
		plaintextSuffix string
		format          string
	}{
		{".enc.toml.yaml", ".toml", "toml"},
		{".enc.yaml", ".yaml", "yaml"},
		{".enc.json", ".json", "json"},
		{".enc.env", ".env", "env"},
		{".enc.ini", ".ini", "ini"},
		{".enc.bin", ".bin", "binary"},
	}
	for _, suffix := range suffixes {
		if strings.HasSuffix(strings.ToLower(base), suffix.encryptedSuffix) {
			name := base[:len(base)-len(suffix.encryptedSuffix)]
			return filepath.Join(dir, name+suffix.plaintextSuffix), suffix.format, nil
		}
	}
	return "", "", fmt.Errorf("encrypted filename %s does not follow the yewseal scan protocol; add a format rule to make the plaintext format unambiguous", path)
}

func PlaintextStemPathForEncrypted(path string) (string, string, bool) {
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	stem, format, ok := EncryptedStemAndFormat(base)
	if !ok {
		return "", "", false
	}
	return filepath.Join(dir, stem), format, true
}

func EncryptedStemAndFormat(base string) (string, string, bool) {
	suffixes := []struct {
		encryptedSuffix string
		format          string
	}{
		{".enc.toml.yaml", "toml"},
		{".enc.yaml", "yaml"},
		{".enc.json", "json"},
		{".enc.env", "env"},
		{".enc.ini", "ini"},
		{".enc.bin", "binary"},
	}
	lowerBase := strings.ToLower(base)
	for _, suffix := range suffixes {
		if strings.HasSuffix(lowerBase, suffix.encryptedSuffix) {
			return base[:len(base)-len(suffix.encryptedSuffix)], suffix.format, true
		}
	}
	return "", "", false
}

func plaintextPathForEncryptedWithFormat(path, format string) (string, error) {
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	lowerBase := strings.ToLower(base)
	suffix := ".enc." + encryptedFormatSuffix(format)
	if strings.HasSuffix(lowerBase, suffix) {
		name := base[:len(base)-len(suffix)]
		if filepath.Ext(name) != "" {
			return filepath.Join(dir, name), nil
		}
		return filepath.Join(dir, name+"."+plaintextFormatSuffix(format)), nil
	}
	if strings.HasSuffix(lowerBase, ".enc.toml.yaml") {
		name := base[:len(base)-len(".enc.toml.yaml")]
		if filepath.Ext(name) != "" {
			return filepath.Join(dir, name), nil
		}
		return filepath.Join(dir, name+"."+plaintextFormatSuffix(format)), nil
	}
	if stem, _, ok := EncryptedStemAndFormat(base); ok {
		if filepath.Ext(stem) != "" {
			return filepath.Join(dir, stem), nil
		}
		return filepath.Join(dir, stem+"."+plaintextFormatSuffix(format)), nil
	}
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	if strings.HasSuffix(strings.ToLower(name), ".enc") {
		name = name[:len(name)-len(".enc")]
	}
	return filepath.Join(dir, name+"."+plaintextFormatSuffix(format)), nil
}

func encryptedFormatSuffix(format string) string {
	if format == "toml" || format == "yaml" {
		return "yaml"
	}
	if format == "binary" {
		return "bin"
	}
	return format
}

func plaintextFormatSuffix(format string) string {
	if format == "binary" {
		return "bin"
	}
	return format
}
