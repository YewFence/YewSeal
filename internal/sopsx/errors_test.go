package sopsx

import (
	"testing"

	sopsage "github.com/YewFence/sops/v3/age"
	"github.com/stretchr/testify/require"
)

func TestUnmatchedIdentityClassification(t *testing.T) {
	owner, other := newTestKey(t), newTestKey(t)
	for _, format := range []string{"toml", "yaml", "json", "env", "ini", "binary"} {
		t.Run(format, func(t *testing.T) {
			data, err := Encrypt(samplePlaintext(format), format, []string{owner.recipient})
			require.NoError(t, err)
			_, err = Decrypt(data, format, other.identity)
			require.ErrorIs(t, err, ErrNoMatchingIdentity)
		})
	}
}

func TestDataKeyErrorsAreNotUnmatchedIdentities(t *testing.T) {
	owner, other := newTestKey(t), newTestKey(t)
	for _, mutation := range []string{"invalid-key", "advertised-match", "missing-mac", "no-keys", "invalid-recipient", "changed-value"} {
		t.Run(mutation, func(t *testing.T) {
			data, err := Encrypt([]byte("token: value\n"), "yaml", []string{owner.recipient})
			require.NoError(t, err)
			store, err := storeForFormat("yaml")
			require.NoError(t, err)
			tree, err := store.LoadEncryptedFile(data)
			require.NoError(t, err)
			key := tree.Metadata.KeyGroups[0][0].(*sopsage.MasterKey)
			identity := other.identity
			switch mutation {
			case "invalid-key":
				key.EncryptedKey = "not an age envelope"
			case "advertised-match":
				key.Recipient = other.recipient
			case "missing-mac":
				tree.Metadata.MessageAuthenticationCode = ""
			case "no-keys":
				tree.Metadata.KeyGroups = nil
			case "invalid-recipient":
				key.Recipient = "invalid"
			case "changed-value":
				tree.Branches[0][0].Value = "changed"
				identity = owner.identity
			}
			data, err = store.EmitEncryptedFile(tree)
			require.NoError(t, err)
			_, err = Decrypt(data, "yaml", identity)
			require.Error(t, err)
			require.NotErrorIs(t, err, ErrNoMatchingIdentity)
		})
	}
}

func TestUsableRecipientSurvivesAnotherBrokenEnvelope(t *testing.T) {
	owner, other := newTestKey(t), newTestKey(t)
	data, err := Encrypt([]byte("token: value\n"), "yaml", []string{other.recipient, owner.recipient})
	require.NoError(t, err)
	store, err := storeForFormat("yaml")
	require.NoError(t, err)
	tree, err := store.LoadEncryptedFile(data)
	require.NoError(t, err)
	tree.Metadata.KeyGroups[0][0].(*sopsage.MasterKey).EncryptedKey = "broken"
	data, err = store.EmitEncryptedFile(tree)
	require.NoError(t, err)
	plain, err := Decrypt(data, "yaml", owner.identity)
	require.NoError(t, err)
	require.Equal(t, "token: value\n", string(plain))
}
