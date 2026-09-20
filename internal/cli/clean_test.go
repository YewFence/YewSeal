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
		{name: "both flags", args: []string{"--remove-different", "--skip-different"}},
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
			require.ErrorContains(t, err, "--remove-different and --skip-different cannot both be true")
		})
	}
}

func TestCleanExplicitFalseOverridesTrueEnvironment(t *testing.T) {
	t.Setenv("YEWSEAL_CLEAN_REMOVE_DIFFERENT", "true")
	t.Setenv("YEWSEAL_CLEAN_SKIP_DIFFERENT", "true")

	cmd := newRootCommand("test", func() (*config.Config, error) {
		return nil, errors.New("config reached")
	})
	cmd.SetArgs([]string{"clean", "--remove-different=false", "--skip-different=false"})
	err := cmd.Execute()
	require.ErrorContains(t, err, "failed to load config: config reached")
	require.NotContains(t, err.Error(), "cannot both be true")
}

func TestCleanRejectsRemovedForceFlag(t *testing.T) {
	cmd := newRootCommand("test", func() (*config.Config, error) {
		t.Fatal("removed flag must fail before config loading")
		return nil, nil
	})
	cmd.SetArgs([]string{"clean", "--force"})
	require.ErrorContains(t, cmd.Execute(), "unknown flag: --force")
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
			require.Contains(t, out.String(), "--remove-different")
			require.Contains(t, out.String(), "--skip-different")
			require.NotContains(t, out.String(), "--force")
		})
	}
}
