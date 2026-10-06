package sopsconfig

import (
	"path/filepath"
	"testing"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestRenderScopesCiphertextPathsIndependentlyOfPlaintext(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "service")
	pairs := []config.ResolvedFilePair{
		{PlaintextPath: filepath.Join(root, "external.yaml"), EncryptedPath: filepath.Join(dir, "nested", "local.enc.yaml"), Recipients: []string{"age1owner"}},
		{PlaintextPath: filepath.Join(dir, "local.yaml"), EncryptedPath: filepath.Join(root, "external.enc.yaml"), Recipients: []string{"age1owner"}},
		{EncryptedPath: "relative.enc.yaml", Recipients: []string{"age1owner"}},
		{EncryptedPath: "../outside.enc.yaml"},
		{EncryptedPath: filepath.Join(root, "service-other", "outside.enc.yaml")},
	}
	data, err := Render(pairs, dir)
	require.NoError(t, err)
	var policy sopsConfig
	require.NoError(t, yaml.Unmarshal(data, &policy))
	require.Equal(t, []creationRule{
		{PathRegex: `^nested/local\.enc\.yaml$`, Age: "age1owner"},
		{PathRegex: `^relative\.enc\.yaml$`, Age: "age1owner"},
	}, policy.CreationRules)
	require.NotContains(t, string(data), root)
}
