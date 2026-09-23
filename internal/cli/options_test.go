package cli

import (
	"errors"
	"strings"
	"testing"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
)

type optionResolverFixture struct {
	Force    bool   `mapstructure:"force"`
	Parallel int    `mapstructure:"parallel"`
	Output   string `mapstructure:"output"`
}

func newOptionResolverFixture(t *testing.T, run func(optionResolverFixture, *optionResolver, *cobra.Command) error) *cobra.Command {
	t.Helper()
	opts := optionResolverFixture{Parallel: 1}
	cmd := &cobra.Command{Use: "encrypt"}
	cmd.Flags().BoolVar(&opts.Force, "force", false, "Force")
	cmd.Flags().IntVar(&opts.Parallel, "parallel", opts.Parallel, "Parallel workers")
	cmd.Flags().StringVar(&opts.Output, "output", "", "Output")
	resolver := newOptionResolver(cmd, &opts)
	cmd.Args = resolver.before(cobra.NoArgs)
	cmd.RunE = func(cmd *cobra.Command, _ []string) error { return run(opts, resolver, cmd) }
	return cmd
}

func TestOptionResolverUsesEnvironmentAndTracksPresence(t *testing.T) {
	t.Setenv("YEWSEAL_ENCRYPT_FORCE", "true")
	t.Setenv("YEWSEAL_ENCRYPT_PARALLEL", "4")
	t.Setenv("YEWSEAL_ENCRYPT_OUTPUT", "review.yaml")
	cmd := newOptionResolverFixture(t, func(opts optionResolverFixture, resolver *optionResolver, cmd *cobra.Command) error {
		require.True(t, opts.Force)
		require.Equal(t, 4, opts.Parallel)
		require.Equal(t, "review.yaml", opts.Output)
		require.True(t, resolver.IsSet("output"))
		require.False(t, cmdFlagChanged(cmd, "output"))
		return nil
	})
	require.NoError(t, cmd.Execute())
}

func TestOptionResolverFlagOverridesInvalidEnvironment(t *testing.T) {
	t.Setenv("YEWSEAL_ENCRYPT_FORCE", "invalid")
	cmd := newOptionResolverFixture(t, func(opts optionResolverFixture, _ *optionResolver, _ *cobra.Command) error {
		require.False(t, opts.Force)
		return nil
	})
	cmd.SetArgs([]string{"--force=false"})
	require.NoError(t, cmd.Execute())
}

func TestOptionResolverAggregatesInvalidEnvironment(t *testing.T) {
	t.Setenv("YEWSEAL_ENCRYPT_FORCE", "invalid")
	t.Setenv("YEWSEAL_ENCRYPT_PARALLEL", "invalid")
	cmd := newOptionResolverFixture(t, func(optionResolverFixture, *optionResolver, *cobra.Command) error {
		t.Fatal("invalid environment must fail before RunE")
		return nil
	})
	err := cmd.Execute()
	require.ErrorContains(t, err, "invalid YEWSEAL_ENCRYPT_FORCE: expected a boolean")
	require.ErrorContains(t, err, "invalid YEWSEAL_ENCRYPT_PARALLEL: expected an integer")
}

func TestOptionResolverIgnoresEmptyAndUnknownEnvironment(t *testing.T) {
	t.Setenv("YEWSEAL_ENCRYPT_FORCE", "")
	t.Setenv("YEWSEAL_ENCRYPT_UNKNOWN", "true")
	cmd := newOptionResolverFixture(t, func(opts optionResolverFixture, resolver *optionResolver, _ *cobra.Command) error {
		require.False(t, opts.Force)
		require.Equal(t, 1, opts.Parallel)
		require.False(t, resolver.IsSet("force"))
		return nil
	})
	require.NoError(t, cmd.Execute())
}

func TestEveryEnvironmentEnabledFlagDocumentsItsVariable(t *testing.T) {
	root := newRootCommand("test", nil)
	root.PersistentFlags().VisitAll(func(flag *pflag.Flag) {
		require.Contains(t, flag.Usage, "env YEWSEAL_"+envToken(flag.Name))
	})
	for _, cmd := range root.Commands() {
		if cmd.Name() == "completion" || cmd.Name() == "help" {
			continue
		}
		prefix := commandEnvPrefix(cmd.Name())
		cmd.LocalNonPersistentFlags().VisitAll(func(flag *pflag.Flag) {
			if isCLIOnlyFlag(flag) {
				require.NotContains(t, flag.Usage, "env ", cmd.Name()+" --"+flag.Name)
				return
			}
			expected := envName(prefix, flag.Name)
			if shared, ok := sharedEnvName(flag); ok {
				expected = shared
			}
			require.Contains(t, flag.Usage, "env "+expected, cmd.Name()+" --"+flag.Name)
		})
	}
}

