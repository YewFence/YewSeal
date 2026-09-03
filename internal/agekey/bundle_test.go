package agekey

import (
	"os"
	"testing"

	"filippo.io/age"
	"github.com/stretchr/testify/require"
)

func TestNewIdentityBundleDeduplicatesAndPreservesOrder(t *testing.T) {
	first, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	second, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	bundle, err := NewIdentityBundle([]string{first.String(), first.String(), second.String()})
	require.NoError(t, err)
	require.Equal(t, []string{first.String(), second.String()}, bundle.Identities())
	require.Equal(t, first.String()+"\n"+second.String(), bundle.String())
}

func TestGetIdentityBundleFromKeyFileCollectsValidIdentities(t *testing.T) {
	first, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	second, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	path := t.TempDir() + "/keys.txt"
	content := "# owner\n" + first.String() + "\nnot an identity\n# backup\n" + second.String() + "\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))

	bundle, err := GetIdentityBundle(path)
	require.NoError(t, err)
	require.Equal(t, []string{first.String(), second.String()}, bundle.Identities())
}

func TestGetIdentityBundleFromEnvRejectsEmptyItem(t *testing.T) {
	first, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	t.Setenv("YEWSEAL_AGE_IDENTITIES", first.String()+",,"+first.String())

	_, err = GetIdentityBundle("")
	require.EqualError(t, err, "YEWSEAL_AGE_IDENTITIES item 2 is empty")
}

func TestGetIdentityBundleExplicitFileDoesNotFallBack(t *testing.T) {
	first, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	t.Setenv("YEWSEAL_AGE_IDENTITIES", first.String())

	_, err = GetIdentityBundle(t.TempDir() + "/missing-keys.txt")
	require.Error(t, err)
	require.NotContains(t, err.Error(), first.String())
}

func TestGetIdentityBundleEnvironmentPrecedesConfiguredFallback(t *testing.T) {
	environmentIdentity, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	fallbackIdentity, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	fallbackPath := t.TempDir() + "/keys.txt"
	require.NoError(t, os.WriteFile(fallbackPath, []byte(fallbackIdentity.String()+"\n"), 0600))
	t.Setenv("YEWSEAL_AGE_IDENTITIES", environmentIdentity.String())
	bundle, err := GetIdentityBundleWithFallback("", fallbackPath)
	require.NoError(t, err)
	require.Equal(t, []string{environmentIdentity.String()}, bundle.Identities())
}

func TestGetIdentityBundleUsesConfiguredFallback(t *testing.T) {
	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	path := t.TempDir() + "/keys.txt"
	require.NoError(t, os.WriteFile(path, []byte(identity.String()+"\n"), 0600))
	bundle, err := GetIdentityBundleWithFallback("", path)
	require.NoError(t, err)
	require.Equal(t, []string{identity.String()}, bundle.Identities())
}
