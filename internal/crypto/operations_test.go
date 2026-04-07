package crypto

import (
	"os"
	"testing"

	"filippo.io/age"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Encrypt Tests
// ============================================================================

func TestEncrypt_Success(t *testing.T) {
	skipIfNoRemarshal(t)
	env := setupIntegrationEnv(t)

	// Create input TOML file
	inputFile := "config.toml"
	outputFile := "config.enc.toml.yaml"
	err := os.WriteFile(inputFile, sampleTOML(), 0644)
	require.NoError(t, err)

	// Run encrypt
	err = Encrypt(inputFile, outputFile, env.keyFile, env.publicKey, "", false)
	require.NoError(t, err)

	// Verify output file exists
	_, err = os.Stat(outputFile)
	require.NoError(t, err)

	// Verify output is SOPS encrypted (contains sops metadata)
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "sops:")
	assert.Contains(t, string(content), "age:")
}

func TestEncrypt_WithSopsYaml(t *testing.T) {
	skipIfNoRemarshal(t)
	env := setupIntegrationEnv(t)

	// Create .sops.yaml configuration
	sopsConfig := `creation_rules:
  - path_regex: ^config\.enc\.toml\.yaml$
    age: ` + env.publicKey + `
`
	err := os.WriteFile(".sops.yaml", []byte(sopsConfig), 0644)
	require.NoError(t, err)

	// Create input TOML file
	inputFile := "config.toml"
	outputFile := "config.enc.toml.yaml"
	err = os.WriteFile(inputFile, sampleTOML(), 0644)
	require.NoError(t, err)

	// Run encrypt (should use .sops.yaml)
	err = Encrypt(inputFile, outputFile, env.keyFile, "", "", false)
	require.NoError(t, err)

	// Verify output file exists and is encrypted
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "sops:")
}

func TestEncrypt_WithoutSopsYaml(t *testing.T) {
	skipIfNoRemarshal(t)
	env := setupIntegrationEnv(t)

	// Ensure no .sops.yaml exists
	os.Remove(".sops.yaml")

	// Create input TOML file
	inputFile := "config.toml"
	outputFile := "config.enc.toml.yaml"
	err := os.WriteFile(inputFile, sampleTOML(), 0644)
	require.NoError(t, err)

	// Run encrypt with explicit public key
	err = Encrypt(inputFile, outputFile, env.keyFile, env.publicKey, "", false)
	require.NoError(t, err)

	// Verify output file exists and is encrypted
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "sops:")
}

