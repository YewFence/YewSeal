package agekey

import (
	"fmt"
	"os"
	"strings"

	"filippo.io/age"
)

// IdentityBundle contains the normalized identities available to a consumer.
type IdentityBundle struct {
	identities []string
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
	if len(bundle.identities) == 0 {
		return IdentityBundle{}, fmt.Errorf("no valid Age identity found")
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

// GetIdentityBundle resolves identities without a configured fallback path.
func GetIdentityBundle(keyFile string) (IdentityBundle, error) {
	return GetIdentityBundleWithFallback(keyFile, "")
}

// GetIdentityBundleWithFallback resolves an explicit key file first, then environment sources,
// then the configured key file before using the built-in default path.
func GetIdentityBundleWithFallback(keyFile, fallbackKeyFile string) (IdentityBundle, error) {
	if keyFile != "" {
		return readIdentityBundle(keyFile)
	}

	if value := os.Getenv("YEWSEAL_AGE_IDENTITIES"); value != "" {
		parts := strings.Split(value, ",")
		for i, part := range parts {
			if strings.TrimSpace(part) == "" {
				return IdentityBundle{}, fmt.Errorf("YEWSEAL_AGE_IDENTITIES item %d is empty", i+1)
			}
		}
		return NewIdentityBundle(parts)
	}
	if value := os.Getenv("SOPS_AGE_KEY"); value != "" {
		return parseIdentityFile(value)
	}
	if path := os.Getenv("SOPS_AGE_KEY_FILE"); path != "" {
		content, err := os.ReadFile(path)
		if err == nil {
			return parseIdentityFile(string(content))
		}
		if !os.IsNotExist(err) {
			return IdentityBundle{}, &keyFileReadError{path: path, err: err}
		}
	}
	if os.Getenv("SOPS_AGE_KEY_CMD") != "" {
		value, err := GetAgeKey("")
		if err != nil {
			return IdentityBundle{}, err
		}
		return parseIdentityFile(value)
	}
	if fallbackKeyFile != "" {
		return readIdentityBundle(fallbackKeyFile)
	}
	value, err := GetAgeKey("")
	if err != nil {
		return IdentityBundle{}, err
	}
	return parseIdentityFile(value)
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
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || !strings.HasPrefix(line, "AGE-SECRET-KEY-") {
			continue
		}
		identities = append(identities, line)
	}
	return NewIdentityBundle(identities)
}

type keyFileReadError struct {
	path string
	err  error
}

func (e *keyFileReadError) Error() string {
	return fmt.Sprintf("failed to read Age key file %s: %v", e.path, e.err)
}
func (e *keyFileReadError) Unwrap() error { return e.err }
