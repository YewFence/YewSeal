package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRootDoesNotExposeSync(t *testing.T) {
	for _, args := range [][]string{{"sync"}, {"sync", "pull"}} {
		t.Run(args[len(args)-1], func(t *testing.T) {
			cmd := NewRootCommand("test")
			cmd.SetArgs(args)
			require.ErrorContains(t, cmd.Execute(), `unknown command "sync"`)
		})
	}

	cmd := NewRootCommand("test")
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--help"})
	require.NoError(t, cmd.Execute())
	require.NotContains(t, out.String(), "sync")
}

func TestKeyFileHelpNamesInlineAndExternalIdentitySources(t *testing.T) {
	usage := NewRootCommand("test").PersistentFlags().Lookup("key-file").Usage
	for _, source := range []string{"YEWSEAL_KEY_FILE", "YEWSEAL_AGE_IDENTITIES", "SOPS_AGE_KEY", "SOPS_AGE_KEY_FILE", "SOPS_AGE_KEY_CMD", ".age/keys.txt"} {
		require.Contains(t, usage, source)
	}
}
