package cli

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func clearCLIEnvironment(t *testing.T) {
	t.Helper()
	for _, name := range []string{"AGE_KEY_FILE", "SOPS_OUTPUT_FILE", "YEWSEAL_FORMAT", "SOPS_FORMAT"} {
		t.Setenv(name, "")
		require.NoError(t, os.Unsetenv(name))
	}
}

func quietCommand(cmd *cobra.Command, args []string) {
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs(append([]string{}, args...))
}

func TestInformationCommandsNeverLoadConfig(t *testing.T) {
	clearCLIEnvironment(t)
	for _, args := range [][]string{
		{}, {"--version"}, {"--help"}, {"help", "view"},
		{"completion", "bash"}, {"completion", "zsh"}, {"completion", "fish"}, {"completion", "powershell"},
		{"__complete", ""}, {"__complete", "decrypt", "--"},
		{"encrypt", "--help"}, {"decrypt", "--help", "--format", "invalid"},
		{"plan", "--help"}, {"edit", "--help"}, {"view", "--help"}, {"diff", "--help"}, {"init", "--help"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			calls := 0
			cmd := newRootCommand("test", func() (*config.Config, error) {
				calls++
				return nil, errors.New("must not load")
			})
			require.Zero(t, calls, "constructing the command must not load config")
			quietCommand(cmd, args)
			require.NoError(t, cmd.Execute())
			require.Zero(t, calls)
		})
	}
}

func TestInvalidArgumentsNeverLoadConfig(t *testing.T) {
	clearCLIEnvironment(t)
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"--version", "--unknown-option"}, "unknown flag"},
		{[]string{"decrypt", "--help", "--parallel", "bad"}, "invalid argument"},
		{[]string{"encrypt", "one", "two"}, "accepts at most 1 arg"},
		{[]string{"decrypt", "one", "two"}, "accepts at most 1 arg"},
		{[]string{"plan", "one", "two"}, "accepts at most 1 arg"},
		{[]string{"diff", "one", "two"}, "accepts at most 1 arg"},
		{[]string{"view"}, "accepts 1 arg"},
		{[]string{"view", " "}, "requires exactly one target"},
		{[]string{"edit"}, "edit requires exactly one configured target"},
		{[]string{"edit", "extra", "--file", "secret.yaml"}, "unknown command"},
		{[]string{"encrypt", "--format", "bad"}, "unsupported format"},
		{[]string{"decrypt", "--format", "bad"}, "unsupported format"},
		{[]string{"plan", "--format", "bad"}, "unsupported format"},
		{[]string{"view", "secret.yaml", "--format", "bad"}, "unsupported format"},
		{[]string{"diff", "--format", "bad"}, "unsupported format"},
		{[]string{"diff", "--color", "bad"}, "unsupported color mode"},
		{[]string{"encrypt", "--parallel", "0"}, "--parallel must be at least 1"},
		{[]string{"decrypt", "--parallel", "-1"}, "--parallel must be at least 1"},
		{[]string{"plan", "--parallel", "0"}, "--parallel must be at least 1"},
		{[]string{"decrypt", "--pattern", "!"}, "missing pattern after negation"},
		{[]string{"encrypt", "--output", "out.yaml"}, "--output is only supported"},
		{[]string{"decrypt", "--format", "yaml"}, "--format is only supported"},
		{[]string{"plan", "--format", "yaml"}, "--format is only supported"},
		{[]string{"diff", "--format", "yaml"}, "--format is only supported"},
	} {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			calls := 0
			cmd := newRootCommand("test", func() (*config.Config, error) {
				calls++
				return nil, errors.New("must not load")
			})
			quietCommand(cmd, tc.args)
			require.ErrorContains(t, cmd.Execute(), tc.want)
			require.Zero(t, calls)
		})
	}
}

func TestBusinessCommandsLoadConfigOncePerExecution(t *testing.T) {
	clearCLIEnvironment(t)
	for _, args := range [][]string{
		{"encrypt"}, {"e"}, {"decrypt"}, {"d"}, {"plan"},
		{"edit", "--file", "secret.enc.yaml"}, {"view", "secret.enc.yaml"}, {"diff"},
	} {
		t.Run(args[0], func(t *testing.T) {
			calls := 0
			failure := errors.New("config read failed")
			cmd := newRootCommand("test", func() (*config.Config, error) {
				calls++
				return nil, failure
			})
			require.Zero(t, calls)
			quietCommand(cmd, args)
			for execution := 1; execution <= 2; execution++ {
				err := cmd.Execute()
				require.ErrorIs(t, err, failure)
				require.ErrorContains(t, err, "failed to load config")
				require.Equal(t, execution, calls)
			}
		})
	}
}

func TestConfigLoadingDistinguishesMissingFromEmpty(t *testing.T) {
	clearCLIEnvironment(t)
	t.Chdir(t.TempDir())
	require.NoError(t, os.Mkdir(".git", 0755))
	for _, args := range [][]string{
		{"encrypt"}, {"decrypt"}, {"plan"}, {"view", "secret.enc.yaml"}, {"edit", "--file", "secret.enc.yaml"}, {"diff"},
	} {
		cmd := NewRootCommand("test")
		quietCommand(cmd, args)
		require.ErrorContains(t, cmd.Execute(), "no YewSeal configuration found")
	}

	require.NoError(t, os.WriteFile(".yewseal.toml", nil, 0600))
	cmd := NewRootCommand("test")
	quietCommand(cmd, []string{"plan"})
	err := cmd.Execute()
	require.ErrorContains(t, err, "no configured file pairs selected")
	require.NotContains(t, err.Error(), "failed to load config")
}

func TestSuccessfulConfigLoadIsNotCached(t *testing.T) {
	clearCLIEnvironment(t)
	t.Chdir(t.TempDir())
	require.NoError(t, os.WriteFile(".yewseal.toml", nil, 0600))
	calls := 0
	cmd := newRootCommand("test", func() (*config.Config, error) {
		calls++
		return config.LoadConfig()
	})
	quietCommand(cmd, []string{"plan"})
	require.ErrorContains(t, cmd.Execute(), "no configured file pairs selected")
	require.Equal(t, 1, calls)
	require.NoError(t, os.WriteFile(".yewseal.toml", []byte("[broken"), 0600))
	require.ErrorContains(t, cmd.Execute(), "failed to parse config file")
	require.Equal(t, 2, calls)
}

func TestInitDoesNotLoadOrSilentlyOverwriteBrokenConfig(t *testing.T) {
	clearCLIEnvironment(t)
	t.Chdir(t.TempDir())
	broken := []byte("[broken")
	require.NoError(t, os.WriteFile(".yewseal.toml", broken, 0600))
	require.NoError(t, os.WriteFile("config.yaml", []byte("token: value\n"), 0600))
	load := func() (*config.Config, error) {
		t.Fatal("init must not load project configuration")
		return nil, nil
	}
	args := []string{"init", "--input", "config.yaml", "--output", "config.enc.yaml", "--skip-sops-config"}
	cmd := newRootCommand("test", load)
	quietCommand(cmd, args)
	require.ErrorContains(t, cmd.Execute(), "already exists, use --force to overwrite")
	content, err := os.ReadFile(".yewseal.toml")
	require.NoError(t, err)
	require.Equal(t, broken, content)
	require.NoFileExists(t, ".age/keys.txt")

	cmd = newRootCommand("test", load)
	quietCommand(cmd, append(args, "--force"))
	require.NoError(t, cmd.Execute())
	_, err = config.LoadConfig()
	require.NoError(t, err)
	require.FileExists(t, ".age/keys.txt")
}
