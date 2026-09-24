package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"filippo.io/age"
	"github.com/stretchr/testify/require"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/presentation"
)

type failingWriter struct{ err error }

func (w failingWriter) Write([]byte) (int, error) { return 0, w.err }

func prepareIdentitiesProject(t *testing.T, registered bool) {
	t.Helper()
	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	publicKey := identity.Recipient().String()
	if !registered {
		other, err := age.GenerateX25519Identity()
		require.NoError(t, err)
		publicKey = other.Recipient().String()
	}
	require.NoError(t, os.WriteFile(".yewseal.toml", []byte("[recipients.registry]\nowner = '"+publicKey+"'\n"), 0600))
	require.NoError(t, os.MkdirAll(".age", 0700))
	require.NoError(t, os.WriteFile(".age/keys.txt", []byte(identity.String()+"\n"), 0600))
}

func runIdentities(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	cmd := newRootCommand("test", config.LoadConfig)
	var out, diagnostics bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&diagnostics)
	cmd.SetArgs(append([]string{"identities"}, args...))
	err = cmd.Execute()
	return out.String(), diagnostics.String(), err
}

func identitiesReportJSON(t *testing.T, stdout string) (source string, shadowed []string, identities []map[string]any) {
	t.Helper()
	var report struct {
		Command    string           `json:"command"`
		Source     string           `json:"source"`
		Shadowed   []string         `json:"shadowed"`
		Identities []map[string]any `json:"identities"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &report))
	require.Equal(t, "identities", report.Command)
	return report.Source, report.Shadowed, report.Identities
}

func TestIdentitiesJSONReportsSourceAndRegistryAlias(t *testing.T) {
	clearCLIEnvironment(t)
	t.Chdir(t.TempDir())

	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	configContent := "[recipients.registry]\nowner = '" + identity.Recipient().String() + "'\n"
	require.NoError(t, os.WriteFile(".yewseal.toml", []byte(configContent), 0600))
	require.NoError(t, os.MkdirAll(".age", 0700))
	require.NoError(t, os.WriteFile(".age/keys.txt", []byte(identity.String()+"\n"), 0600))

	stdout, _, err := runIdentities(t, "--json")
	require.NoError(t, err)
	source, _, identities := identitiesReportJSON(t, stdout)
	require.Equal(t, "file:.age/keys.txt", source)
	require.Len(t, identities, 1)
	require.Equal(t, identity.Recipient().String(), identities[0]["public_key"])
	require.Equal(t, "owner", identities[0]["alias"])
	require.NotContains(t, identities[0], "secret")
	require.NotContains(t, identities[0], "warning")
}

func TestIdentitiesRevealAddsSecretField(t *testing.T) {
	clearCLIEnvironment(t)
	t.Chdir(t.TempDir())

	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(".yewseal.toml", []byte("[recipients.registry]\nowner = '"+identity.Recipient().String()+"'\n"), 0600))
	t.Setenv("YEWSEAL_AGE_IDENTITIES", identity.String())

	stdout, _, err := runIdentities(t, "--json", "--reveal")
	require.NoError(t, err)
	_, _, identities := identitiesReportJSON(t, stdout)
	require.Equal(t, identity.String(), identities[0]["secret"])
}

func TestIdentitiesWarnsForMalformedBundleItems(t *testing.T) {
	clearCLIEnvironment(t)
	t.Chdir(t.TempDir())

	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(".yewseal.toml", []byte("[recipients.registry]\nowner = '"+identity.Recipient().String()+"'\n"), 0600))
	t.Setenv("YEWSEAL_AGE_IDENTITIES", identity.String()+",INVALID")

	stdout, stderr, err := runIdentities(t, "--json")
	require.NoError(t, err)
	_, _, identities := identitiesReportJSON(t, stdout)
	require.Len(t, identities, 1)
	require.Contains(t, stderr, "WARNING ignored malformed Age identity bundle item at line 1, item 2 (IN…ID)")
	require.NotContains(t, stderr, identity.String())
	require.NotContains(t, stderr, "INVALID")
}

func TestIdentitiesRequiresAProjectConfig(t *testing.T) {
	clearCLIEnvironment(t)
	t.Chdir(t.TempDir())

	_, _, err := runIdentities(t, "--json")
	require.Error(t, err)
	require.Contains(t, err.Error(), "no YewSeal configuration found")
}

func TestIdentitiesUnregisteredPublicKeyWarns(t *testing.T) {
	clearCLIEnvironment(t)
	t.Chdir(t.TempDir())

	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	other, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	configContent := "[recipients.registry]\nowner = '" + other.Recipient().String() + "'\n"
	require.NoError(t, os.WriteFile(".yewseal.toml", []byte(configContent), 0600))
	t.Setenv("YEWSEAL_AGE_IDENTITIES", identity.String())

	stdout, stderr, err := runIdentities(t, "--json")
	require.NoError(t, err)
	_, _, identities := identitiesReportJSON(t, stdout)
	require.Contains(t, identities[0]["warning"], "not registered")
	require.Contains(t, stderr, "warning:")
}

func TestIdentitiesKeyFileShadowsConfiguredSources(t *testing.T) {
	clearCLIEnvironment(t)
	t.Chdir(t.TempDir())

	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	keyPath := filepath.Join(t.TempDir(), "keys.txt")
	require.NoError(t, os.WriteFile(keyPath, []byte(identity.String()+"\n"), 0600))
	require.NoError(t, os.WriteFile(".yewseal.toml", []byte("[recipients.registry]\nowner = '"+identity.Recipient().String()+"'\n"), 0600))
	t.Setenv("YEWSEAL_AGE_IDENTITIES", identity.String())
	require.NoError(t, os.MkdirAll(".age", 0700))
	require.NoError(t, os.WriteFile(".age/keys.txt", []byte(identity.String()+"\n"), 0600))

	stdout, _, err := runIdentities(t, "--json", "--key-file", keyPath)
	require.NoError(t, err)
	source, shadowed, identities := identitiesReportJSON(t, stdout)
	require.Equal(t, "file:"+keyPath, source)
	require.Equal(t, []string{"env:YEWSEAL_AGE_IDENTITIES", "file:.age/keys.txt"}, shadowed)
	require.Len(t, identities, 1)
}

func TestIdentitiesHumanTableAndRevealColumn(t *testing.T) {
	clearCLIEnvironment(t)
	t.Chdir(t.TempDir())

	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	configContent := "[recipients.registry]\nowner = '" + identity.Recipient().String() + "'\n"
	require.NoError(t, os.WriteFile(".yewseal.toml", []byte(configContent), 0600))
	require.NoError(t, os.MkdirAll(".age", 0700))
	require.NoError(t, os.WriteFile(".age/keys.txt", []byte(identity.String()+"\n"), 0600))

	stdout, _, err := runIdentities(t)
	require.NoError(t, err)
	require.Contains(t, stdout, "Source file:.age/keys.txt")
	require.Contains(t, stdout, "Alias")
	require.Contains(t, stdout, "owner")
	require.NotContains(t, stdout, identity.String())

	revealed, _, err := runIdentities(t, "--reveal")
	require.NoError(t, err)
	require.Contains(t, revealed, "Secret")
	require.Contains(t, revealed, identity.String())
}

func TestIdentitiesNoSourcePrintsEmptyReport(t *testing.T) {
	clearCLIEnvironment(t)
	t.Chdir(t.TempDir())
	require.NoError(t, os.WriteFile(".yewseal.toml", []byte("[recipients.registry]\nowner = 'age1r09mha3l82nt25r3kujgkpw4ts60ezntwcj74vnk0t3e9elyu3rswkx08j'\n"), 0600))

	stdout, stderr, err := runIdentities(t, "--json")
	require.NoError(t, err)
	require.Empty(t, stderr)
	source, _, identities := identitiesReportJSON(t, stdout)
	require.Empty(t, source)
	require.Empty(t, identities)
}

func TestIdentitiesMalformedSourcePrintsEmptyReportWithWarning(t *testing.T) {
	clearCLIEnvironment(t)
	t.Chdir(t.TempDir())
	require.NoError(t, os.WriteFile(".yewseal.toml", []byte("[recipients.registry]\nowner = 'age1r09mha3l82nt25r3kujgkpw4ts60ezntwcj74vnk0t3e9elyu3rswkx08j'\n"), 0600))
	t.Setenv("YEWSEAL_AGE_IDENTITIES", "invalid")

	stdout, stderr, err := runIdentities(t, "--json")
	require.NoError(t, err)
	source, _, identities := identitiesReportJSON(t, stdout)
	require.Equal(t, "env:YEWSEAL_AGE_IDENTITIES", source)
	require.Empty(t, identities)
	require.Contains(t, stderr, "WARNING ignored malformed Age identity bundle item")
}

func TestIdentitiesFailsWhenAWarningCannotBeDelivered(t *testing.T) {
	clearCLIEnvironment(t)
	t.Chdir(t.TempDir())
	prepareIdentitiesProject(t, false)

	cmd := newRootCommand("test", config.LoadConfig)
	cmd.SetOut(io.Discard)
	cmd.SetErr(failingWriter{err: errors.New("closed pipe")})
	cmd.SetArgs([]string{"identities", "--json"})

	executed, err := cmd.ExecuteC()
	require.True(t, presentation.DiagnosticsFailed(err), "unexpected error: %v", err)
	require.Equal(t, 1, ExitCode(executed, err))
}

func TestIdentitiesFailsWhenTheReportCannotBeDelivered(t *testing.T) {
	clearCLIEnvironment(t)
	t.Chdir(t.TempDir())
	prepareIdentitiesProject(t, true)

	cmd := newRootCommand("test", config.LoadConfig)
	cmd.SetOut(failingWriter{err: errors.New("no space left")})
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"identities"})

	executed, err := cmd.ExecuteC()
	var outputErr *presentation.OutputError
	require.ErrorAs(t, err, &outputErr)
	require.Equal(t, "content", outputErr.Channel)
	require.Equal(t, 1, ExitCode(executed, err))
}