func TestEncrypt_FileNotExist(t *testing.T) {
	env := setupIntegrationEnv(t)

	err := Encrypt("nonexistent.toml", "output.yaml", env.keyFile, env.publicKey, "", false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestEncrypt_InvalidTOML(t *testing.T) {
	skipIfNoRemarshal(t)
	env := setupIntegrationEnv(t)

	// Create invalid TOML file
	invalidTOML := []byte(`[section
key = "missing bracket"`)
	err := os.WriteFile("invalid.toml", invalidTOML, 0644)
	require.NoError(t, err)

	err = Encrypt("invalid.toml", "output.yaml", env.keyFile, env.publicKey, "", false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to convert TOML to YAML")
}

// ============================================================================
// Decrypt Tests
// ============================================================================

func TestDecrypt_Success(t *testing.T) {
	skipIfNoRemarshal(t)
	env := setupIntegrationEnv(t)

	// First encrypt a file
	inputFile := "config.toml"
	encryptedFile := "config.enc.toml.yaml"
	decryptedFile := "config.decrypted.toml"
	originalContent := sampleTOML()

	err := os.WriteFile(inputFile, originalContent, 0644)
	require.NoError(t, err)

	err = Encrypt(inputFile, encryptedFile, env.keyFile, env.publicKey, "", false)
	require.NoError(t, err)

	// Now decrypt
	err = Decrypt(encryptedFile, decryptedFile, env.keyFile, "", false)
	require.NoError(t, err)

	// Verify decrypted content contains the original data
	decryptedContent, err := os.ReadFile(decryptedFile)
	require.NoError(t, err)

	assert.Contains(t, string(decryptedContent), "database")
	assert.Contains(t, string(decryptedContent), "localhost")
	assert.Contains(t, string(decryptedContent), "secret123")
	assert.Contains(t, string(decryptedContent), "server")
}

func TestDecrypt_FileNotExist(t *testing.T) {
	env := setupIntegrationEnv(t)

	err := Decrypt("nonexistent.yaml", "output.toml", env.keyFile, "", false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestDecrypt_InvalidEncryptedFile(t *testing.T) {
	env := setupIntegrationEnv(t)

	// Create a non-encrypted YAML file
	plainYAML := []byte(`database:
  host: localhost
  port: 5432
`)
	err := os.WriteFile("plain.yaml", plainYAML, 0644)
	require.NoError(t, err)

	// Use YAML output to avoid remarshal dependency
	err = Decrypt("plain.yaml", "output.yaml", env.keyFile, "", false)
	assert.Error(t, err)
}

func TestDecrypt_WrongKey(t *testing.T) {
	skipIfNoRemarshal(t)
	env := setupIntegrationEnv(t)

	// First encrypt a file
	inputFile := "config.toml"
	encryptedFile := "config.enc.toml.yaml"

	err := os.WriteFile(inputFile, sampleTOML(), 0644)
	require.NoError(t, err)

	err = Encrypt(inputFile, encryptedFile, env.keyFile, env.publicKey, "", false)
	require.NoError(t, err)

	// Generate a different key using embedded age library
	identity2, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	wrongKeyDir := ".wrong-age"
	err = os.MkdirAll(wrongKeyDir, 0700)
	require.NoError(t, err)

	wrongKeyFile := wrongKeyDir + "/keys.txt"
	wrongKeyContent := "# public key: " + identity2.Recipient().String() + "\n" + identity2.String() + "\n"
	err = os.WriteFile(wrongKeyFile, []byte(wrongKeyContent), 0600)
	require.NoError(t, err)

	// Try to decrypt with wrong key
	err = Decrypt(encryptedFile, "output.toml", wrongKeyFile, "", false)
	assert.Error(t, err)
}

// ============================================================================
// Round-trip Tests
// ============================================================================

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	skipIfNoRemarshal(t)
	env := setupIntegrationEnv(t)

	inputFile := "original.toml"
	encryptedFile := "encrypted.yaml"
	decryptedFile := "decrypted.toml"
	originalContent := sampleTOML()

	err := os.WriteFile(inputFile, originalContent, 0644)
	require.NoError(t, err)

	// Encrypt
	err = Encrypt(inputFile, encryptedFile, env.keyFile, env.publicKey, "", false)
	require.NoError(t, err)

	// Decrypt
	err = Decrypt(encryptedFile, decryptedFile, env.keyFile, "", false)
	require.NoError(t, err)

	// Verify content integrity
	decryptedContent, err := os.ReadFile(decryptedFile)
	require.NoError(t, err)

	// Check all key values are preserved
	assert.Contains(t, string(decryptedContent), "host")
	assert.Contains(t, string(decryptedContent), "localhost")
	assert.Contains(t, string(decryptedContent), "port")
	assert.Contains(t, string(decryptedContent), "5432")
	assert.Contains(t, string(decryptedContent), "password")
	assert.Contains(t, string(decryptedContent), "secret123")
	assert.Contains(t, string(decryptedContent), "server")
	assert.Contains(t, string(decryptedContent), "name")
	assert.Contains(t, string(decryptedContent), "test")
	assert.Contains(t, string(decryptedContent), "enabled")
	assert.Contains(t, string(decryptedContent), "true")
}

func TestEncryptDecrypt_ComplexTOML(t *testing.T) {
	skipIfNoRemarshal(t)
	env := setupIntegrationEnv(t)

	inputFile := "complex.toml"
	encryptedFile := "complex.enc.yaml"
	decryptedFile := "complex.decrypted.toml"

	err := os.WriteFile(inputFile, complexTOML(), 0644)
	require.NoError(t, err)

	// Encrypt
	err = Encrypt(inputFile, encryptedFile, env.keyFile, env.publicKey, "", false)
	require.NoError(t, err)

	// Decrypt
	err = Decrypt(encryptedFile, decryptedFile, env.keyFile, "", false)
	require.NoError(t, err)

	// Verify complex structure is preserved
	decryptedContent, err := os.ReadFile(decryptedFile)
	require.NoError(t, err)

	assert.Contains(t, string(decryptedContent), "test-app")
	assert.Contains(t, string(decryptedContent), "database")
	assert.Contains(t, string(decryptedContent), "pool")
	assert.Contains(t, string(decryptedContent), "routes")
	assert.Contains(t, string(decryptedContent), "api.example.com")
	assert.Contains(t, string(decryptedContent), "observability")
	assert.Contains(t, string(decryptedContent), "API_KEY")
	assert.Contains(t, string(decryptedContent), "sk-test-12345")
}

// ============================================================================
// Encrypt Verbose Mode Test
// ============================================================================

func TestEncrypt_VerboseMode(t *testing.T) {
	skipIfNoRemarshal(t)
	env := setupIntegrationEnv(t)

	inputFile := "config.toml"
	outputFile := "config.enc.yaml"

	err := os.WriteFile(inputFile, sampleTOML(), 0644)
	require.NoError(t, err)

	// Run encrypt with verbose mode (should not error)
	err = Encrypt(inputFile, outputFile, env.keyFile, env.publicKey, "", true)
	require.NoError(t, err)

	// Verify output
	_, err = os.Stat(outputFile)
	require.NoError(t, err)
}

func TestDecrypt_VerboseMode(t *testing.T) {
	skipIfNoRemarshal(t)
	env := setupIntegrationEnv(t)

	inputFile := "config.toml"
	encryptedFile := "config.enc.yaml"
	decryptedFile := "config.dec.toml"

	err := os.WriteFile(inputFile, sampleTOML(), 0644)
	require.NoError(t, err)

	err = Encrypt(inputFile, encryptedFile, env.keyFile, env.publicKey, "", false)
	require.NoError(t, err)

	// Run decrypt with verbose mode (should not error)
	err = Decrypt(encryptedFile, decryptedFile, env.keyFile, "", true)
	require.NoError(t, err)
}
