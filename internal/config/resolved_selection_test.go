package config

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolvedFilePairsToFilePairsCopiesFieldsAndAliases(t *testing.T) {
	aliases := []string{"dev"}
	pairs := ResolvedFilePairsToFilePairs([]ResolvedFilePair{
		{
			PlaintextPath:    "app.toml",
			EncryptedPath:    "app.toml.enc",
			Format:           "toml",
			ConfigPath:       ".yewseal.toml",
			RecipientAliases: aliases,
		},
	})

	require.Len(t, pairs, 1)
	assert.Equal(t, FilePair{
		PlaintextPath: "app.toml",
		EncryptedPath: "app.toml.enc",
		Format:        "toml",
		ConfigPath:    ".yewseal.toml",
		Recipients:    &[]string{"dev"},
	}, pairs[0])

	// 别名切片是拷贝，修改源切片不影响结果
	aliases[0] = "changed"
	assert.Equal(t, "dev", (*pairs[0].Recipients)[0])
}

func TestResolvedFilePairsToFilePairsNilAliasesStayNil(t *testing.T) {
	pairs := ResolvedFilePairsToFilePairs([]ResolvedFilePair{
		{PlaintextPath: "app.toml", EncryptedPath: "app.toml.enc"},
	})
	require.Len(t, pairs, 1)
	assert.Nil(t, pairs[0].Recipients)
}

func TestResolvedFilePairsToTaskPairsCopiesRecipients(t *testing.T) {
	recipients := []string{"age1xyz"}
	pairs := ResolvedFilePairsToTaskPairs([]ResolvedFilePair{
		{
			PlaintextPath: "app.toml",
			EncryptedPath: "app.toml.enc",
			Format:        "toml",
			Recipients:    recipients,
		},
	})

	require.Len(t, pairs, 1)
	assert.Equal(t, "app.toml", pairs[0].PlaintextPath)
	assert.Equal(t, "app.toml.enc", pairs[0].EncryptedPath)
	assert.Equal(t, "toml", pairs[0].Format)
	assert.Equal(t, []string{"age1xyz"}, pairs[0].Recipients)

	recipients[0] = "changed"
	assert.Equal(t, "age1xyz", pairs[0].Recipients[0])
}

func TestDisplayResolvedFilePairsRewritesAllPathFields(t *testing.T) {
	root := t.TempDir()
	inRoot := filepath.Join(root, "sub", "app.toml")
	pairs := DisplayResolvedFilePairs([]ResolvedFilePair{
		{
			PlaintextPath:   inRoot,
			EncryptedPath:   inRoot + ".enc",
			ConfigPath:      filepath.Join(root, ".yewseal.toml"),
			PlaintextSource: ValueSource{Kind: "mapping", ConfigPath: filepath.Join(root, ".yewseal.toml")},
			EncryptedSource: ValueSource{Kind: "mapping", ConfigPath: filepath.Join(root, ".yewseal.toml")},
			FormatSource:    ValueSource{Kind: "detect", ConfigPath: filepath.Join(root, ".yewseal.toml")},
		},
	}, root)

	require.Len(t, pairs, 1)
	pair := pairs[0]
	assert.Equal(t, filepath.Join("sub", "app.toml"), pair.PlaintextPath)
	assert.Equal(t, filepath.Join("sub", "app.toml.enc"), pair.EncryptedPath)
	assert.Equal(t, ".yewseal.toml", pair.ConfigPath)
	assert.Equal(t, ".yewseal.toml", pair.PlaintextSource.ConfigPath)
	assert.Equal(t, ".yewseal.toml", pair.EncryptedSource.ConfigPath)
	assert.Equal(t, ".yewseal.toml", pair.FormatSource.ConfigPath)
}

func TestFormatValueSource(t *testing.T) {
	root := t.TempDir()
	inRoot := filepath.Join(root, ".yewseal.toml")

	tests := []struct {
		name   string
		source ValueSource
		want   string
	}{
		{name: "empty kind falls back to unknown", source: ValueSource{}, want: "unknown"},
		{name: "kind only", source: ValueSource{Kind: "default"}, want: "default"},
		{name: "kind with detail", source: ValueSource{Kind: "mapping", Detail: "(explicit)"}, want: "mapping (explicit)"},
		{name: "config path is displayed relative to cwd", source: ValueSource{Kind: "mapping", ConfigPath: inRoot}, want: ".yewseal.toml mapping"},
		{name: "config path with detail", source: ValueSource{Kind: "mapping", ConfigPath: inRoot, Detail: "(explicit)"}, want: ".yewseal.toml (explicit)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, FormatValueSource(tt.source, root))
		})
	}
}
