package task

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"filippo.io/age"
	"github.com/YewFence/YewSeal/internal/agekey"
	"github.com/YewFence/YewSeal/internal/sopsx"
	"github.com/stretchr/testify/require"
)

func TestDecryptOutcomesAndSideEffects(t *testing.T) {
	for _, workers := range []int{1, 4} {
		for _, scenario := range []string{"partial", "strict", "all-skipped", "broken", "blocked-output"} {
			t.Run(fmt.Sprintf("%s/parallel=%d", scenario, workers), func(t *testing.T) {
				t.Chdir(t.TempDir())
				owner, err := age.GenerateX25519Identity()
				require.NoError(t, err)
				other, err := age.GenerateX25519Identity()
				require.NoError(t, err)
				bundle, err := agekey.NewIdentityBundle([]string{owner.String()})
				require.NoError(t, err)
				plain := []byte("token: value\n")
				for name, recipient := range map[string]string{"owned.enc.yaml": owner.Recipient().String(), "other.enc.yaml": other.Recipient().String()} {
					data, err := sopsx.Encrypt(plain, "yaml", []string{recipient})
					require.NoError(t, err)
					require.NoError(t, os.WriteFile(name, data, 0600))
				}
				stale := []byte("local edits that must survive\n")
				require.NoError(t, os.WriteFile("stale.yaml", stale, 0600))
				pairs := []FilePair{
					{PlaintextPath: "unavailable/other.yaml", EncryptedPath: "other.enc.yaml", Format: "yaml"},
					{PlaintextPath: "stale.yaml", EncryptedPath: "other.enc.yaml", Format: "yaml"},
				}
				if scenario != "all-skipped" {
					pairs = append(pairs, FilePair{PlaintextPath: "available/owned.yaml", EncryptedPath: "owned.enc.yaml", Format: "yaml"})
				}
				if scenario == "broken" {
					require.NoError(t, os.WriteFile("broken.enc.yaml", []byte("not sops"), 0600))
					pairs = append([]FilePair{{PlaintextPath: "broken-output/value.yaml", EncryptedPath: "broken.enc.yaml", Format: "yaml"}}, pairs...)
				}
				if scenario == "blocked-output" {
					require.NoError(t, os.WriteFile("blocked", []byte("not a directory"), 0600))
					pairs = append([]FilePair{{PlaintextPath: "blocked/value.yaml", EncryptedPath: "owned.enc.yaml", Format: "yaml"}}, pairs...)
				}
				summary, err := Decrypt(Options{FilePairs: pairs, IdentityBundle: bundle, Parallel: workers, Strict: scenario == "strict", Force: true})
				if scenario == "partial" {
					require.NoError(t, err)
				} else {
					require.Error(t, err)
				}
				require.Equal(t, 2, summary.SkippedCount)
				require.Equal(t, len(pairs), summary.TotalFiles)
				if scenario == "broken" || scenario == "blocked-output" {
					require.Equal(t, 1, summary.FailedCount)
				} else {
					require.Zero(t, summary.FailedCount)
				}
				require.NoDirExists(t, "unavailable")
				require.NoDirExists(t, "broken-output")
				content, readErr := os.ReadFile("stale.yaml")
				require.NoError(t, readErr)
				require.Equal(t, stale, content)
				if scenario != "all-skipped" {
					require.Equal(t, 1, summary.SuccessCount)
					content, readErr = os.ReadFile("available/owned.yaml")
					require.NoError(t, readErr)
					require.Equal(t, plain, content)
				} else {
					require.Zero(t, summary.SuccessCount)
				}
				var report bytes.Buffer
				require.NoError(t, summary.Report(&report, "decrypted"))
				require.Contains(t, report.String(), "SKIPPED other.enc.yaml")
				require.Contains(t, report.String(), "encrypted content was not verified")
			})
		}
	}
}

func TestDiffGroupUnion(t *testing.T) {
	for _, names := range [][]string{
		{"dev.yaml", "dev.enc.yaml", "prod.enc.yaml", "new.yaml"},
		{"prod.enc.yaml"},
		{"new.yaml"},
	} {
		t.Run(names[0], func(t *testing.T) {
			root := t.TempDir()
			for _, name := range names {
				require.NoError(t, os.WriteFile(filepath.Join(root, name), []byte("unused"), 0600))
			}
			pairs, err := BuildProjectGroupFilePairs(GroupOptions{Root: root, Patterns: []string{"*.yaml"}, Mode: ModeDiff})
			require.NoError(t, err)
			expected := 1
			if len(names) > 1 {
				expected = 3
			}
			require.Len(t, pairs, expected)
			for _, pair := range pairs {
				require.Equal(t, "yaml", pair.Format)
			}
		})
	}
}

func TestDiffGroupUnionKeepsExistingYMLPath(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"config.yml", "config.enc.yaml"} {
		require.NoError(t, os.WriteFile(filepath.Join(root, name), []byte("unused"), 0600))
	}
	pairs, err := BuildProjectGroupFilePairs(GroupOptions{Root: root, Mode: ModeDiff})
	require.NoError(t, err)
	require.Len(t, pairs, 1)
	require.Equal(t, filepath.Join(root, "config.yml"), pairs[0].PlaintextPath)
	require.NoError(t, os.WriteFile(filepath.Join(root, "config.yaml"), []byte("ambiguous"), 0600))
	pairs, err = BuildProjectGroupFilePairs(GroupOptions{Root: root, Mode: ModeDiff})
	require.NoError(t, err)
	require.Len(t, pairs, 2, "configuration selection must arbitrate competing discovered mappings")
}

func TestDiffGroupUnionPreservesCompetingCiphertextCandidates(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"config.conf", "config.conf.enc.env", "config.conf.enc.ini"} {
		require.NoError(t, os.WriteFile(filepath.Join(root, name), []byte("unused"), 0600))
	}
	pairs, err := BuildProjectGroupFilePairs(GroupOptions{Root: root, Mode: ModeDiff, Patterns: []string{"*.conf"}, FormatRules: []string{"*.conf=env"}})
	require.NoError(t, err)
	require.Len(t, pairs, 2)
}
