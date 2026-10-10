package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"filippo.io/age"
	"github.com/YewFence/YewSeal/internal/config"
	"github.com/stretchr/testify/require"
)

func TestScanningCommandsReportOneVerboseSummaryOnStderr(t *testing.T) {
	for _, tc := range []struct {
		name  string
		args  []string
		scans int
	}{
		{name: "encrypt includes metadata scans", args: []string{"encrypt", "-v", "--force"}, scans: 3},
		{name: "decrypt", args: []string{"decrypt", "-v", "--update-gitignore=false"}, scans: 1},
		{name: "clean", args: []string{"clean", "-v", "--force"}, scans: 2},
		{name: "diff", args: []string{"diff", "-v"}, scans: 1},
		{name: "view", args: []string{"view", "app.toml", "-v"}, scans: 1},
		{name: "edit", args: []string{"edit", "-f", "app.toml", "-v"}, scans: 1},
		{name: "verify", args: []string{"verify", "-v", "--no-decrypt", "--sync-sops-config=false"}, scans: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			clearCLIEnvironment(t)
			root := t.TempDir()
			t.Chdir(root)
			require.NoError(t, os.WriteFile(filepath.Join(root, "app.toml"), []byte("value = 1\n"), 0600))
			identity, err := age.GenerateX25519Identity()
			require.NoError(t, err)
			defaults := []string{"owner"}
			cfg := &config.Config{
				CurrentDir: root, ProjectRoot: root,
				Recipients: config.RecipientConfig{Defaults: &defaults, Registry: map[string]string{"owner": identity.Recipient().String()}},
				Encryption: config.EncryptionConfig{
					Files:  []config.FilePair{{PlaintextPath: filepath.Join(root, "app.toml"), EncryptedPath: filepath.Join(root, "app.enc.toml"), Format: "toml"}},
					Groups: []config.GroupConfig{{ConfigDir: root, Patterns: []string{"*.toml"}}},
				},
			}
			cmd := newRootCommand("test", func() (*config.Config, error) { return cfg, nil })
			var output, diagnostics bytes.Buffer
			cmd.SetOut(&output)
			cmd.SetErr(&diagnostics)
			cmd.SetArgs(tc.args)
			err = cmd.Execute()
			if tc.args[0] == "view" || tc.args[0] == "edit" || tc.args[0] == "verify" || tc.args[0] == "decrypt" {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			visited := []string{".", "app.toml"}
			expected := fmt.Sprintf("Scan: %d entries, %d directories expanded\n", len(visited)*tc.scans, tc.scans)
			require.Contains(t, diagnostics.String(), expected)
			require.Equal(t, 1, strings.Count(diagnostics.String(), "Scan:"))
			require.NotContains(t, output.String(), "Scan:")
		})
	}
}
