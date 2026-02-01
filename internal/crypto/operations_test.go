package crypto

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testEnvironment holds the test environment configuration
type testEnvironment struct {
	tempDir    string
	publicKey  string
	privateKey string
	keyFile    string
}

// skipIfToolsMissing checks if required external tools are available
func skipIfToolsMissing(t *testing.T) {
	tools := []string{"age", "age-keygen", "sops", "toml2yaml", "yaml2toml"}
	for _, tool := range tools {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("Skipping test: %s is not installed", tool)
		}
	}
}

// setupTestEnvironment creates an isolated test environment with Age keys
func setupTestEnvironment(t *testing.T) *testEnvironment {
	t.Helper()
	skipIfToolsMissing(t)

	tempDir := t.TempDir()

	// Change to temp directory for test isolation
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(tempDir)
	require.NoError(t, err)
	t.Cleanup(func() {
		os.Chdir(oldWd)
	})

	// Create .age directory
	ageDir := filepath.Join(tempDir, ".age")
	err = os.MkdirAll(ageDir, 0700)
	require.NoError(t, err)

	// Generate Age key
	keyFile := filepath.Join(ageDir, "keys.txt")
	cmd := exec.Command("age-keygen", "-o", keyFile)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Failed to generate age key: %s", string(output))

	// Read key file to extract public key
	keyContent, err := os.ReadFile(keyFile)
	require.NoError(t, err)

	publicKey := ExtractPublicKey(string(output) + string(keyContent))
	require.NotEmpty(t, publicKey, "Failed to extract public key")

	// Extract private key
	privateKey := ""
	for _, line := range strings.Split(string(keyContent), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "AGE-SECRET-KEY-") {
			privateKey = line
			break
		}
	}
	require.NotEmpty(t, privateKey, "Failed to extract private key")

	return &testEnvironment{
		tempDir:    tempDir,
		publicKey:  publicKey,
		privateKey: privateKey,
		keyFile:    keyFile,
	}
}

// sampleTOML returns a sample TOML content for testing
func sampleTOML() []byte {
	return []byte(`[database]
host = "localhost"
port = 5432
password = "secret123"

[server]
name = "test"
enabled = true
`)
}

// complexTOML returns a complex nested TOML content for testing
func complexTOML() []byte {
	return []byte(`name = "test-app"
version = "1.0.0"

[database]
host = "localhost"
port = 5432
password = "secret123"

[database.pool]
min = 5
max = 20

[[routes]]
pattern = "api.example.com"
custom_domain = true

[[routes]]
pattern = "web.example.com"
custom_domain = false

[observability.logs]
enabled = true
level = "info"

[vars]
API_KEY = "sk-test-12345"
SECRET_TOKEN = "abc123xyz"
`)
}

// ============================================================================
// Encrypt Tests
// ============================================================================

func TestEncrypt_Success(t *testing.T) {
	env := setupTestEnvironment(t)

	// Create input TOML file
	inputFile := "config.toml"
	outputFile := "config.enc.toml.yaml"
	err := os.WriteFile(inputFile, sampleTOML(), 0644)
	require.NoError(t, err)

	// Run encrypt
	err = Encrypt(inputFile, outputFile, env.keyFile, env.publicKey, false)
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
	env := setupTestEnvironment(t)

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
	err = Encrypt(inputFile, outputFile, env.keyFile, "", false)
	require.NoError(t, err)

	// Verify output file exists and is encrypted
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "sops:")
}

func TestEncrypt_WithoutSopsYaml(t *testing.T) {
	env := setupTestEnvironment(t)

	// Ensure no .sops.yaml exists
	os.Remove(".sops.yaml")

	// Create input TOML file
	inputFile := "config.toml"
	outputFile := "config.enc.toml.yaml"
	err := os.WriteFile(inputFile, sampleTOML(), 0644)
	require.NoError(t, err)

	// Run encrypt with explicit public key
	err = Encrypt(inputFile, outputFile, env.keyFile, env.publicKey, false)
	require.NoError(t, err)

	// Verify output file exists and is encrypted
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "sops:")
}

