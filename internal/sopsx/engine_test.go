package sopsx

import (
	"strings"
	"testing"
	"time"

	"filippo.io/age"
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

			encData, err := Encrypt(plain, format, key.recipient)
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

func TestEncryptTomlEmbedsSopsMetadataNatively(t *testing.T) {
	key := newTestKey(t)

	encData, err := Encrypt(samplePlaintext("toml"), "toml", key.recipient)
	require.NoError(t, err)

	content := string(encData)
	assert.Contains(t, content, "ENC[")
	assert.Contains(t, content, "[sops]")
	assert.Contains(t, content, key.recipient)
	assert.NotContains(t, content, "secret123")
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

	encData, err := Encrypt(plain, "toml", key.recipient)
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

	encData, err := Encrypt(samplePlaintext("toml"), "toml", key.recipient)
	require.NoError(t, err)

	_, err = Decrypt(encData, "toml", other.identity)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no matching age key found")
}

func TestInspect(t *testing.T) {
	key := newTestKey(t)

	before := time.Now().UTC()
	encData, err := Encrypt(samplePlaintext("toml"), "toml", key.recipient)
	require.NoError(t, err)

	info, err := Inspect(encData, "toml")
	require.NoError(t, err)
	assert.Equal(t, []string{key.recipient}, info.AgeRecipients)
	assert.Equal(t, sopsVersion, info.Version)
	// sops serializes LastModified with second precision.
	assert.WithinDuration(t, before, info.LastModified, 2*time.Second)
}

func TestExtractAgeRecipients(t *testing.T) {
	key := newTestKey(t)

	encData, err := Encrypt(samplePlaintext("yaml"), "yaml", key.recipient)
	require.NoError(t, err)

	recipients, err := ExtractAgeRecipients(encData, "yaml")
	require.NoError(t, err)
	assert.Equal(t, []string{key.recipient}, recipients)
}

func TestRekeyRotatesDataKey(t *testing.T) {
	oldKey := newTestKey(t)
	newKey := newTestKey(t)
	extraKey := newTestKey(t)

	encData, err := Encrypt(samplePlaintext("toml"), "toml", oldKey.recipient)
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

	encData, err := Encrypt(samplePlaintext("toml"), "toml", key.recipient)
	require.NoError(t, err)

	_, err = Rekey(encData, "toml", key.identity, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one new recipient")
}

func TestUnknownFormatRejected(t *testing.T) {
	key := newTestKey(t)

	_, err := Encrypt([]byte("x"), "xml", key.recipient)
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

	encData, err := Encrypt(samplePlaintext("toml"), "toml", key.recipient)
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
