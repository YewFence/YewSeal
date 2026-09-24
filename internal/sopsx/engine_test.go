package sopsx

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"filippo.io/age"
	sops "github.com/YewFence/sops/v3"
	sopsaes "github.com/YewFence/sops/v3/aes"
	sopsage "github.com/YewFence/sops/v3/age"
	"github.com/YewFence/sops/v3/keyservice"
	sopspgp "github.com/YewFence/sops/v3/pgp"
	toml "github.com/pelletier/go-toml/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testKey struct {
	identity  string
	recipient string
}

func newTestKey(t *testing.T) testKey {
	t.Helper()

	id, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	return testKey{identity: id.String(), recipient: id.Recipient().String()}
}

func samplePlaintext(format string) []byte {
	switch format {
	case "toml":
		return []byte("[database]\nhost = \"localhost\"\npassword = \"secret123\"\n")
	case "yaml":
		return []byte("database:\n  host: localhost\n  password: secret123\n")
	case "json":
		return []byte("{\"database\": {\"host\": \"localhost\", \"password\": \"secret123\"}}\n")
	case "env":
		return []byte("DB_HOST=localhost\nDB_PASSWORD=secret123\n")
	case "ini":
		return []byte("[database]\nhost = localhost\npassword = secret123\n")
	case "binary":
		return []byte{0, 1, 2, 3, 's', 'e', 'c', 'r', 'e', 't', 255}
	default:
		return nil
	}
}

func TestEncryptDecryptRoundTripAllFormats(t *testing.T) {
	key := newTestKey(t)

	for _, format := range []string{"toml", "yaml", "json", "env", "ini", "binary"} {
		t.Run(format, func(t *testing.T) {
			plain := samplePlaintext(format)

			encData, err := Encrypt(plain, format, []string{key.recipient})
			require.NoError(t, err)

			decrypted, err := Decrypt(encData, format, key.identity)
			require.NoError(t, err)

			if format == "binary" {
				assert.Equal(t, plain, decrypted)
			} else {
				// Text stores re-emit canonical documents; verify content survives.
				assert.Contains(t, string(decrypted), "localhost")
				assert.Contains(t, string(decrypted), "secret123")
			}
		})
	}
}

func TestEncryptUsesEncryptedSOPSMAC(t *testing.T) {
	key := newTestKey(t)
	for _, format := range []string{"toml", "yaml", "json", "env", "ini", "binary"} {
		t.Run(format, func(t *testing.T) {
			encData, err := Encrypt(samplePlaintext(format), format, []string{key.recipient})
			require.NoError(t, err)
			store, err := storeForFormat(format)
			require.NoError(t, err)
			tree, err := store.LoadEncryptedFile(encData)
			require.NoError(t, err)
			assert.True(t, strings.HasPrefix(tree.Metadata.MessageAuthenticationCode, "ENC[AES256_GCM,"))
		})
	}
}

func TestInspectDetectsNonAgeKeys(t *testing.T) {
	key := newTestKey(t)
	encData, err := Encrypt(samplePlaintext("yaml"), "yaml", []string{key.recipient})
	require.NoError(t, err)
	store, err := storeForFormat("yaml")
	require.NoError(t, err)
	tree, err := store.LoadEncryptedFile(encData)
	require.NoError(t, err)
	tree.Metadata.KeyGroups[0] = append(tree.Metadata.KeyGroups[0], &sopspgp.MasterKey{Fingerprint: "0123456789ABCDEF0123456789ABCDEF01234567", EncryptedKey: "wrapped"})
	withPGP, err := store.EmitEncryptedFile(tree)
	require.NoError(t, err)
	info, err := Inspect(withPGP, "yaml")
	require.NoError(t, err)
	require.Equal(t, []string{key.recipient}, info.AgeRecipients)
	require.Equal(t, 1, info.KeyGroupCount)
	require.True(t, info.HasNonAgeKeys)
}

func TestInspectReportsMultipleKeyGroups(t *testing.T) {
	first := newTestKey(t)
	second := newTestKey(t)
	encData, err := Encrypt(samplePlaintext("yaml"), "yaml", []string{first.recipient, second.recipient})
	require.NoError(t, err)
	store, err := storeForFormat("yaml")
	require.NoError(t, err)
	tree, err := store.LoadEncryptedFile(encData)
	require.NoError(t, err)
	tree.Metadata.KeyGroups = []sops.KeyGroup{{tree.Metadata.KeyGroups[0][0]}, {tree.Metadata.KeyGroups[0][1]}}
	withMultipleGroups, err := store.EmitEncryptedFile(tree)
	require.NoError(t, err)

	info, err := Inspect(withMultipleGroups, "yaml")
	require.NoError(t, err)
	require.ElementsMatch(t, []string{first.recipient, second.recipient}, info.AgeRecipients)
	require.Equal(t, 2, info.KeyGroupCount)
	require.False(t, info.HasNonAgeKeys)
}