func TestCLIOnlyFlagIgnoresEnvironment(t *testing.T) {
	t.Setenv("YEWSEAL_CLEAN_FORCE", "true")
	opts := struct {
		Force bool `mapstructure:"force"`
	}{}
	cmd := &cobra.Command{Use: "clean"}
	cmd.Flags().BoolVar(&opts.Force, "force", false, "Force")
	markCLIOnlyFlag(cmd.Flags().Lookup("force"))
	resolver := newOptionResolver(cmd, &opts)
	cmd.Args = resolver.before(cobra.NoArgs)
	cmd.RunE = func(*cobra.Command, []string) error {
		require.False(t, opts.Force)
		return nil
	}
	require.NoError(t, cmd.Execute())
}

func TestInvalidCommandEnvironmentPrecedesConfigLoading(t *testing.T) {
	t.Setenv("YEWSEAL_DECRYPT_PARALLEL", "invalid")
	cmd := newRootCommand("test", func() (*config.Config, error) {
		t.Fatal("invalid environment must fail before loading configuration")
		return nil, nil
	})
	quietCommand(cmd, []string{"decrypt"})
	require.ErrorContains(t, cmd.Execute(), "invalid YEWSEAL_DECRYPT_PARALLEL")
}

func TestCommandEnvironmentOutputParticipatesInArgumentValidation(t *testing.T) {
	t.Setenv("YEWSEAL_ENCRYPT_OUTPUT", "review.yaml")
	cmd := newRootCommand("test", func() (*config.Config, error) {
		t.Fatal("invalid option combination must fail before loading configuration")
		return nil, nil
	})
	quietCommand(cmd, []string{"encrypt"})
	require.ErrorContains(t, cmd.Execute(), "--output is only supported with exactly one file target")
}

func TestGlobalOptionEnvironmentIsInherited(t *testing.T) {
	t.Setenv("YEWSEAL_KEY_FILE", "ci-keys.txt")
	opts := struct {
		KeyFile string `mapstructure:"key-file"`
	}{}
	root := &cobra.Command{Use: "yews"}
	root.PersistentFlags().String("key-file", "", "Key file")
	child := &cobra.Command{Use: "decrypt"}
	resolver := newOptionResolver(child, &opts)
	child.Args = resolver.before(cobra.NoArgs)
	child.RunE = func(*cobra.Command, []string) error {
		require.Equal(t, "ci-keys.txt", opts.KeyFile)
		return nil
	}
	root.AddCommand(child)
	root.SetArgs([]string{"decrypt"})
	require.NoError(t, root.Execute())
}

func TestInvalidEnvironmentDoesNotBlockInformationCommands(t *testing.T) {
	t.Setenv("YEWSEAL_DECRYPT_PARALLEL", "invalid")
	for _, args := range [][]string{{"--version"}, {"decrypt", "--help"}, {"completion", "bash"}} {
		cmd := newRootCommand("test", func() (*config.Config, error) {
			t.Fatal("information commands must not load configuration")
			return nil, nil
		})
		quietCommand(cmd, args)
		require.NoError(t, cmd.Execute())
	}
}

func TestInitSyncEnvironmentCanDisableSopsConfig(t *testing.T) {
	clearCLIEnvironment(t)
	t.Chdir(t.TempDir())
	t.Setenv("YEWSEAL_INIT_INPUT", "config.yaml")
	t.Setenv("YEWSEAL_SYNC_SOPS_CONFIG", "false")
	cmd := newRootCommand("test", func() (*config.Config, error) {
		t.Fatal("init must not load project configuration")
		return nil, nil
	})
	quietCommand(cmd, []string{"init"})
	require.NoError(t, cmd.Execute())
	require.FileExists(t, ".yewseal.toml")
	require.NoFileExists(t, ".sops.yaml")
}

func TestInactiveCommandEnvironmentIsIgnored(t *testing.T) {
	t.Setenv("YEWSEAL_DECRYPT_PARALLEL", "invalid")
	failure := errors.New("config read failed")
	cmd := newRootCommand("test", func() (*config.Config, error) { return nil, failure })
	quietCommand(cmd, []string{"plan"})
	err := cmd.Execute()
	require.ErrorIs(t, err, failure)
	require.False(t, strings.Contains(err.Error(), "YEWSEAL_DECRYPT_PARALLEL"))
}

func cmdFlagChanged(cmd *cobra.Command, name string) bool {
	return cmd.Flags().Lookup(name).Changed
}

func TestSyncSOPSConfigUsesOneSharedEnvironmentVariable(t *testing.T) {
	root := newRootCommand("test", nil)
	for _, name := range []string{"init", "encrypt", "verify"} {
		cmd, _, err := root.Find([]string{name})
		require.NoError(t, err)
		usage := cmd.Flags().Lookup("sync-sops-config").Usage
		require.Contains(t, usage, "(env YEWSEAL_SYNC_SOPS_CONFIG)", name)
		require.NotContains(t, usage, "YEWSEAL_"+strings.ToUpper(name)+"_SYNC_SOPS_CONFIG", name)
	}
}