func TestEncrypt_FileNotExist(t *testing.T) {
	env := setupTestEnvironment(t)

	err := Encrypt("nonexistent.toml", "output.yaml", env.keyFile, env.publicKey, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestEncrypt_InvalidTOML(t *testing.T) {
	env := setupTestEnvironment(t)

	// Create invalid TOML file
	invalidTOML := []byte(`[section
key = "missing bracket"`)
	err := os.WriteFile("invalid.toml", invalidTOML, 0644)
	require.NoError(t, err)

	err = Encrypt("invalid.toml", "output.yaml", env.keyFile, env.publicKey, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to convert TOML to YAML")
}

// ============================================================================
// Decrypt Tests
// ============================================================================

func TestDecrypt_Success(t *testing.T) {
	env := setupTestEnvironment(t)

	// First encrypt a file
	inputFile := "config.toml"
	encryptedFile := "config.enc.toml.yaml"
	decryptedFile := "config.decrypted.toml"
	originalContent := sampleTOML()

	err := os.WriteFile(inputFile, originalContent, 0644)
	require.NoError(t, err)

	err = Encrypt(inputFile, encryptedFile, env.keyFile, env.publicKey, false)
	require.NoError(t, err)

	// Now decrypt
	err = Decrypt(encryptedFile, decryptedFile, env.keyFile, false)
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
	env := setupTestEnvironment(t)

	err := Decrypt("nonexistent.yaml", "output.toml", env.keyFile, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestDecrypt_InvalidEncryptedFile(t *testing.T) {
	env := setupTestEnvironment(t)

	// Create a non-encrypted YAML file
	plainYAML := []byte(`database:
  host: localhost
  port: 5432
`)
	err := os.WriteFile("plain.yaml", plainYAML, 0644)
	require.NoError(t, err)

	err = Decrypt("plain.yaml", "output.toml", env.keyFile, false)
	assert.Error(t, err)
}

func TestDecrypt_WrongKey(t *testing.T) {
	env := setupTestEnvironment(t)

	// First encrypt a file
	inputFile := "config.toml"
	encryptedFile := "config.enc.toml.yaml"

	err := os.WriteFile(inputFile, sampleTOML(), 0644)
	require.NoError(t, err)

	err = Encrypt(inputFile, encryptedFile, env.keyFile, env.publicKey, false)
	require.NoError(t, err)

	// Generate a different key
	wrongKeyDir := filepath.Join(env.tempDir, ".wrong-age")
	err = os.MkdirAll(wrongKeyDir, 0700)
	require.NoError(t, err)

	wrongKeyFile := filepath.Join(wrongKeyDir, "keys.txt")
	cmd := exec.Command("age-keygen", "-o", wrongKeyFile)
	err = cmd.Run()
	require.NoError(t, err)

	// Try to decrypt with wrong key
	err = Decrypt(encryptedFile, "output.toml", wrongKeyFile, false)
	assert.Error(t, err)
}

// ============================================================================
// Round-trip Tests
// ============================================================================

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	env := setupTestEnvironment(t)

	inputFile := "original.toml"
	encryptedFile := "encrypted.yaml"
	decryptedFile := "decrypted.toml"
	originalContent := sampleTOML()

	err := os.WriteFile(inputFile, originalContent, 0644)
	require.NoError(t, err)

	// Encrypt
	err = Encrypt(inputFile, encryptedFile, env.keyFile, env.publicKey, false)
	require.NoError(t, err)

	// Decrypt
	err = Decrypt(encryptedFile, decryptedFile, env.keyFile, false)
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
	env := setupTestEnvironment(t)

	inputFile := "complex.toml"
	encryptedFile := "complex.enc.yaml"
	decryptedFile := "complex.decrypted.toml"

	err := os.WriteFile(inputFile, complexTOML(), 0644)
	require.NoError(t, err)

	// Encrypt
	err = Encrypt(inputFile, encryptedFile, env.keyFile, env.publicKey, false)
	require.NoError(t, err)

	// Decrypt
	err = Decrypt(encryptedFile, decryptedFile, env.keyFile, false)
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
	env := setupTestEnvironment(t)

	inputFile := "config.toml"
	outputFile := "config.enc.yaml"

	err := os.WriteFile(inputFile, sampleTOML(), 0644)
	require.NoError(t, err)

	// Run encrypt with verbose mode (should not error)
	err = Encrypt(inputFile, outputFile, env.keyFile, env.publicKey, true)
	require.NoError(t, err)

	// Verify output
	_, err = os.Stat(outputFile)
	require.NoError(t, err)
}

func TestDecrypt_VerboseMode(t *testing.T) {
	env := setupTestEnvironment(t)

	inputFile := "config.toml"
	encryptedFile := "config.enc.yaml"
	decryptedFile := "config.dec.toml"

	err := os.WriteFile(inputFile, sampleTOML(), 0644)
	require.NoError(t, err)

	err = Encrypt(inputFile, encryptedFile, env.keyFile, env.publicKey, false)
	require.NoError(t, err)

	// Run decrypt with verbose mode (should not error)
	err = Decrypt(encryptedFile, decryptedFile, env.keyFile, true)
	require.NoError(t, err)
}
