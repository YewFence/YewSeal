package config

import (
	"fmt"
	"os"
	"testing"

	"github.com/YewFence/YewSeal/internal/task"
	"github.com/stretchr/testify/require"
)

func TestLoadPlaintextMode(t *testing.T) {
	for _, tc := range []struct{ field, want string }{
		{"", PlaintextInplace},
		{`plaintext_mode = "inplace"`, PlaintextInplace},
		{`plaintext_mode = "delivery"`, PlaintextDelivery},
		{`plaintext_mode = "unknown"`, ""},
	} {
		t.Run(tc.field, func(t *testing.T) {
			t.Chdir(t.TempDir())
			data := fmt.Sprintf("[[encryption.files]]\nplaintext='secret.yaml'\nencrypted='secret.enc.yaml'\n%s\n", tc.field)
			require.NoError(t, os.WriteFile(".yewseal.toml", []byte(data), 0600))
			cfg, err := LoadConfig()
			if tc.want == "" {
				require.ErrorContains(t, err, "plaintext_mode must be inplace or delivery")
				return
			}
			require.NoError(t, err)
			require.Equal(t, CurrentDir(cfg), cfg.ProjectRoot)
			require.Equal(t, tc.want, cfg.GetFiles()[0].PlaintextMode)
			selection, err := ResolveSelection(cfg, SelectionOptions{Command: task.ModeDecrypt})
			require.NoError(t, err)
			require.Equal(t, tc.want, selection.FilePairs[0].PlaintextMode)
			require.Equal(t, tc.want, ResolvedFilePairsToFilePairs(selection.FilePairs)[0].PlaintextMode)
		})
	}
}

func TestGroupPlaintextModeDefaultsToInplace(t *testing.T) {
	cfg := selectionConfig(t)
	selectionFile(t, cfg.CurrentDir, "secret.enc.yaml")
	cfg.Encryption.Groups = []GroupConfig{{ConfigDir: cfg.CurrentDir, Patterns: []string{"*.yaml"}}}
	selection, err := ResolveSelection(cfg, SelectionOptions{Command: task.ModeDecrypt})
	require.NoError(t, err)
	require.Len(t, selection.FilePairs, 1)
	require.Equal(t, PlaintextInplace, selection.FilePairs[0].PlaintextMode)
}

func TestDecryptSelectionRejectsFilenameOutputOverride(t *testing.T) {
	cfg := selectionConfig(t)
	plain := selectionFile(t, cfg.CurrentDir, "secret.yaml")
	cfg.Encryption.Files = []FilePair{{PlaintextPath: plain, EncryptedPath: plain + ".enc"}}
	_, err := ResolveSelection(cfg, SelectionOptions{Command: task.ModeDecrypt, Targets: []string{plain}, OutputSet: true, Output: "renamed.yaml"})
	require.ErrorContains(t, err, "decrypt does not support output overrides")
}
