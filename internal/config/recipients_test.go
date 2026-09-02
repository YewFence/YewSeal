package config

import (
	"testing"

	"filippo.io/age"
	"github.com/stretchr/testify/require"
)

func TestResolveFileRecipientsUsesCanonicalSortedDefaults(t *testing.T) {
	owner, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	ops, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	defaults := []string{"ops", "owner"}
	cfg := &Config{Recipients: RecipientConfig{
		Defaults: &defaults,
		Registry: map[string]string{
			"owner": owner.Recipient().String(),
			"ops":   ops.Recipient().String(),
		},
	}}

	resolved, err := cfg.ResolveFileRecipients(FilePair{PlaintextPath: "config.yaml"})
	require.NoError(t, err)
	require.Equal(t, defaults, resolved.Aliases)
	require.Len(t, resolved.Recipients, 2)
	require.Less(t, resolved.Recipients[0], resolved.Recipients[1])
	require.Equal(t, "defaults", resolved.Provenance.Kind)
}

func TestResolveFileRecipientsRejectsDuplicateAlias(t *testing.T) {
	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	aliases := []string{"owner", "owner"}
	cfg := &Config{Recipients: RecipientConfig{
		Registry: map[string]string{"owner": identity.Recipient().String()},
	}}

	_, err = cfg.ResolveFileRecipients(FilePair{PlaintextPath: "config.yaml", Recipients: &aliases})
	require.EqualError(t, err, `duplicate recipient alias "owner"`)
}

func TestValidateRecipientConfigRejectsDuplicatePublicKey(t *testing.T) {
	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	publicKey := identity.Recipient().String()
	cfg := &Config{Recipients: RecipientConfig{Registry: map[string]string{
		"owner":  publicKey,
		"backup": publicKey,
	}}}

	require.EqualError(t, cfg.ValidateRecipientConfig(), `recipient aliases "backup" and "owner" resolve to the same public key`)
}

func TestValidateRecipientConfigRejectsInvalidAlias(t *testing.T) {
	cfg := &Config{Recipients: RecipientConfig{Registry: map[string]string{
		"prod.ops": "age1invalid",
	}}}

	require.EqualError(t, cfg.ValidateRecipientConfig(), `invalid recipient alias "prod.ops"`)
}
