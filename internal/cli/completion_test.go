package cli

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"filippo.io/age"
	"github.com/YewFence/YewSeal/internal/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestRegisteredTargetCompletionSuggestsCommandSide(t *testing.T) {
	clearCLIEnvironment(t)
	root := t.TempDir()
	t.Chdir(root)
	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	defaults := []string{"owner"}
	cfg := &config.Config{
		CurrentDir: root, ProjectRoot: root,
		Recipients: config.RecipientConfig{Defaults: &defaults, Registry: map[string]string{"owner": identity.Recipient().String()}},
		Encryption: config.EncryptionConfig{
			Files: []config.FilePair{{
				PlaintextPath:   filepath.Join(root, "config.toml"),
				EncryptedPath:   filepath.Join(root, "config.enc.toml"),
				Format:          "toml",
				ConfigPath:      filepath.Join(root, ".yewseal.toml"),
				RecipientSource: config.ValueSource{Kind: "defaults"},
			}},
		},
	}

	complete := func(command, toComplete string) ([]string, cobra.ShellCompDirective) {
		cmd := newRootCommand("test", func() (*config.Config, error) { return cfg, nil })
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetErr(io.Discard)
		cmd.SetArgs([]string{cobra.ShellCompRequestCmd, command, toComplete})
		require.NoError(t, cmd.Execute())
		return parseCompletion(t, out.String())
	}

	t.Run("encrypt completes plaintext only", func(t *testing.T) {
		paths, directive := complete("encrypt", "")
		require.Equal(t, []string{"config.toml"}, paths)
		require.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
	})
	t.Run("decrypt completes encrypted only", func(t *testing.T) {
		paths, directive := complete("decrypt", "")
		require.Equal(t, []string{"config.enc.toml"}, paths)
		require.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
	})
	t.Run("plan completes either side", func(t *testing.T) {
		paths, _ := complete("plan", "")
		require.Equal(t, []string{"config.enc.toml", "config.toml"}, paths)
	})
	t.Run("verify completes either side", func(t *testing.T) {
		paths, _ := complete("verify", "")
		require.Equal(t, []string{"config.enc.toml", "config.toml"}, paths)
	})
	t.Run("edit completes encrypted like decrypt and stops after one argument", func(t *testing.T) {
		paths, _ := complete("edit", "")
		require.Equal(t, []string{"config.enc.toml"}, paths)
		cmd := newRootCommand("test", func() (*config.Config, error) { return cfg, nil })
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetErr(io.Discard)
		cmd.SetArgs([]string{cobra.ShellCompRequestCmd, "edit", "config.enc.toml", ""})
		require.NoError(t, cmd.Execute())
		_, directive := parseCompletion(t, out.String())
		require.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
	})
	t.Run("prefix filters matches", func(t *testing.T) {
		paths, _ := complete("decrypt", "config.e")
		require.Equal(t, []string{"config.enc.toml"}, paths)
		paths, directive := complete("decrypt", "missing")
		require.Empty(t, paths)
		require.Equal(t, cobra.ShellCompDirectiveDefault, directive)
	})
	t.Run("dot-slash prefix keeps the user's style", func(t *testing.T) {
		paths, directive := complete("decrypt", "./config.e")
		require.Equal(t, []string{"./config.enc.toml"}, paths)
		require.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
	})
	t.Run("absolute prefix suggests absolute paths", func(t *testing.T) {
		prefix := root + string(filepath.Separator)
		paths, directive := complete("plan", prefix)
		require.Equal(t, []string{prefix + "config.enc.toml", prefix + "config.toml"}, paths)
		require.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
		paths, _ = complete("decrypt", filepath.Join(root, "config.e"))
		require.Equal(t, []string{filepath.Join(root, "config.enc.toml")}, paths)
	})
	t.Run("absolute prefix that matches nothing falls back", func(t *testing.T) {
		paths, directive := complete("decrypt", filepath.Join(root, "no-such"))
		require.Empty(t, paths)
		require.Equal(t, cobra.ShellCompDirectiveDefault, directive)
	})
	t.Run("dot-slash prefix never suggests absolute registrations", func(t *testing.T) {
		outside := t.TempDir()
		cfg.Encryption.Files = append(cfg.Encryption.Files, config.FilePair{
			PlaintextPath:  filepath.Join(outside, "config.toml"),
			EncryptedPath:  filepath.Join(outside, "config.enc.toml"),
			Format:         "toml",
		})
		paths, _ := complete("decrypt", "./")
		require.Equal(t, []string{"./config.enc.toml"}, paths)
		prefix := outside + string(filepath.Separator)
		paths, directive := complete("decrypt", prefix)
		require.Equal(t, []string{prefix + "config.enc.toml"}, paths)
		require.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
	})
	t.Run("dot-slash prefix without a match falls back", func(t *testing.T) {
		paths, directive := complete("decrypt", "./no-such-prefix")
		require.Empty(t, paths)
		require.Equal(t, cobra.ShellCompDirectiveDefault, directive)
	})
	t.Run("completion path errors fall back to shell default", func(t *testing.T) {
		first, err := age.GenerateX25519Identity()
		require.NoError(t, err)
		second, err := age.GenerateX25519Identity()
		require.NoError(t, err)
		firstAliases, secondAliases := []string{"first"}, []string{"second"}
		require.NoError(t, os.WriteFile(filepath.Join(root, "config.yaml"), []byte("key: value\n"), 0600))
		cfg.Encryption.Groups = []config.GroupConfig{
			{Patterns: []string{"config.yaml"}, Recipients: &firstAliases, ConfigDir: root},
			{Patterns: []string{"config.yaml"}, Recipients: &secondAliases, ConfigDir: root},
		}
		cfg.Recipients.Registry["first"] = first.Recipient().String()
		cfg.Recipients.Registry["second"] = second.Recipient().String()
		paths, directive := complete("encrypt", "")
		require.Empty(t, paths)
		require.Equal(t, cobra.ShellCompDirectiveDefault, directive)
	})
}

func TestRegisteredTargetCompletionFallsBackWithoutConfig(t *testing.T) {
	clearCLIEnvironment(t)
	t.Chdir(t.TempDir())
	cmd := newRootCommand("test", func() (*config.Config, error) {
		return nil, errors.New("no YewSeal configuration found")
	})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{cobra.ShellCompRequestCmd, "encrypt", ""})
	require.NoError(t, cmd.Execute())
	_, directive := parseCompletion(t, out.String())
	require.Equal(t, cobra.ShellCompDirectiveDefault, directive)
}

func parseCompletion(t *testing.T, output string) ([]string, cobra.ShellCompDirective) {
	t.Helper()
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	require.GreaterOrEqual(t, len(lines), 1)
	last := lines[len(lines)-1]
	require.True(t, strings.HasPrefix(last, ":"), "unexpected completion output: %q", output)
	directive, err := strconv.Atoi(last[1:])
	require.NoError(t, err)
	return lines[:len(lines)-1], cobra.ShellCompDirective(directive)
}
