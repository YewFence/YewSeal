package crypto

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/utils"
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

	err := utils.UpdateGitignore([]config.FilePair{
		{Dec: "wrangler.toml", Enc: "wrangler.enc.toml.yaml"},
	})
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

	err = utils.UpdateGitignore([]config.FilePair{
		{Dec: "config.toml", Enc: "config.enc.toml.yaml"},
		{Dec: ".dev.vars", Enc: ".dev.vars.enc.yaml"},
	})
	require.NoError(t, err)

	// Verify content was appended
	content, err := os.ReadFile(".gitignore")
	require.NoError(t, err)

	assert.Contains(t, string(content), "node_modules/")
	assert.Contains(t, string(content), ".env")
	assert.Contains(t, string(content), "# YewSeal")
	assert.Contains(t, string(content), "config.toml")
	assert.Contains(t, string(content), ".dev.vars")
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

	err = utils.UpdateGitignore([]config.FilePair{
		{Dec: "wrangler.toml", Enc: "wrangler.enc.toml.yaml"},
		{Dec: "different.toml", Enc: "different.enc.toml.yaml"},
	})
	require.NoError(t, err)

	// Verify the new plaintext file was appended without losing the old one
	content, err := os.ReadFile(".gitignore")
	require.NoError(t, err)

	assert.Contains(t, string(content), "wrangler.toml")
	assert.Contains(t, string(content), "different.toml")
}

func TestCollectInitFilePairs_NonInteractiveDefaultsEncryptedName(t *testing.T) {
	filePairs := collectInitFilePairs("app.toml", "")

	require.Len(t, filePairs, 1)
	assert.Equal(t, "app.toml", filePairs[0].Dec)
	assert.Equal(t, "app.enc.toml.yaml", filePairs[0].Enc)
}

func TestConfirmInitOverwrite_NonInteractiveExistingConfig(t *testing.T) {
	tempDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	err := os.WriteFile(".yewseal.toml", []byte("existing = true"), 0644)
	require.NoError(t, err)

	allowed, err := confirmInitOverwrite(false, false)
	assert.False(t, allowed)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--force")
}

func TestCollectInitFilePairs_InteractiveMultiple(t *testing.T) {
	oldStdin := os.Stdin
	inputFile, err := os.CreateTemp(t.TempDir(), "stdin-*")
	require.NoError(t, err)
	defer inputFile.Close()
	defer func() { os.Stdin = oldStdin }()

	_, err = inputFile.WriteString("app.toml\n\nY\n.dev.vars\n.dev.vars.enc.yaml\nn\n")
	require.NoError(t, err)
	_, err = inputFile.Seek(0, 0)
	require.NoError(t, err)

	os.Stdin = inputFile

	filePairs := collectInitFilePairs("", "")
	require.Len(t, filePairs, 2)
	assert.Equal(t, "app.toml", filePairs[0].Dec)
	assert.Equal(t, "app.enc.toml.yaml", filePairs[0].Enc)
	assert.Equal(t, ".dev.vars", filePairs[1].Dec)
	assert.Equal(t, ".dev.vars.enc.yaml", filePairs[1].Enc)
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
