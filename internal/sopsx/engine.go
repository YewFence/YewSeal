// Package sopsx is YewSeal's encryption engine facade. It wraps the sops
// library (the github.com/YewFence/sops/v3 fork, which adds a native TOML
// store) behind a small, stable API so that the rest of the codebase never
// touches sops types. All functions accept YewSeal format names:
// "toml", "yaml", "json", "env", "ini", "binary".
package sopsx

import (
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"filippo.io/age"
	sops "github.com/YewFence/sops/v3"
	"github.com/YewFence/sops/v3/aes"
	sopsage "github.com/YewFence/sops/v3/age"
	sopsconfig "github.com/YewFence/sops/v3/config"
	"github.com/YewFence/sops/v3/stores/dotenv"
	"github.com/YewFence/sops/v3/stores/ini"
	"github.com/YewFence/sops/v3/stores/json"
	"github.com/YewFence/sops/v3/stores/toml"
	"github.com/YewFence/sops/v3/stores/yaml"
)

// sopsVersion matches the fork's version.Version. It is hardcoded because
// importing the version package would pull CLI-only dependencies (urfave/cli).
const sopsVersion = "3.13.3"

// Info describes an encrypted file's sops metadata, readable without a key.
type Info struct {
	AgeRecipients []string
	LastModified  time.Time
	Version       string
}

// storeForFormat maps YewSeal format names to sops store implementations.
// Unknown formats are an error; the caller is expected to validate first.
func storeForFormat(format string) (sops.Store, error) {
	switch format {
	case "toml":
		return toml.NewStore(&sopsconfig.TOMLStoreConfig{}), nil
	case "yaml":
		return &yaml.Store{}, nil
	case "json":
		return &json.Store{}, nil
	case "env":
		return &dotenv.Store{}, nil
	case "ini":
		return &ini.Store{}, nil
	case "binary":
		return json.NewBinaryStore(&sopsconfig.JSONBinaryStoreConfig{}), nil
	default:
		return nil, fmt.Errorf("unsupported format %q (supported: toml, yaml, json, env, ini, binary)", format)
	}
}

// Encrypt encrypts plain data for one or more age recipients.
func Encrypt(plainData []byte, format string, ageRecipients []string) ([]byte, error) {
	store, err := storeForFormat(format)
	if err != nil {
		return nil, err
	}
	if len(ageRecipients) == 0 {
		return nil, fmt.Errorf("at least one age recipient is required")
	}

	// Load plain data into tree branches
	branches, err := store.LoadPlainFile(plainData)
	if err != nil {
		return nil, fmt.Errorf("failed to load plain data: %w", err)
	}

	masterKeys := make(sops.KeyGroup, 0, len(ageRecipients))
	for _, recipient := range ageRecipients {
		masterKey, err := sopsage.MasterKeyFromRecipient(recipient)
		if err != nil {
			return nil, fmt.Errorf("failed to create age master key: %w", err)
		}
		masterKeys = append(masterKeys, masterKey)
	}

	// Generate random 32-byte data key
	dataKey := make([]byte, 32)
	if _, err := rand.Read(dataKey); err != nil {
		return nil, fmt.Errorf("failed to generate data key: %w", err)
	}

	// Encrypt data key for every recipient.
	for _, masterKey := range masterKeys {
		if err := masterKey.Encrypt(dataKey); err != nil {
			return nil, fmt.Errorf("failed to encrypt data key with age: %w", err)
		}
	}

	// Build tree with metadata
	tree := sops.Tree{
		Branches: branches,
		Metadata: sops.Metadata{
			KeyGroups: []sops.KeyGroup{masterKeys},
			Version:   sopsVersion,
		},
	}

	// Encrypt tree values
	mac, err := tree.Encrypt(dataKey, aes.NewCipher())
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt tree: %w", err)
	}

	tree.Metadata.MessageAuthenticationCode = mac
	tree.Metadata.LastModified = time.Now().UTC()

	// Emit encrypted file
	encData, err := store.EmitEncryptedFile(tree)
	if err != nil {
		return nil, fmt.Errorf("failed to emit encrypted file: %w", err)
	}

	return encData, nil
}

// Decrypt decrypts encrypted data with an age identity
// (AGE-SECRET-KEY-...) and verifies the MAC.
func Decrypt(encData []byte, format, ageIdentity string) ([]byte, error) {
	store, err := storeForFormat(format)
	if err != nil {
		return nil, err
	}

	tree, err := loadAndDecryptTree(store, encData, ageIdentity)
	if err != nil {
		return nil, err
	}

	// Emit plain file
	plainData, err := store.EmitPlainFile(tree.Branches)
	if err != nil {
		return nil, fmt.Errorf("failed to emit plain file: %w", err)
	}

	return plainData, nil
}

// Inspect reads an encrypted file's metadata without requiring a private key.
func Inspect(encData []byte, format string) (Info, error) {
	store, err := storeForFormat(format)
	if err != nil {
		return Info{}, err
	}

	tree, err := store.LoadEncryptedFile(encData)
	if err != nil {
		return Info{}, fmt.Errorf("failed to load encrypted file: %w", err)
	}

	return Info{
		AgeRecipients: ageRecipientsFromTree(tree),
		LastModified:  tree.Metadata.LastModified,
		Version:       tree.Metadata.Version,
	}, nil
}

// ExtractAgeRecipients returns all age recipients of an encrypted file.
func ExtractAgeRecipients(encData []byte, format string) ([]string, error) {
	info, err := Inspect(encData, format)
	if err != nil {
		return nil, err
	}
	if len(info.AgeRecipients) == 0 {
		return nil, fmt.Errorf("no age recipient found in encrypted file metadata")
	}
	return info.AgeRecipients, nil
}