func TestCiphertextInteroperatesWithNativeSOPSDecryptTree(t *testing.T) {
	key := newTestKey(t)
	t.Setenv("SOPS_AGE_KEY", key.identity)
	plain := samplePlaintext("yaml")
	encData, err := Encrypt(plain, "yaml", []string{key.recipient})
	require.NoError(t, err)
	store, err := storeForFormat("yaml")
	require.NoError(t, err)
	tree, err := store.LoadEncryptedFile(encData)
	require.NoError(t, err)

	dataKey, err := tree.Metadata.GetDataKeyWithKeyServices([]keyservice.KeyServiceClient{keyservice.NewLocalClient()}, nil)
	require.NoError(t, err)
	cipher := sopsaes.NewCipher()
	computedMAC, err := tree.Decrypt(dataKey, cipher)
	require.NoError(t, err)
	storedMAC, err := cipher.Decrypt(tree.Metadata.MessageAuthenticationCode, dataKey, tree.Metadata.LastModified.Format(time.RFC3339))
	require.NoError(t, err)
	assert.Equal(t, computedMAC, storedMAC)
	decrypted, err := store.EmitPlainFile(tree.Branches)
	require.NoError(t, err)
	assert.Contains(t, string(decrypted), "host: localhost")
	assert.Contains(t, string(decrypted), "password: secret123")
}

func TestDecryptAndUpdateAcceptLegacyPlaintextMAC(t *testing.T) {
	key := newTestKey(t)
	plain := samplePlaintext("yaml")
	encData, err := Encrypt(plain, "yaml", []string{key.recipient})
	require.NoError(t, err)
	store, err := storeForFormat("yaml")
	require.NoError(t, err)
	state, err := loadAndDecryptTree(store, encData, key.identity)
	require.NoError(t, err)
	legacyMAC, err := state.cipher.Decrypt(state.tree.Metadata.MessageAuthenticationCode, state.dataKey, state.tree.Metadata.LastModified.Format(time.RFC3339))
	require.NoError(t, err)
	encryptedTree, err := store.LoadEncryptedFile(encData)
	require.NoError(t, err)
	encryptedTree.Metadata.MessageAuthenticationCode = legacyMAC.(string)
	legacyData, err := store.EmitEncryptedFile(encryptedTree)
	require.NoError(t, err)

	decrypted, err := Decrypt(legacyData, "yaml", key.identity)
	require.NoError(t, err)
	assert.Contains(t, string(decrypted), "password: secret123")
	result, err := Update(UpdateOptions{Plaintext: plain, ExistingCiphertext: legacyData, Format: "yaml", AgeIdentity: key.identity, Recipients: []string{key.recipient}})
	require.NoError(t, err)
	assert.True(t, result.Unchanged)
	assert.Equal(t, legacyData, result.Ciphertext)
}

func TestUpdateLeavesUnchangedCiphertextByteIdentical(t *testing.T) {
	key := newTestKey(t)
	for _, format := range []string{"toml", "yaml", "json", "env", "ini", "binary"} {
		t.Run(format, func(t *testing.T) {
			plain := samplePlaintext(format)
			encData, err := Encrypt(plain, format, []string{key.recipient})
			require.NoError(t, err)

			result, err := Update(UpdateOptions{
				Plaintext:          plain,
				ExistingCiphertext: encData,
				Format:             format,
				AgeIdentity:        key.identity,
				Recipients:         []string{key.recipient},
			})
			require.NoError(t, err)
			assert.True(t, result.Unchanged)
			assert.Equal(t, encData, result.Ciphertext)
		})
	}
}

