package agekey

import (
	"fmt"
	"os"
	"strings"
	"unicode"

	"filippo.io/age"
	"github.com/YewFence/YewSeal/internal/errx"
)

// IdentityBundle contains normalized identities and redacted parse diagnostics.
type IdentityBundle struct {
	identities []string
	warnings   []string
}

// NewIdentityBundle validates and deduplicates Age identities.
func NewIdentityBundle(identities []string) (IdentityBundle, error) {
	seen := make(map[string]struct{}, len(identities))
	bundle := IdentityBundle{identities: make([]string, 0, len(identities))}
	for _, value := range identities {
		identity := strings.TrimSpace(value)
		if identity == "" {
			return IdentityBundle{}, fmt.Errorf("age identity must not be empty")
		}
		if _, ok := seen[identity]; ok {
			continue
		}
		if _, err := age.ParseX25519Identity(identity); err != nil {
			return IdentityBundle{}, fmt.Errorf("invalid Age identity: %w", err)
		}
		seen[identity] = struct{}{}
		bundle.identities = append(bundle.identities, identity)
	}
	return bundle, nil
}

// String serializes the bundle for the SOPS Age parser.
func (b IdentityBundle) String() string {
	return strings.Join(b.identities, "\n")
}

// Identities returns a copy of the bundle identities for internal consumers.
func (b IdentityBundle) Identities() []string {
	return append([]string(nil), b.identities...)
}

// Warnings returns redacted diagnostics produced while parsing the bundle.
func (b IdentityBundle) Warnings() []string {
	return append([]string(nil), b.warnings...)
}

// GetIdentityBundle resolves an explicit key file first, then environment sources,
// then .age/keys.txt relative to the current working directory.
func GetIdentityBundle(keyFile string) (IdentityBundle, error) {
	bundle, err := getIdentityBundle(keyFile)
	if err != nil {
		return bundle, errx.Usage(err)
	}
	return bundle, nil
}

func getIdentityBundle(keyFile string) (IdentityBundle, error) {
	for _, layer := range identityLayers(keyFile) {
		if !layer.present() {
			continue
		}
		return layer.load()
	}
	return IdentityBundle{}, nil
}

func readIdentityBundle(path string) (IdentityBundle, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return IdentityBundle{}, &keyFileReadError{path: path, err: err}
	}
	return parseIdentityFile(string(content))
}

func parseIdentityFile(content string) (IdentityBundle, error) {
	identities := make([]string, 0)
	warnings := make([]string, 0)
	for lineIndex, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.FieldsFunc(line, func(r rune) bool {
			return r == ',' || unicode.IsSpace(r)
		})
		for fieldIndex, field := range fields {
			if _, err := age.ParseX25519Identity(field); err != nil {
				warnings = append(warnings, fmt.Sprintf("ignored malformed Age identity bundle item at line %d, item %d (%s)", lineIndex+1, fieldIndex+1, redactIdentityItem(field)))
				continue
			}
			identities = append(identities, field)
		}
	}
	bundle, err := NewIdentityBundle(identities)
	if err != nil {
		return IdentityBundle{}, err
	}
	bundle.warnings = warnings
	return bundle, nil
}

func redactIdentityItem(value string) string {
	const ageIdentityPrefix = "AGE-SECRET-KEY-"
	const minHiddenRunes = 8
	runes := []rune(value)
	if strings.HasPrefix(value, ageIdentityPrefix) && len(runes) >= len(ageIdentityPrefix)+4+minHiddenRunes {
		return ageIdentityPrefix + "…" + string(runes[len(runes)-4:])
	}
	if len(runes) >= 2+2+3 {
		return string(runes[:2]) + "…" + string(runes[len(runes)-2:])
	}
	return fmt.Sprintf("[REDACTED: %d chars]", len(runes))
}

type keyFileReadError struct {
	path string
	err  error
}

func (e *keyFileReadError) Error() string {
	return fmt.Sprintf("failed to read Age key file %s: %v", e.path, e.err)
}
func (e *keyFileReadError) Unwrap() error { return e.err }
