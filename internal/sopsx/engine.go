// Package sopsx is YewSeal's encryption engine facade. It wraps the sops
// library (the github.com/YewFence/sops/v3 fork, which adds a native TOML
// store) behind a small, stable API so that the rest of the codebase never
// touches sops types. All functions accept YewSeal format names:
// "toml", "yaml", "json", "env", "ini", "binary".
package sopsx

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
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

// UpdateOptions describes a reconciliation of plaintext and an existing SOPS file.
type UpdateOptions struct {
	Plaintext          []byte
	ExistingCiphertext []byte
	Format             string
	AgeIdentity        string
	Recipients         []string
}

// UpdateResult reports whether Update emitted replacement ciphertext.
type UpdateResult struct {
	Ciphertext        []byte
	ContentChanged    bool
	RecipientsChanged bool
	Unchanged         bool
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

	// Generate random 32-byte data key
	dataKey := make([]byte, 32)
	if _, err := rand.Read(dataKey); err != nil {
		return nil, fmt.Errorf("failed to generate data key: %w", err)
	}

	masterKeys, err := wrapDataKey(dataKey, ageRecipients)
	if err != nil {
		return nil, err
	}

	// Build tree with metadata
	tree := sops.Tree{
		Branches: branches,
		Metadata: sops.Metadata{
			KeyGroups: []sops.KeyGroup{masterKeys},
			Version:   sopsVersion,
		},
	}

	if err := encryptTree(&tree, dataKey, aes.NewCipher()); err != nil {
		return nil, err
	}

	// Emit encrypted file
	encData, err := store.EmitEncryptedFile(tree)
	if err != nil {
		return nil, fmt.Errorf("failed to emit encrypted file: %w", err)
	}

	return encData, nil
}

// Update reconciles plaintext and recipients with an existing SOPS file. It
// reuses the existing data key and the IVs stashed while decrypting unchanged
// values, so only changed values receive new ciphertext.
func Update(opts UpdateOptions) (UpdateResult, error) {
	store, err := storeForFormat(opts.Format)
	if err != nil {
		return UpdateResult{}, err
	}
	if len(opts.Recipients) == 0 {
		return UpdateResult{}, fmt.Errorf("at least one age recipient is required")
	}
	if opts.AgeIdentity == "" {
		return UpdateResult{}, fmt.Errorf("age identity is required")
	}

	state, err := loadAndDecryptTree(store, opts.ExistingCiphertext, opts.AgeIdentity)
	if err != nil {
		return UpdateResult{}, err
	}
	oldPlaintext, err := store.EmitPlainFile(state.tree.Branches)
	if err != nil {
		return UpdateResult{}, fmt.Errorf("failed to emit existing plain file: %w", err)
	}
	newBranches, err := store.LoadPlainFile(opts.Plaintext)
	if err != nil {
		return UpdateResult{}, fmt.Errorf("failed to load plain data: %w", err)
	}

	contentChanged := !bytes.Equal(oldPlaintext, opts.Plaintext) && !reflect.DeepEqual(state.tree.Branches, newBranches)
	recipientsChanged := !hasCanonicalAgeRecipients(*state.tree, opts.Recipients)
	if !contentChanged && !recipientsChanged {
		return UpdateResult{Ciphertext: opts.ExistingCiphertext, Unchanged: true}, nil
	}

	if !contentChanged {
		// Reload the encrypted branches because decrypting state.tree mutated them.
		tree, err := store.LoadEncryptedFile(opts.ExistingCiphertext)
		if err != nil {
			return UpdateResult{}, fmt.Errorf("failed to reload encrypted file: %w", err)
		}
		group, err := wrapDataKey(state.dataKey, opts.Recipients)
		if err != nil {
			return UpdateResult{}, err
		}
		tree.Metadata.KeyGroups = []sops.KeyGroup{group}
		out, err := store.EmitEncryptedFile(tree)
		if err != nil {
			return UpdateResult{}, fmt.Errorf("failed to emit encrypted file: %w", err)
		}
		return UpdateResult{Ciphertext: out, RecipientsChanged: true}, nil
	}

	state.tree.Branches = newBranches
	if recipientsChanged {
		group, err := wrapDataKey(state.dataKey, opts.Recipients)
		if err != nil {
			return UpdateResult{}, err
		}
		state.tree.Metadata.KeyGroups = []sops.KeyGroup{group}
	}
	if err := encryptTree(state.tree, state.dataKey, state.cipher); err != nil {
		return UpdateResult{}, err
	}
	out, err := store.EmitEncryptedFile(*state.tree)
	if err != nil {
		return UpdateResult{}, fmt.Errorf("failed to emit encrypted file: %w", err)
	}
	return UpdateResult{Ciphertext: out, ContentChanged: true, RecipientsChanged: recipientsChanged}, nil
}

