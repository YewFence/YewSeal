package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"filippo.io/age"
	"github.com/stretchr/testify/require"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/errx"
)

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

func TestIdentitiesNoSourceFailsWithUsageError(t *testing.T) {
	clearCLIEnvironment(t)
	t.Chdir(t.TempDir())

	_, _, err := runIdentities(t)
	require.Error(t, err)
	var usage *errx.UsageError
	require.True(t, errors.As(err, &usage))
}
