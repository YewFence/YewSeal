package cli

import (
	"bytes"
	"errors"
	"testing"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/stretchr/testify/require"
)

func TestUpdateGitignoreHelpUsesSharedEnvironment(t *testing.T) {
	t.Setenv("YEWSEAL_UPDATE_GITIGNORE", "invalid")
	for _, command := range []string{"init", "encrypt", "decrypt"} {
		t.Run(command, func(t *testing.T) {
			root := newRootCommand("test", func() (*config.Config, error) {
				t.Fatal("help must not load configuration")
				return nil, nil
			})
			var output bytes.Buffer
			root.SetOut(&output)
			root.SetErr(&output)
			root.SetArgs([]string{command, "--help"})
			require.NoError(t, root.Execute())
			require.Contains(t, output.String(), "--update-gitignore")
			require.Contains(t, output.String(), "env YEWSEAL_UPDATE_GITIGNORE")
			cmd, _, err := root.Find([]string{command})
			require.NoError(t, err)
			flag := cmd.Flags().Lookup("update-gitignore")
			require.Equal(t, "true", flag.DefValue)
			name, shared := sharedEnvName(flag)
			require.True(t, shared)
			require.Equal(t, "YEWSEAL_UPDATE_GITIGNORE", name)
		})
	}
}

func TestUpdateGitignoreEnvironmentDoesNotAffectReadOnlyCommands(t *testing.T) {
	t.Setenv("YEWSEAL_UPDATE_GITIGNORE", "invalid")
	for _, command := range []string{"verify", "plan"} {
		t.Run(command, func(t *testing.T) {
			failure := errors.New("config unavailable")
			root := newRootCommand("test", func() (*config.Config, error) {
				return nil, failure
			})
			quietCommand(root, []string{command})
			require.ErrorIs(t, root.Execute(), failure)
			cmd, _, err := root.Find([]string{command})
			require.NoError(t, err)
			require.Nil(t, cmd.Flags().Lookup("update-gitignore"))
		})
	}
}