// Rekey re-encrypts encData for newRecipients. The data key is rotated: the
// old identity unwraps the current data key, all values are re-encrypted with
// a fresh data key, and the fresh key is wrapped for every new recipient.
func Rekey(encData []byte, format, ageIdentity string, newRecipients []string) ([]byte, error) {
	if len(newRecipients) == 0 {
		return nil, fmt.Errorf("at least one new recipient is required")
	}

	store, err := storeForFormat(format)
	if err != nil {
		return nil, err
	}

	// Decrypt values in place with the old data key (also verifies the MAC)
	tree, err := loadAndDecryptTree(store, encData, ageIdentity)
	if err != nil {
		return nil, err
	}

	// Wrap a fresh data key for every new recipient
	dataKey := make([]byte, 32)
	if _, err := rand.Read(dataKey); err != nil {
		return nil, fmt.Errorf("failed to generate data key: %w", err)
	}
	group := make(sops.KeyGroup, 0, len(newRecipients))
	for _, recipient := range newRecipients {
		masterKey, err := sopsage.MasterKeyFromRecipient(recipient)
		if err != nil {
			return nil, fmt.Errorf("failed to create age master key for recipient %q: %w", recipient, err)
		}
		if err := masterKey.Encrypt(dataKey); err != nil {
			return nil, fmt.Errorf("failed to encrypt data key with age: %w", err)
		}
		group = append(group, masterKey)
	}

	// Re-encrypt the decrypted values with the fresh data key
	mac, err := tree.Encrypt(dataKey, aes.NewCipher())
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt tree: %w", err)
	}

	tree.Metadata.KeyGroups = []sops.KeyGroup{group}
	tree.Metadata.MessageAuthenticationCode = mac
	tree.Metadata.LastModified = time.Now().UTC()

	out, err := store.EmitEncryptedFile(*tree)
	if err != nil {
		return nil, fmt.Errorf("failed to emit encrypted file: %w", err)
	}
	return out, nil
}

// loadAndDecryptTree loads an encrypted file and decrypts its values in place
// using the age identity, verifying the MAC before returning.
func loadAndDecryptTree(store sops.Store, encData []byte, ageIdentity string) (*sops.Tree, error) {
	tree, err := store.LoadEncryptedFile(encData)
	if err != nil {
		return nil, fmt.Errorf("failed to load encrypted file: %w", err)
	}
	if tree.Metadata.MessageAuthenticationCode == "" {
		return nil, fmt.Errorf("encrypted file is missing its MAC")
	}

	// Parse age identity and decrypt data key
	var identities sopsage.ParsedIdentities
	if err := identities.Import(ageIdentity); err != nil {
		return nil, fmt.Errorf("failed to parse age identity: %w", err)
	}

	dataKey, err := decryptTreeDataKey(&tree, identities)
	if err != nil {
		return nil, err
	}

	// Decrypt tree values
	mac, err := tree.Decrypt(dataKey, aes.NewCipher())
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt tree: %w", err)
	}

	// Verify MAC
	originalMAC := tree.Metadata.MessageAuthenticationCode
	if originalMAC != "" && originalMAC != mac {
		return nil, fmt.Errorf("MAC mismatch: file may have been tampered with")
	}

	return &tree, nil
}

// decryptTreeDataKey iterates age master keys in metadata, injects identities,
// and attempts to decrypt the data key. Thread-safe (no global state).
func decryptTreeDataKey(tree *sops.Tree, identities sopsage.ParsedIdentities) ([]byte, error) {
	var attempts []error
	onlyUnmatched := true
	for _, group := range tree.Metadata.KeyGroups {
		for _, key := range group {
			ageMK, ok := key.(*sopsage.MasterKey)
			if !ok {
				onlyUnmatched = false
				attempts = append(attempts, fmt.Errorf("unsupported non-age data key"))
				continue
			}
			recipient, err := age.ParseX25519Recipient(ageMK.Recipient)
			if err != nil {
				onlyUnmatched = false
				attempts = append(attempts, fmt.Errorf("invalid or unsupported age recipient: %w", err))
				continue
			}
			identities.ApplyToMasterKey(ageMK)
			dataKey, err := ageMK.Decrypt()
			if err == nil {
				return dataKey, nil
			}
			attempts = append(attempts, err)
			var unmatched *age.NoIdentityMatchError
			if !errors.As(err, &unmatched) {
				onlyUnmatched = false
			}
			// A failed unwrap for an advertised local recipient may be corruption.
			for _, identity := range identities {
				x25519, ok := identity.(*age.X25519Identity)
				if !ok || x25519.Recipient().String() == recipient.String() {
					onlyUnmatched = false
				}
			}
		}
	}
	if len(attempts) == 0 {
		return nil, fmt.Errorf("encrypted file has no usable age data keys")
	}
	if onlyUnmatched {
		return nil, fmt.Errorf("%w: %w", ErrNoMatchingIdentity, errors.Join(attempts...))
	}
	return nil, fmt.Errorf("failed to decrypt data key: %w", errors.Join(attempts...))
}

// ageRecipientsFromTree collects all age recipients from the file metadata.
func ageRecipientsFromTree(tree sops.Tree) []string {
	var recipients []string
	for _, group := range tree.Metadata.KeyGroups {
		for _, key := range group {
			ageMK, ok := key.(*sopsage.MasterKey)
			if !ok {
				continue
			}
			if ageMK.Recipient != "" {
				recipients = append(recipients, ageMK.Recipient)
			}
		}
	}
	return recipients
}
