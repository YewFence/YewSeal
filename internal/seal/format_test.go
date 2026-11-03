package seal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveFormat(t *testing.T) {
	for _, tc := range []struct {
		name, path, override, want string
	}{
		{"inferred", "config.yml", "", "yaml"},
		{"explicit wins", "config.yaml", "dotenv", "env"},
		{"extensionless", "secret", "bin", "binary"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveFormat(tc.path, tc.override)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestResolveFormatRejectsUnknownDetectedFormatWithBinaryHint(t *testing.T) {
	_, err := resolveFormat("config.unknown", "")
	require.ErrorContains(t, err, "could not detect format for config.unknown")
	assert.Contains(t, err.Error(), "Declare a format in .yewseal.toml; use binary")
}

func TestResolveFormatRejectsInvalidExplicitOverride(t *testing.T) {
	_, err := resolveFormat("config.yaml", "xml")
	require.ErrorContains(t, err, `unsupported format override "xml"`)
}
