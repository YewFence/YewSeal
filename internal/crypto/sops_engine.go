package crypto

import (
	"crypto/rand"
	"fmt"
	"time"

	sops "github.com/getsops/sops/v3"
	"github.com/getsops/sops/v3/aes"
	sopsage "github.com/getsops/sops/v3/age"
	"github.com/getsops/sops/v3/stores/dotenv"
	"github.com/getsops/sops/v3/stores/ini"
	"github.com/getsops/sops/v3/stores/json"
	"github.com/getsops/sops/v3/stores/yaml"
)

const sopsVersion = "3.12.1"

// storeForFormat returns the appropriate sops store for the given format string
func storeForFormat(format string) sops.Store {
	switch format {
	case "yaml":
		return &yaml.Store{}
	case "json":
		return &json.Store{}
	case "dotenv":
		return &dotenv.Store{}
	case "ini":
		return &ini.Store{}
	default:
		return &yaml.Store{}
	}
}

// sopsEncryptData encrypts plain data using the sops library.
// format: "yaml", "json", "dotenv", "ini"
// agePublicKey: age recipient public key string (age1...)
func sopsEncryptData(plainData []byte, format string, agePublicKey string) ([]byte, error) {
	store := storeForFormat(format)

	// Load plain data into tree branches
	branches, err := store.LoadPlainFile(plainData)
	if err != nil {
		return nil, fmt.Errorf("failed to load plain data: %w", err)
	}

	// Create age master key from recipient
	masterKey, err := sopsage.MasterKeyFromRecipient(agePublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create age master key: %w", err)
	}

	// Generate random 32-byte data key
	dataKey := make([]byte, 32)
	if _, err := rand.Read(dataKey); err != nil {
		return nil, fmt.Errorf("failed to generate data key: %w", err)
	}

	// Encrypt data key with age master key
	if err := masterKey.Encrypt(dataKey); err != nil {
		return nil, fmt.Errorf("failed to encrypt data key with age: %w", err)
	}

	// Build tree with metadata
	tree := sops.Tree{
		Branches: branches,
		Metadata: sops.Metadata{
			KeyGroups: []sops.KeyGroup{
				{masterKey},
			},
			Version: sopsVersion,
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

// sopsDecryptData decrypts encrypted data using the sops library.
// format: "yaml", "json", "dotenv", "ini"
// agePrivateKey: age identity private key string (AGE-SECRET-KEY-...)
func sopsDecryptData(encData []byte, format string, agePrivateKey string) ([]byte, error) {
	store := storeForFormat(format)

	// Load encrypted file into tree
	tree, err := store.LoadEncryptedFile(encData)
	if err != nil {
		return nil, fmt.Errorf("failed to load encrypted file: %w", err)
	}

	// Parse age identity and decrypt data key
	var identities sopsage.ParsedIdentities
	if err := identities.Import(agePrivateKey); err != nil {
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

	// Emit plain file
	plainData, err := store.EmitPlainFile(tree.Branches)
	if err != nil {
		return nil, fmt.Errorf("failed to emit plain file: %w", err)
	}

	return plainData, nil
}

// decryptTreeDataKey iterates age master keys in metadata, injects identities,
// and attempts to decrypt the data key. Thread-safe (no global state).
func decryptTreeDataKey(tree *sops.Tree, identities sopsage.ParsedIdentities) ([]byte, error) {
	for _, group := range tree.Metadata.KeyGroups {
		for _, key := range group {
			ageMK, ok := key.(*sopsage.MasterKey)
			if !ok {
				continue
			}
			identities.ApplyToMasterKey(ageMK)
			dataKey, err := ageMK.Decrypt()
			if err == nil {
				return dataKey, nil
			}
		}
	}
	return nil, fmt.Errorf("failed to decrypt data key: no matching age key found")
}

// extractAgeRecipientFromTree extracts the age public key from encrypted file metadata.
// Used by Edit to re-encrypt with the same key after editing.
func extractAgeRecipientFromTree(tree sops.Tree) (string, error) {
	for _, group := range tree.Metadata.KeyGroups {
		for _, key := range group {
			ageMK, ok := key.(*sopsage.MasterKey)
			if !ok {
				continue
			}
			if ageMK.Recipient != "" {
				return ageMK.Recipient, nil
			}
		}
	}
	return "", fmt.Errorf("no age recipient found in encrypted file metadata")
}