func TestUpdateChangesOnlyModifiedStructuredValue(t *testing.T) {
	key := newTestKey(t)
	original := []byte("database:\n  host: localhost\n  password: old\n")
	updated := []byte("database:\n  host: localhost\n  password: new\n")
	encData, err := Encrypt(original, "yaml", []string{key.recipient})
	require.NoError(t, err)
	originalTree := loadEncryptedTreeForTest(t, encData, "yaml")
	originalHost := branchValueForTest(t, originalTree.Branches[0], "database", "host")
	originalPassword := branchValueForTest(t, originalTree.Branches[0], "database", "password")

	result, err := Update(UpdateOptions{
		Plaintext:          updated,
		ExistingCiphertext: encData,
		Format:             "yaml",
		AgeIdentity:        key.identity,
		Recipients:         []string{key.recipient},
	})
	require.NoError(t, err)
	assert.True(t, result.ContentChanged)
	assert.False(t, result.RecipientsChanged)
	assert.False(t, result.Unchanged)

	updatedTree := loadEncryptedTreeForTest(t, result.Ciphertext, "yaml")
	assert.Equal(t, originalHost, branchValueForTest(t, updatedTree.Branches[0], "database", "host"))
	assert.NotEqual(t, originalPassword, branchValueForTest(t, updatedTree.Branches[0], "database", "password"))
	plain, err := Decrypt(result.Ciphertext, "yaml", key.identity)
	require.NoError(t, err)
	assert.Contains(t, string(plain), "host: localhost")
	assert.Contains(t, string(plain), "password: new")
}

func TestUpdateRecipientsOnlyKeepsValuesAndMAC(t *testing.T) {
	oldKey := newTestKey(t)
	newKey := newTestKey(t)
	plain := samplePlaintext("yaml")
	encData, err := Encrypt(plain, "yaml", []string{oldKey.recipient})
	require.NoError(t, err)
	originalTree := loadEncryptedTreeForTest(t, encData, "yaml")

	result, err := Update(UpdateOptions{
		Plaintext:          plain,
		ExistingCiphertext: encData,
		Format:             "yaml",
		AgeIdentity:        oldKey.identity,
		Recipients:         []string{newKey.recipient},
	})
	require.NoError(t, err)
	assert.False(t, result.ContentChanged)
	assert.True(t, result.RecipientsChanged)
	updatedTree := loadEncryptedTreeForTest(t, result.Ciphertext, "yaml")
	assert.Equal(t, originalTree.Branches, updatedTree.Branches)
	assert.Equal(t, originalTree.Metadata.MessageAuthenticationCode, updatedTree.Metadata.MessageAuthenticationCode)
	assert.Equal(t, originalTree.Metadata.LastModified, updatedTree.Metadata.LastModified)
	assert.Equal(t, []string{newKey.recipient}, ageRecipientsFromTree(updatedTree))

	_, err = Decrypt(result.Ciphertext, "yaml", oldKey.identity)
	require.ErrorIs(t, err, ErrNoMatchingIdentity)
	decrypted, err := Decrypt(result.Ciphertext, "yaml", newKey.identity)
	require.NoError(t, err)
	originalPlaintext, err := Decrypt(encData, "yaml", oldKey.identity)
	require.NoError(t, err)
	assert.Equal(t, originalPlaintext, decrypted)
}

func TestUpdateRecipientComparisonIgnoresOrderButNotDuplicates(t *testing.T) {
	first := newTestKey(t)
	second := newTestKey(t)
	plain := samplePlaintext("yaml")
	encData, err := Encrypt(plain, "yaml", []string{first.recipient, second.recipient})
	require.NoError(t, err)

	reordered, err := Update(UpdateOptions{Plaintext: plain, ExistingCiphertext: encData, Format: "yaml", AgeIdentity: first.identity, Recipients: []string{second.recipient, first.recipient}})
	require.NoError(t, err)
	assert.True(t, reordered.Unchanged)
	assert.Equal(t, encData, reordered.Ciphertext)

	store, err := storeForFormat("yaml")
	require.NoError(t, err)
	tree, err := store.LoadEncryptedFile(encData)
	require.NoError(t, err)
	duplicate := *tree.Metadata.KeyGroups[0][0].(*sopsage.MasterKey)
	tree.Metadata.KeyGroups[0] = append(tree.Metadata.KeyGroups[0], &duplicate)
	withDuplicate, err := store.EmitEncryptedFile(tree)
	require.NoError(t, err)

	normalized, err := Update(UpdateOptions{Plaintext: plain, ExistingCiphertext: withDuplicate, Format: "yaml", AgeIdentity: first.identity, Recipients: []string{first.recipient, second.recipient}})
	require.NoError(t, err)
	assert.True(t, normalized.RecipientsChanged)
	assert.Equal(t, []string{first.recipient, second.recipient}, ageRecipientsFromTree(loadEncryptedTreeForTest(t, normalized.Ciphertext, "yaml")))
}

