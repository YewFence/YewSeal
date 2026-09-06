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

func TestGetIdentityBundleDefaultFileCollectsAllIdentities(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, name := range []string{"YEWSEAL_AGE_IDENTITIES", "SOPS_AGE_KEY", "SOPS_AGE_KEY_FILE", "SOPS_AGE_KEY_CMD"} {
		t.Setenv(name, "")
	}
	first, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	second, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	require.NoError(t, os.Mkdir(".age", 0700))
	require.NoError(t, os.WriteFile(".age/keys.txt", []byte(first.String()+"\n"+second.String()+"\n"), 0600))

	bundle, err := GetIdentityBundle("")
	require.NoError(t, err)
	require.Equal(t, []string{first.String(), second.String()}, bundle.Identities())
}

func TestGetIdentityBundleEnvironmentPrecedesDefaultFile(t *testing.T) {
	t.Chdir(t.TempDir())
	environmentIdentity, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	fallbackIdentity, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	require.NoError(t, os.Mkdir(".age", 0700))
	fallbackPath := ".age/keys.txt"
	require.NoError(t, os.WriteFile(fallbackPath, []byte(fallbackIdentity.String()+"\n"), 0600))
	t.Setenv("YEWSEAL_AGE_IDENTITIES", environmentIdentity.String())
	bundle, err := GetIdentityBundle("")
	require.NoError(t, err)
	require.Equal(t, []string{environmentIdentity.String()}, bundle.Identities())
}

func TestGetIdentityBundleMissingDefaultFile(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, name := range []string{"YEWSEAL_AGE_IDENTITIES", "SOPS_AGE_KEY", "SOPS_AGE_KEY_FILE", "SOPS_AGE_KEY_CMD"} {
		t.Setenv(name, "")
	}
	_, err := GetIdentityBundle("")
	require.ErrorContains(t, err, "no Age key found")
}

func TestGetIdentityBundleExplicitFilePrecedesEnvironment(t *testing.T) {
	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	path := t.TempDir() + "/keys.txt"
	require.NoError(t, os.WriteFile(path, []byte(identity.String()+"\n"), 0600))
	t.Setenv("YEWSEAL_AGE_IDENTITIES", "invalid")
	bundle, err := GetIdentityBundle(path)
	require.NoError(t, err)
	require.Equal(t, []string{identity.String()}, bundle.Identities())
}

func TestGetIdentityBundleSOPSSourcesPrecedeDefaultFile(t *testing.T) {
	for _, source := range []string{"SOPS_AGE_KEY", "SOPS_AGE_KEY_FILE", "SOPS_AGE_KEY_CMD"} {
		t.Run(source, func(t *testing.T) {
			t.Chdir(t.TempDir())
			for _, name := range []string{"YEWSEAL_AGE_IDENTITIES", "SOPS_AGE_KEY", "SOPS_AGE_KEY_FILE", "SOPS_AGE_KEY_CMD"} {
				t.Setenv(name, "")
			}
			first, err := age.GenerateX25519Identity()
			require.NoError(t, err)
			second, err := age.GenerateX25519Identity()
			require.NoError(t, err)
			content := first.String() + "\n" + second.String() + "\n"
			require.NoError(t, os.Mkdir(".age", 0700))
			require.NoError(t, os.WriteFile(".age/keys.txt", []byte("invalid"), 0600))
			switch source {
			case "SOPS_AGE_KEY":
				t.Setenv(source, content)
				t.Setenv("SOPS_AGE_KEY_FILE", "missing.txt")
				t.Setenv("SOPS_AGE_KEY_CMD", "exit 1")
			case "SOPS_AGE_KEY_FILE":
				require.NoError(t, os.WriteFile("identities.txt", []byte(content), 0600))
				t.Setenv(source, "identities.txt")
				t.Setenv("SOPS_AGE_KEY_CMD", "exit 1")
			case "SOPS_AGE_KEY_CMD":
				t.Setenv(source, "echo "+first.String()+" && echo "+second.String())
			}
			bundle, err := GetIdentityBundle("")
			require.NoError(t, err)
			require.Equal(t, []string{first.String(), second.String()}, bundle.Identities())
		})
	}
}