// Decrypt decrypts encrypted data with an age identity
// (AGE-SECRET-KEY-...) and verifies the MAC.
func Decrypt(encData []byte, format, ageIdentity string) ([]byte, error) {
	store, err := storeForFormat(format)
	if err != nil {
		return nil, err
	}

	state, err := loadAndDecryptTree(store, encData, ageIdentity)
	if err != nil {
		return nil, err
	}

	// Emit plain file
	plainData, err := store.EmitPlainFile(state.tree.Branches)
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
	state, err := loadAndDecryptTree(store, encData, ageIdentity)
	if err != nil {
		return nil, err
	}

	// Wrap a fresh data key for every new recipient
	dataKey := make([]byte, 32)
	if _, err := rand.Read(dataKey); err != nil {
		return nil, fmt.Errorf("failed to generate data key: %w", err)
	}
	group, err := wrapDataKey(dataKey, newRecipients)
	if err != nil {
		return nil, err
	}

	// Re-encrypt the decrypted values with the fresh data key
	state.tree.Metadata.KeyGroups = []sops.KeyGroup{group}
	if err := encryptTree(state.tree, dataKey, aes.NewCipher()); err != nil {
		return nil, err
	}

	out, err := store.EmitEncryptedFile(*state.tree)
	if err != nil {
		return nil, fmt.Errorf("failed to emit encrypted file: %w", err)
	}
	return out, nil
}

// loadAndDecryptTree loads an encrypted file and decrypts its values in place
// using the age identity, verifying the MAC before returning.
type decryptedTree struct {
	tree    *sops.Tree
	dataKey []byte
	cipher  aes.Cipher
}

func loadAndDecryptTree(store sops.Store, encData []byte, ageIdentity string) (decryptedTree, error) {
	tree, err := store.LoadEncryptedFile(encData)
	if err != nil {
		return decryptedTree{}, fmt.Errorf("failed to load encrypted file: %w", err)
	}
	if tree.Metadata.MessageAuthenticationCode == "" {
		return decryptedTree{}, fmt.Errorf("encrypted file is missing its MAC")
	}

	// Parse age identity and decrypt data key
	var identities sopsage.ParsedIdentities
	if err := identities.Import(ageIdentity); err != nil {
		return decryptedTree{}, fmt.Errorf("failed to parse age identity: %w", err)
	}

	dataKey, err := decryptTreeDataKey(&tree, identities)
	if err != nil {
		return decryptedTree{}, err
	}

	// Decrypt tree values
	cipher := aes.NewCipher()
	mac, err := tree.Decrypt(dataKey, cipher)
	if err != nil {
		return decryptedTree{}, fmt.Errorf("failed to decrypt tree: %w", err)
	}

	// Verify MAC
	storedMACString := tree.Metadata.MessageAuthenticationCode
	if strings.HasPrefix(storedMACString, "ENC[") {
		storedMAC, err := cipher.Decrypt(storedMACString, dataKey, tree.Metadata.LastModified.Format(time.RFC3339))
		if err != nil {
			return decryptedTree{}, fmt.Errorf("failed to decrypt MAC: %w", err)
		}
		var ok bool
		storedMACString, ok = storedMAC.(string)
		if !ok {
			return decryptedTree{}, fmt.Errorf("decrypted MAC has unexpected type %T", storedMAC)
		}
	}
	if storedMACString != mac {
		return decryptedTree{}, fmt.Errorf("MAC mismatch: file may have been tampered with")
	}

	return decryptedTree{tree: &tree, dataKey: dataKey, cipher: cipher}, nil
}

func encryptTree(tree *sops.Tree, dataKey []byte, cipher aes.Cipher) error {
	mac, err := tree.Encrypt(dataKey, cipher)
	if err != nil {
		return fmt.Errorf("failed to encrypt tree: %w", err)
	}
	tree.Metadata.LastModified = time.Now().UTC()
	tree.Metadata.MessageAuthenticationCode, err = cipher.Encrypt(mac, dataKey, tree.Metadata.LastModified.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("failed to encrypt MAC: %w", err)
	}
	return nil
}

func wrapDataKey(dataKey []byte, recipients []string) (sops.KeyGroup, error) {
	group := make(sops.KeyGroup, 0, len(recipients))
	for _, recipient := range recipients {
		masterKey, err := sopsage.MasterKeyFromRecipient(recipient)
		if err != nil {
			return nil, fmt.Errorf("failed to create age master key for recipient %q: %w", recipient, err)
		}
		if err := masterKey.Encrypt(dataKey); err != nil {
			return nil, fmt.Errorf("failed to encrypt data key with age: %w", err)
		}
		group = append(group, masterKey)
	}
	return group, nil
}

func sameRecipients(left, right []string) bool {
	left = append([]string(nil), left...)
	right = append([]string(nil), right...)
	sort.Strings(left)
	sort.Strings(right)
	return reflect.DeepEqual(left, right)
}

func hasCanonicalAgeRecipients(tree sops.Tree, configured []string) bool {
	if len(tree.Metadata.KeyGroups) != 1 || len(tree.Metadata.KeyGroups[0]) != len(configured) {
		return false
	}

	recipients := make([]string, 0, len(configured))
	for _, key := range tree.Metadata.KeyGroups[0] {
		ageMK, ok := key.(*sopsage.MasterKey)
		if !ok || ageMK == nil || ageMK.Recipient == "" {
			return false
		}
		recipients = append(recipients, ageMK.Recipient)
	}
	return sameRecipients(recipients, configured)
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