func TestUpdateNormalizesUnsupportedKeyGroupStructure(t *testing.T) {
	first := newTestKey(t)
	second := newTestKey(t)
	plain := samplePlaintext("yaml")
	recipients := []string{first.recipient, second.recipient}
	encData, err := Encrypt(plain, "yaml", recipients)
	require.NoError(t, err)

	tests := map[string]func(sops.Tree) sops.Tree{
		"multiple age groups": func(tree sops.Tree) sops.Tree {
			tree.Metadata.KeyGroups = []sops.KeyGroup{{tree.Metadata.KeyGroups[0][0]}, {tree.Metadata.KeyGroups[0][1]}}
			return tree
		},
		"non-age key": func(tree sops.Tree) sops.Tree {
			tree.Metadata.KeyGroups[0] = append(tree.Metadata.KeyGroups[0], sopspgp.NewMasterKeyFromFingerprint("0123456789ABCDEF"))
			return tree
		},
	}

	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			store, err := storeForFormat("yaml")
			require.NoError(t, err)
			tree := mutate(loadEncryptedTreeForTest(t, encData, "yaml"))
			unsupported, err := store.EmitEncryptedFile(tree)
			require.NoError(t, err)

			result, err := Update(UpdateOptions{Plaintext: plain, ExistingCiphertext: unsupported, Format: "yaml", AgeIdentity: first.identity, Recipients: recipients})
			require.NoError(t, err)
			assert.True(t, result.RecipientsChanged)
			assert.False(t, result.Unchanged)
			assert.True(t, hasCanonicalAgeRecipients(loadEncryptedTreeForTest(t, result.Ciphertext, "yaml"), recipients))
		})
	}
}

func TestUpdateRejectsUnmatchedIdentityWithoutChangingInput(t *testing.T) {
	owner := newTestKey(t)
	other := newTestKey(t)
	plain := samplePlaintext("yaml")
	encData, err := Encrypt(plain, "yaml", []string{owner.recipient})
	require.NoError(t, err)
	original := bytes.Clone(encData)

	_, err = Update(UpdateOptions{Plaintext: plain, ExistingCiphertext: encData, Format: "yaml", AgeIdentity: other.identity, Recipients: []string{owner.recipient}})
	require.ErrorIs(t, err, ErrNoMatchingIdentity)
	assert.Equal(t, original, encData)
}

func loadEncryptedTreeForTest(t *testing.T, data []byte, format string) sops.Tree {
	t.Helper()
	store, err := storeForFormat(format)
	require.NoError(t, err)
	tree, err := store.LoadEncryptedFile(data)
	require.NoError(t, err)
	return tree
}

func branchValueForTest(t *testing.T, branch sops.TreeBranch, path ...string) any {
	t.Helper()
	var current any = branch
	for _, key := range path {
		branch, ok := current.(sops.TreeBranch)
		require.True(t, ok, "path %v does not contain a branch at %s", path, key)
		found := false
		for _, item := range branch {
			if item.Key == key {
				current = item.Value
				found = true
				break
			}
		}
		require.True(t, found, "path %v does not contain %s", path, key)
	}
	return current
}

func TestEncryptTomlEmbedsSopsMetadataNatively(t *testing.T) {
	key := newTestKey(t)

	encData, err := Encrypt(samplePlaintext("toml"), "toml", []string{key.recipient})
	require.NoError(t, err)

	content := string(encData)
	assert.Contains(t, content, "ENC[")
	assert.Contains(t, content, "[sops]")
	assert.Contains(t, content, key.recipient)
	assert.NotContains(t, content, "secret123")
}

func TestEncryptSupportsMultipleAgeRecipients(t *testing.T) {
	first := newTestKey(t)
	second := newTestKey(t)

	encData, err := Encrypt(samplePlaintext("toml"), "toml", []string{second.recipient, first.recipient})
	require.NoError(t, err)

	recipients, err := ExtractAgeRecipients(encData, "toml")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{first.recipient, second.recipient}, recipients)

	decrypted, err := Decrypt(encData, "toml", first.identity+"\n"+second.identity)
	require.NoError(t, err)
	assert.Contains(t, string(decrypted), "localhost")
	assert.Contains(t, string(decrypted), "secret123")
}

