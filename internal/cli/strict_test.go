package cli

import (
	"testing"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestStrictEnvironmentPrecedence(t *testing.T) {
	for _, tc := range []struct {
		name, env string
		args      []string
		want      bool
		fail      bool
	}{
		{name: "true", env: "true", want: true},
		{name: "false", env: "false"},
		{name: "invalid", env: "invalid", fail: true},
		{name: "empty", env: "", fail: true},
		{name: "explicit-false", env: "true", args: []string{"--strict=false"}},
		{name: "explicit-true", env: "false", args: []string{"--strict"}, want: true},
		{name: "invalid-overridden", env: "invalid", args: []string{"--strict=false"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("YEWSEAL_STRICT", tc.env)
			cmd := &cobra.Command{}
			var strict bool
			cmd.Flags().BoolVar(&strict, "strict", false, "")
			require.NoError(t, cmd.ParseFlags(tc.args))
			err := resolveStrict(cmd, &strict)
			if tc.fail {
				require.ErrorContains(t, err, "invalid YEWSEAL_STRICT")
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.want, strict)
			}
		})
	}
}

func TestInvalidStrictEnvironmentDoesNotBlockInformation(t *testing.T) {
	clearCLIEnvironment(t)
	t.Setenv("YEWSEAL_STRICT", "invalid")
	for _, args := range [][]string{{"--version"}, {"decrypt", "--help"}, {"diff", "--help"}, {"completion", "bash"}, {"init", "--help"}} {
		cmd := newRootCommand("test", func() (*config.Config, error) {
			t.Fatal("information commands must not load configuration")
			return nil, nil
		})
		quietCommand(cmd, args)
		require.NoError(t, cmd.Execute())
	}
	for _, args := range [][]string{{"decrypt"}, {"diff"}} {
		cmd := newRootCommand("test", func() (*config.Config, error) {
			t.Fatal("invalid strict environment must fail before loading configuration")
			return nil, nil
		})
		quietCommand(cmd, args)
		require.ErrorContains(t, cmd.Execute(), "invalid YEWSEAL_STRICT")
	}
}
