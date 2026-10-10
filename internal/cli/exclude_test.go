package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"filippo.io/age"
	"github.com/YewFence/YewSeal/internal/config"
	"github.com/stretchr/testify/require"
)

func TestPlanShowsStaticExcludeOriginsAndVerboseScanCounts(t *testing.T) {
	clearCLIEnvironment(t)
	root := t.TempDir()
	t.Chdir(root)
	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	configPath := filepath.Join(root, ".yewseal.toml")
	require.NoError(t, os.WriteFile(configPath, fmt.Appendf(nil, `exclude = ["target/", "*.example.toml", "!keep.example.toml"]
[[encryption.groups]]
patterns = ["*.toml", "!.yewseal.toml"]
[recipients]
defaults = ["owner"]
[recipients.registry]
owner = %q
`, identity.Recipient().String()), 0600))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "target", "deep"), 0700))
	require.NoError(t, os.WriteFile(filepath.Join(root, "target", "deep", "build.toml"), []byte("value = 1\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "app.toml"), []byte("value = 1\n"), 0600))
	for _, tc := range []struct {
		name    string
		args    []string
		json    bool
		verbose bool
	}{
		{name: "text", args: []string{"plan"}},
		{name: "source", args: []string{"plan", "--source"}},
		{name: "verbose", args: []string{"plan", "-v"}, verbose: true},
		{name: "json", args: []string{"plan", "--json"}, json: true},
		{name: "json verbose with target", args: []string{"plan", "app.toml", "--json", "-v"}, json: true, verbose: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newRootCommand("test", config.LoadConfig)
			var output, diagnostics bytes.Buffer
			cmd.SetOut(&output)
			cmd.SetErr(&diagnostics)
			cmd.SetArgs(tc.args)
			require.NoError(t, cmd.Execute())
			if tc.verbose {
				visited := []string{".", ".yewseal.toml", "app.toml", "target"}
				require.Equal(t, fmt.Sprintf("Scan: %d entries, %d directories expanded\n", len(visited)*2, 2), diagnostics.String())
			} else {
				require.Empty(t, diagnostics.String())
			}
			if tc.json {
				var payload struct {
					Exclude []struct {
						Pattern string `json:"pattern"`
						Config  string `json:"config"`
						Index   int    `json:"index"`
					} `json:"exclude"`
					FilePairs []json.RawMessage `json:"file_pairs"`
					ScanStats json.RawMessage   `json:"scan_stats"`
				}
				require.NoError(t, json.Unmarshal(output.Bytes(), &payload))
				require.Len(t, payload.FilePairs, 1)
				require.Len(t, payload.Exclude, 3)
				for i, pattern := range []string{"target/", "*.example.toml", "!keep.example.toml"} {
					require.Equal(t, pattern, payload.Exclude[i].Pattern)
					require.Equal(t, configPath, payload.Exclude[i].Config)
					require.Equal(t, i+1, payload.Exclude[i].Index)
				}
				require.Nil(t, payload.ScanStats)
			} else {
				require.Contains(t, output.String(), "Exclude rules (loaded configuration)\n")
				require.Contains(t, output.String(), "  .yewseal.toml exclude[1] = \"target/\"\n")
				require.Contains(t, output.String(), "  .yewseal.toml exclude[2] = \"*.example.toml\"\n")
				require.Contains(t, output.String(), "  .yewseal.toml exclude[3] = \"!keep.example.toml\"\n")
			}
			require.NotContains(t, output.String(), "target/deep")
			require.NotContains(t, output.String(), "Scan:")
		})
	}
}