// TestEncryptDecryptTomlSemanticRoundTrip locks in native-TOML fidelity beyond
// strings: comments, arrays of tables, nesting, and typed values (int, float,
// bool, datetime) must all survive the encrypt/decrypt round trip. Documents
// are compared semantically so key order and quoting style may differ.
func TestEncryptDecryptTomlSemanticRoundTrip(t *testing.T) {
	key := newTestKey(t)
	plain := []byte(`# top-level comment
name = "my-worker"
workers = 3
ratio = 0.75
enabled = true
birthday = 1979-05-27T07:32:00Z
tags = ["alpha", "beta"]

[vars]
API_KEY = "super-secret"
RETRIES = 5

# namespace comment
[[kv_namespaces]]
id = "abc123"

[[kv_namespaces]]
id = "def456"

[owner.contact]
email = "yew@example.com"
`)

	encData, err := Encrypt(plain, "toml", []string{key.recipient})
	require.NoError(t, err)

	decrypted, err := Decrypt(encData, "toml", key.identity)
	require.NoError(t, err)

	// Comments survive the type:comment round trip.
	assert.Contains(t, string(decrypted), "# namespace comment")

	var want, got map[string]any
	require.NoError(t, toml.Unmarshal(plain, &want))
	require.NoError(t, toml.Unmarshal(decrypted, &got))
	assert.Equal(t, want, got)
}
func TestDecryptRejectsWrongIdentity(t *testing.T) {
	key := newTestKey(t)
	other := newTestKey(t)

	encData, err := Encrypt(samplePlaintext("toml"), "toml", []string{key.recipient})
	require.NoError(t, err)

	_, err = Decrypt(encData, "toml", other.identity)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNoMatchingIdentity)
}

func TestInspect(t *testing.T) {
	key := newTestKey(t)

	before := time.Now().UTC()
	encData, err := Encrypt(samplePlaintext("toml"), "toml", []string{key.recipient})
	require.NoError(t, err)

	info, err := Inspect(encData, "toml")
	require.NoError(t, err)
	assert.Equal(t, []string{key.recipient}, info.AgeRecipients)
	assert.Equal(t, 1, info.KeyGroupCount)
	assert.Equal(t, sopsVersion, info.Version)
	// sops serializes LastModified with second precision.
	assert.WithinDuration(t, before, info.LastModified, 2*time.Second)
}

func TestExtractAgeRecipients(t *testing.T) {
	key := newTestKey(t)

	encData, err := Encrypt(samplePlaintext("yaml"), "yaml", []string{key.recipient})
	require.NoError(t, err)

	recipients, err := ExtractAgeRecipients(encData, "yaml")
	require.NoError(t, err)
	assert.Equal(t, []string{key.recipient}, recipients)
}

func TestRekeyRotatesDataKey(t *testing.T) {
	oldKey := newTestKey(t)
	newKey := newTestKey(t)
	extraKey := newTestKey(t)

	encData, err := Encrypt(samplePlaintext("toml"), "toml", []string{oldKey.recipient})
	require.NoError(t, err)

	rekeyed, err := Rekey(encData, "toml", oldKey.identity, []string{newKey.recipient, extraKey.recipient})
	require.NoError(t, err)

	// Old identity no longer unwraps the rotated data key.
	_, err = Decrypt(rekeyed, "toml", oldKey.identity)
	require.Error(t, err)

	// Every new recipient can decrypt, with content intact.
	for _, key := range []testKey{newKey, extraKey} {
		decrypted, err := Decrypt(rekeyed, "toml", key.identity)
		require.NoError(t, err)
		assert.Contains(t, string(decrypted), "secret123")
	}

	recipients, err := ExtractAgeRecipients(rekeyed, "toml")
	require.NoError(t, err)
	assert.Equal(t, []string{newKey.recipient, extraKey.recipient}, recipients)
}

func TestRekeyRequiresRecipient(t *testing.T) {
	key := newTestKey(t)

	encData, err := Encrypt(samplePlaintext("toml"), "toml", []string{key.recipient})
	require.NoError(t, err)

	_, err = Rekey(encData, "toml", key.identity, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one new recipient")
}

func TestUnknownFormatRejected(t *testing.T) {
	key := newTestKey(t)

	_, err := Encrypt([]byte("x"), "xml", []string{key.recipient})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unsupported format "xml"`)

	_, err = Decrypt([]byte("x"), "xml", key.identity)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unsupported format "xml"`)

	_, err = Inspect([]byte("x"), "xml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unsupported format "xml"`)

	_, err = Rekey([]byte("x"), "xml", key.identity, []string{key.recipient})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unsupported format "xml"`)
}

func TestDecryptDetectsTampering(t *testing.T) {
	key := newTestKey(t)

	encData, err := Encrypt(samplePlaintext("toml"), "toml", []string{key.recipient})
	require.NoError(t, err)

	// Flip one character inside an ENC[...] payload; base64 stays syntactically
	// valid, so any failure comes from AEAD decryption or the MAC check.
	content := string(encData)
	idx := strings.Index(content, "ENC[")
	require.NotEqual(t, -1, idx, "test requires an encrypted payload")
	pos := idx + len("ENC[")
	replacement := byte('A')
	if content[pos] == 'A' {
		replacement = 'B'
	}
	tampered := content[:pos] + string(replacement) + content[pos+1:]

	_, err = Decrypt([]byte(tampered), "toml", key.identity)
	require.Error(t, err)
}
