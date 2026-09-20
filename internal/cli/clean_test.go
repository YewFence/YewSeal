package cli

import (
	"bytes"
	"errors"
	"testing"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/stretchr/testify/require"
)

func TestCleanStrategyFlagsAreMutuallyExclusive(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		args []string
	}{
		{name: "remove and skip flags", args: []string{"--remove-different", "--skip-different"}},
		{name: "force and remove flags", args: []string{"--force", "--remove-different"}},
		{name: "force and skip flags", args: []string{"--force", "--skip-different"}},
		{name: "force with env remove", env: map[string]string{"YEWSEAL_CLEAN_REMOVE_DIFFERENT": "true"}, args: []string{"--force"}},
		{name: "force with env skip", env: map[string]string{"YEWSEAL_CLEAN_SKIP_DIFFERENT": "true"}, args: []string{"--force"}},
		{name: "flag remove with env skip", env: map[string]string{"YEWSEAL_CLEAN_SKIP_DIFFERENT": "true"}, args: []string{"--remove-different"}},
		{name: "env remove with flag skip", env: map[string]string{"YEWSEAL_CLEAN_REMOVE_DIFFERENT": "true"}, args: []string{"--skip-different"}},
		{name: "both env", env: map[string]string{"YEWSEAL_CLEAN_REMOVE_DIFFERENT": "true", "YEWSEAL_CLEAN_SKIP_DIFFERENT": "true"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for name, value := range tc.env {
				t.Setenv(name, value)
			}
			cmd := newRootCommand("test", func() (*config.Config, error) {
				return nil, errors.New("config must not load before strategy validation")
			})
			cmd.SetArgs(append([]string{"clean"}, tc.args...))
			err := cmd.Execute()
			require.ErrorContains(t, err, "--force, --remove-different, and --skip-different are mutually exclusive")
		})
	}
}

func TestCleanExplicitFalseOverridesTrueEnvironment(t *testing.T) {
	t.Setenv("YEWSEAL_CLEAN_REMOVE_DIFFERENT", "true")
	t.Setenv("YEWSEAL_CLEAN_SKIP_DIFFERENT", "true")

	cmd := newRootCommand("test", func() (*config.Config, error) {
		return nil, errors.New("config reached")
	})
	cmd.SetArgs([]string{"clean", "--force=false", "--remove-different=false", "--skip-different=false"})
	err := cmd.Execute()
	require.ErrorContains(t, err, "failed to load config: config reached")
	require.NotContains(t, err.Error(), "cannot both be true")
}

func TestCleanHelpAndAliasDoNotLoadConfig(t *testing.T) {
	for _, args := range [][]string{{"clean", "--help"}, {"help", "clean"}, {"c", "--help"}} {
		t.Run(args[0], func(t *testing.T) {
			var out bytes.Buffer
			cmd := newRootCommand("test", func() (*config.Config, error) {
				t.Fatal("help must not load the project config")
				return nil, nil
			})
			cmd.SetOut(&out)
			cmd.SetErr(&out)
			cmd.SetArgs(args)
			require.NoError(t, cmd.Execute())
			require.Contains(t, out.String(), "--force")
			require.Contains(t, out.String(), "--remove-different")
			require.Contains(t, out.String(), "--skip-different")
			require.NotContains(t, out.String(), "YEWSEAL_CLEAN_FORCE")
		})
	}

	clean, _, err := newRootCommand("test", nil).Find([]string{"clean"})
	require.NoError(t, err)
	force := clean.Flags().Lookup("force")
	require.NotNil(t, force)
	require.Empty(t, force.Shorthand)
	require.True(t, isCLIOnlyFlag(force))
}
