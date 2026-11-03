package fileformat

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalize(t *testing.T) {
	for _, spelling := range Spellings() {
		canonical, ok := Normalize(spelling)
		require.True(t, ok)
		upper, ok := Normalize(" \t" + strings.ToUpper(spelling) + "\t ")
		require.True(t, ok)
		assert.Equal(t, canonical, upper)
	}
	for input, want := range map[string]string{"yml": "yaml", "dotenv": "env", "bin": "binary"} {
		got, ok := Normalize(input)
		require.True(t, ok)
		assert.Equal(t, want, got)
	}
	for _, input := range []string{"", "xml", "vars"} {
		got, ok := Normalize(input)
		assert.False(t, ok)
		assert.Empty(t, got)
	}
}

func TestDetect(t *testing.T) {
	for _, tc := range []struct{ path, want string }{
		{"config.toml", "toml"}, {"config.yaml", "yaml"}, {"config.yml", "yaml"},
		{"config.json", "json"}, {"config.env", "env"}, {"config.ini", "ini"},
		{"config.bin", "binary"}, {"config.binary", "binary"},
		{"config.TOML", "toml"}, {"config.YAML", "yaml"}, {"config.JSON", "json"},
		{".dev.vars", ""}, {"secret", ""}, {"config.xml", ""},
	} {
		t.Run(tc.path, func(t *testing.T) {
			got, ok := Detect(tc.path)
			assert.Equal(t, tc.want, got)
			assert.Equal(t, tc.want != "", ok)
		})
	}
}

func TestSpellingsReturnsIndependentCopy(t *testing.T) {
	want := []string{"bin", "binary", "dotenv", "env", "ini", "json", "toml", "yaml", "yml"}
	got := Spellings()
	require.Equal(t, want, got)
	got[0] = "changed"
	assert.Equal(t, want, Spellings())
	assert.Equal(t, "toml, yaml, json, env, ini, binary", Supported())
}
