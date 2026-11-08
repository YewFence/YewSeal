package cli

import (
	"testing"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/stretchr/testify/require"
)

func TestDiffSourceEnvironmentReachesAppLayer(t *testing.T) {
	for _, tc := range []struct {
		name    string
		args    []string
		env     map[string]string
		message string
	}{
		{"empty-flag", []string{"--source="}, nil, "requires a directory"},
		{"blank-env", nil, map[string]string{"YEWSEAL_DIFF_SOURCE": " "}, "requires a directory"},
		{"env-source", nil, map[string]string{"YEWSEAL_DIFF_SOURCE": "missing-tree"}, "must be an existing, real directory"},
		{"flag-wins-over-env", []string{"--source", "missing-tree"}, map[string]string{"YEWSEAL_DIFF_SOURCE": "another"}, "must be an existing, real directory"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			clearCLIEnvironment(t)
			for key, value := range tc.env {
				t.Setenv(key, value)
			}
			cwd := t.TempDir()
			root := newRootCommand("test", func() (*config.Config, error) {
				return &config.Config{CurrentDir: cwd, ProjectRoot: cwd}, nil
			})
			t.Chdir(cwd)
			quietCommand(root, append([]string{"diff"}, tc.args...))
			err := root.Execute()
			require.ErrorContains(t, err, tc.message)
			require.Equal(t, 2, ExitCode(root, err))
		})
	}
}
