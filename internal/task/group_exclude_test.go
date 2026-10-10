package task

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGroupExcludePrunesBothDirectionsAndCountsActualVisits(t *testing.T) {
	for _, mode := range []string{ModeEncrypt, ModeDecrypt, ModePlan, ModeClean} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			for _, path := range []string{
				"app.toml", "app.enc.toml",
				".git/objects/secret.toml", ".git/objects/secret.enc.toml",
				".jj/store/secret.toml", ".jj/store/secret.enc.toml",
				"target/deep/build.toml", "target/deep/build.enc.toml",
				"open/keep.toml", "open/keep.enc.toml",
			} {
				full := filepath.Join(root, filepath.FromSlash(path))
				require.NoError(t, os.MkdirAll(filepath.Dir(full), 0700))
				require.NoError(t, os.WriteFile(full, []byte("value = 1\n"), 0600))
			}
			var stats ScanStats
			pairs, err := BuildProjectGroupFilePairs(GroupOptions{
				Root: root, Patterns: []string{"**/*"}, Exclude: []string{"target/", "!.git/", "!.jj/"},
				Mode: mode, Stats: &stats,
			})
			require.NoError(t, err)
			require.Len(t, pairs, 2)
			for _, pair := range pairs {
				require.Contains(t, []string{filepath.Join(root, "app.toml"), filepath.Join(root, "open", "keep.toml")}, pair.PlaintextPath)
			}
			visited := []string{".", ".git", ".jj", "target", "open", "app.toml", "app.enc.toml", "open/keep.toml", "open/keep.enc.toml"}
			expanded := []string{".", "open"}
			scans := 1
			if mode == ModePlan || mode == ModeClean {
				scans = 2
			}
			require.Equal(t, len(visited)*scans, stats.Entries)
			require.Equal(t, len(expanded)*scans, stats.Directories)
		})
	}
}

func TestGroupExcludeNegationCannotOverridePatternsOrPrunedParents(t *testing.T) {
	for _, mode := range []string{ModeEncrypt, ModeDecrypt, ModePlan} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			for _, stem := range []string{"app", "example", "forbidden", "ignored/keep"} {
				for _, suffix := range []string{".toml", ".enc.toml"} {
					full := filepath.Join(root, filepath.FromSlash(stem+suffix))
					require.NoError(t, os.MkdirAll(filepath.Dir(full), 0700))
					require.NoError(t, os.WriteFile(full, []byte("value = 1\n"), 0600))
				}
			}
			pairs, err := BuildProjectGroupFilePairs(GroupOptions{
				Root: root, Patterns: []string{"**/*.toml", "!forbidden.toml"},
				Exclude: []string{"*.toml", "!app.toml", "!forbidden.toml", "ignored/", "!ignored/keep.toml"}, Mode: mode,
			})
			require.NoError(t, err)
			require.Len(t, pairs, 1)
			require.Equal(t, filepath.Join(root, "app.toml"), pairs[0].PlaintextPath)
		})
	}
}

func TestGroupExcludeDoesNotChangeCiphertextMapping(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "token.enc.toml"), []byte("ciphertext"), 0600))
	pairs, err := BuildProjectGroupFilePairs(GroupOptions{
		Root: root, Patterns: []string{"token.toml", "token"}, Exclude: []string{"token.toml"},
		Mode: ModeDecrypt, AllowEmpty: true,
	})
	require.NoError(t, err)
	require.Empty(t, pairs)
}

func TestGroupExcludeUsesStemForNonstandardPlaintext(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, ".dev.vars.enc.env"), []byte("ciphertext"), 0600))
	pairs, err := BuildProjectGroupFilePairs(GroupOptions{
		Root: root, Patterns: []string{".dev.vars"}, FormatRules: []string{".dev.vars=env"},
		Exclude: []string{".dev.vars"}, Mode: ModeDecrypt, AllowEmpty: true,
	})
	require.NoError(t, err)
	require.Empty(t, pairs)
}

func TestGroupExcludeCanReincludeDirectoryBeforePruning(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "target"), 0700))
	require.NoError(t, os.WriteFile(filepath.Join(root, "target", "app.toml"), []byte("value = 1\n"), 0600))
	pairs, err := BuildProjectGroupFilePairs(GroupOptions{
		Root: root, Patterns: []string{"**/*.toml"}, Exclude: []string{"target/", "!target/"}, Mode: ModeEncrypt,
	})
	require.NoError(t, err)
	require.Len(t, pairs, 1)
}
