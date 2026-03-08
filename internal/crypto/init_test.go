package crypto

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// setupAgeKey Tests
// ============================================================================

func TestSetupAgeKey_NewKey(t *testing.T) {
	tempDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	// Ensure no key exists
	keyFile := ".age/keys.txt"
	os.RemoveAll(".age")

	publicKey, err := setupAgeKey(false)
	require.NoError(t, err)
	assert.NotEmpty(t, publicKey)
	assert.True(t, len(publicKey) > 50, "Public key should be a valid age public key")

	// Verify key file was created
	_, err = os.Stat(keyFile)
	assert.NoError(t, err)
}

func TestSetupAgeKey_ExistingKey(t *testing.T) {
	tempDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	// First, generate a key
	publicKey1, err := setupAgeKey(false)
	require.NoError(t, err)

	// Call again without force - should return same key
	publicKey2, err := setupAgeKey(false)
	require.NoError(t, err)

	assert.Equal(t, publicKey1, publicKey2)
}

func TestSetupAgeKey_ForceRegenerate(t *testing.T) {
	tempDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	// First, generate a key
	publicKey1, err := setupAgeKey(false)
	require.NoError(t, err)

	// Call with force - should generate new key
	publicKey2, err := setupAgeKey(true)
	require.NoError(t, err)

	assert.NotEqual(t, publicKey1, publicKey2)
}

func TestSetupAgeKey_InvalidExistingKey(t *testing.T) {
	tempDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	// Create .age directory with invalid key file
	os.MkdirAll(".age", 0700)
	err := os.WriteFile(".age/keys.txt", []byte("invalid key content"), 0600)
	require.NoError(t, err)

	// Should fail to extract public key from invalid content
	_, err = setupAgeKey(false)
	assert.Error(t, err)
}

// ============================================================================
// updateGitignore Tests
// ============================================================================

func TestUpdateGitignore_Create(t *testing.T) {
	tempDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	// Ensure no .gitignore exists
	os.Remove(".gitignore")

	err := updateGitignore("wrangler.toml")
	require.NoError(t, err)

	// Verify .gitignore was created
	content, err := os.ReadFile(".gitignore")
	require.NoError(t, err)

	assert.Contains(t, string(content), "# YewSeal")
	assert.Contains(t, string(content), "wrangler.toml")
	assert.Contains(t, string(content), ".age/keys.txt")
}

func TestUpdateGitignore_Update(t *testing.T) {
	tempDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	// Create existing .gitignore
	existingContent := `# Existing content
node_modules/
.env
`
	err := os.WriteFile(".gitignore", []byte(existingContent), 0644)
	require.NoError(t, err)

	err = updateGitignore("config.toml")
	require.NoError(t, err)

	// Verify content was appended
	content, err := os.ReadFile(".gitignore")
	require.NoError(t, err)

	assert.Contains(t, string(content), "node_modules/")
	assert.Contains(t, string(content), ".env")
	assert.Contains(t, string(content), "# YewSeal")
	assert.Contains(t, string(content), "config.toml")
	assert.Contains(t, string(content), ".age/keys.txt")
}

func TestUpdateGitignore_AlreadyContainsYewSeal(t *testing.T) {
	tempDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	// Create .gitignore with YewSeal section already
	existingContent := `# Existing content
node_modules/

# YewSeal - Decrypted configuration files
wrangler.toml

# YewSeal - Age private keys
.age/keys.txt
`
	err := os.WriteFile(".gitignore", []byte(existingContent), 0644)
	require.NoError(t, err)

	err = updateGitignore("different.toml")
	require.NoError(t, err)

	// Verify content was NOT modified (YewSeal section already exists)
	content, err := os.ReadFile(".gitignore")
	require.NoError(t, err)

	// Should still have wrangler.toml, not different.toml
	assert.Contains(t, string(content), "wrangler.toml")
	assert.NotContains(t, string(content), "different.toml")
}

// ============================================================================
// createExampleFile Tests
// ============================================================================

func TestCreateExampleFile_Success(t *testing.T) {
	tempDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	// Create input file
	inputContent := `[config]
key = "value"
secret = "should-be-removed"
`
	inputFile := "config.toml"
	err := os.WriteFile(inputFile, []byte(inputContent), 0644)
	require.NoError(t, err)

	// Create example file
	createExampleFile(inputFile)

	// Verify example file was created
	exampleFile := "config.example.toml"
	content, err := os.ReadFile(exampleFile)
	require.NoError(t, err)

	assert.Equal(t, inputContent, string(content))
}

func TestCreateExampleFile_InputNotExist(t *testing.T) {
	tempDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	// Call with non-existent file (should not panic, just print warning)
	createExampleFile("nonexistent.toml")

	// Verify no example file was created
	_, err := os.Stat("nonexistent.example.toml")
	assert.True(t, os.IsNotExist(err))
}

func TestCreateExampleFile_WithPath(t *testing.T) {
	tempDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	// Create subdirectory and input file
	os.MkdirAll("subdir", 0755)
	inputFile := filepath.Join("subdir", "config.toml")
	err := os.WriteFile(inputFile, []byte("[test]\nkey = 1"), 0644)
	require.NoError(t, err)

	createExampleFile(inputFile)

	// Verify example file was created in same directory
	exampleFile := filepath.Join("subdir", "config.example.toml")
	_, err = os.Stat(exampleFile)
	assert.NoError(t, err)
}

func TestCreateExampleFile_PreservesExtension(t *testing.T) {
	tempDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	testCases := []struct {
		inputFile    string
		expectedFile string
	}{
		{"config.toml", "config.example.toml"},
		{"settings.yaml", "settings.example.yaml"},
		{"app.json", "app.example.json"},
	}

	for _, tc := range testCases {
		// Create input file
		err := os.WriteFile(tc.inputFile, []byte("content"), 0644)
		require.NoError(t, err)

		createExampleFile(tc.inputFile)

		// Verify example file name
		_, err = os.Stat(tc.expectedFile)
		assert.NoError(t, err, "Expected %s to exist", tc.expectedFile)

		// Cleanup
		os.Remove(tc.inputFile)
		os.Remove(tc.expectedFile)
	}
}
