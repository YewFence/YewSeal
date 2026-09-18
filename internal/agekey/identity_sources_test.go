package agekey

import (
	"os"
	"path/filepath"
	"testing"

	"filippo.io/age"
	"github.com/stretchr/testify/require"

	"github.com/YewFence/YewSeal/internal/errx"
)

func clearIdentityEnv(t *testing.T) {
	t.Helper()
	for _, name := range []string{"YEWSEAL_AGE_IDENTITIES", "SOPS_AGE_KEY", "SOPS_AGE_KEY_FILE", "YEWSEAL_AGE_KEY_CMD", "SOPS_AGE_KEY_CMD"} {
		t.Setenv(name, "")
	}
}

func TestResolveIdentitySourcesKeyFileWinsAndReportsShadowed(t *testing.T) {
	clearIdentityEnv(t)
	t.Chdir(t.TempDir())

	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	keyPath := filepath.Join(t.TempDir(), "keys.txt")
	require.NoError(t, os.WriteFile(keyPath, []byte(identity.String()+"\n"), 0600))

	t.Setenv("YEWSEAL_AGE_IDENTITIES", identity.String())
	t.Setenv("SOPS_AGE_KEY", identity.String())
	require.NoError(t, os.MkdirAll(".age", 0700))
	require.NoError(t, os.WriteFile(".age/keys.txt", []byte(identity.String()+"\n"), 0600))

	sources, err := ResolveIdentitySources(keyPath)
	require.NoError(t, err)
	require.Equal(t, "file:"+keyPath, sources.Source)
	require.Equal(t, []string{"env:YEWSEAL_AGE_IDENTITIES", "env:SOPS_AGE_KEY", "file:.age/keys.txt"}, sources.Shadowed)
	require.Len(t, sources.Identities, 1)
	require.Equal(t, identity.String(), sources.Identities[0].Secret)
	require.Equal(t, identity.Recipient().String(), sources.Identities[0].PublicKey)
}

func TestResolveIdentitySourcesEnvIdentitiesDerivePublicKey(t *testing.T) {
	clearIdentityEnv(t)
	t.Chdir(t.TempDir())

	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	t.Setenv("YEWSEAL_AGE_IDENTITIES", "# comment\n"+identity.String())

	sources, err := ResolveIdentitySources("")
	require.NoError(t, err)
	require.Equal(t, "env:YEWSEAL_AGE_IDENTITIES", sources.Source)
	require.Empty(t, sources.Shadowed)
	require.Equal(t, identity.Recipient().String(), sources.Identities[0].PublicKey)
}

func TestResolveIdentitySourcesSopsKeyFileMissingFallsThrough(t *testing.T) {
	clearIdentityEnv(t)
	t.Chdir(t.TempDir())

	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	t.Setenv("SOPS_AGE_KEY_FILE", filepath.Join(t.TempDir(), "missing.txt"))
	require.NoError(t, os.MkdirAll(".age", 0700))
	require.NoError(t, os.WriteFile(".age/keys.txt", []byte(identity.String()+"\n"), 0600))

	sources, err := ResolveIdentitySources("")
	require.NoError(t, err)
	require.Equal(t, "file:.age/keys.txt", sources.Source)
	require.Empty(t, sources.Shadowed)
}

func TestResolveIdentitySourcesYewsealKeyCommandLayer(t *testing.T) {
	clearIdentityEnv(t)
	t.Chdir(t.TempDir())

	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	t.Setenv("YEWSEAL_AGE_KEY_CMD", "printf %s "+identity.String())

	sources, err := ResolveIdentitySources("")
	require.NoError(t, err)
	require.Equal(t, "env:YEWSEAL_AGE_KEY_CMD", sources.Source)
	require.Equal(t, identity.Recipient().String(), sources.Identities[0].PublicKey)
}

func TestResolveIdentitySourcesSopsKeyCommandLayer(t *testing.T) {
	clearIdentityEnv(t)
	t.Chdir(t.TempDir())

	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	t.Setenv("SOPS_AGE_KEY_CMD", "printf %s "+identity.String())

	sources, err := ResolveIdentitySources("")
	require.NoError(t, err)
	require.Equal(t, "env:SOPS_AGE_KEY_CMD", sources.Source)
	require.Equal(t, identity.Recipient().String(), sources.Identities[0].PublicKey)
}

func TestResolveIdentitySourcesYewsealKeyCommandShadowsSops(t *testing.T) {
	clearIdentityEnv(t)
	t.Chdir(t.TempDir())

	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	t.Setenv("YEWSEAL_AGE_KEY_CMD", "printf %s "+identity.String())
	t.Setenv("SOPS_AGE_KEY_CMD", "printf not-an-identity")

	sources, err := ResolveIdentitySources("")
	require.NoError(t, err)
	require.Equal(t, "env:YEWSEAL_AGE_KEY_CMD", sources.Source)
	require.Equal(t, []string{"env:SOPS_AGE_KEY_CMD"}, sources.Shadowed)
	require.Equal(t, identity.Recipient().String(), sources.Identities[0].PublicKey)
}

func TestResolveIdentitySourcesNoSourceFails(t *testing.T) {
	clearIdentityEnv(t)
	t.Chdir(t.TempDir())

	_, err := ResolveIdentitySources("")
	var notFound *errx.AgeKeyNotFoundError
	require.ErrorAs(t, err, &notFound)
}

func TestResolveIdentitySourcesDeduplicatesWithinSource(t *testing.T) {
	clearIdentityEnv(t)
	t.Chdir(t.TempDir())

	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	t.Setenv("YEWSEAL_AGE_IDENTITIES", identity.String()+","+identity.String())

	sources, err := ResolveIdentitySources("")
	require.NoError(t, err)
	require.Len(t, sources.Identities, 1)
}

func TestGetIdentityBundleStillResolvesFallbackFile(t *testing.T) {
	clearIdentityEnv(t)
	t.Chdir(t.TempDir())

	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(".age", 0700))
	require.NoError(t, os.WriteFile(".age/keys.txt", []byte("# created: now\n# public key: "+identity.Recipient().String()+"\n"+identity.String()+"\n"), 0600))

	bundle, err := GetIdentityBundle("")
	require.NoError(t, err)
	require.Equal(t, []string{identity.String()}, bundle.Identities())
}
