package crypto

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"filippo.io/age"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/errx"
	"github.com/YewFence/YewSeal/internal/project"
)

// ============================================================================
// Group 1: TOML 加解密核心（需要 remarshal）
// ============================================================================

func TestIntegration_TOML_RoundTrip(t *testing.T) {
	skipIfNoRemarshal(t)
	env := setupIntegrationEnv(t)

	writeTestFile(t, "config.toml", sampleTOML())

	// Encrypt
	err := Encrypt("config.toml", "config.enc.toml.yaml", env.keyFile, env.publicKey, "", false)
	require.NoError(t, err)

	// Verify encrypted file contains SOPS metadata and no plaintext
	encContent, err := os.ReadFile("config.enc.toml.yaml")
	require.NoError(t, err)
	assert.Contains(t, string(encContent), "sops:")
	assert.NotContains(t, string(encContent), "secret123")

	// Decrypt
	err = Decrypt("config.enc.toml.yaml", "config.dec.toml", env.keyFile, "", false)
	require.NoError(t, err)

	// Verify decrypted content preserves key-values
	decContent, err := os.ReadFile("config.dec.toml")
	require.NoError(t, err)
	content := string(decContent)
	assert.Contains(t, content, "localhost")
	assert.Contains(t, content, "secret123")
	assert.Contains(t, content, "5432")
	assert.Contains(t, content, "test")
}

func TestIntegration_TOML_ComplexStructure(t *testing.T) {
	skipIfNoRemarshal(t)
	env := setupIntegrationEnv(t)

	writeTestFile(t, "complex.toml", complexTOML())

	err := Encrypt("complex.toml", "complex.enc.toml.yaml", env.keyFile, env.publicKey, "", false)
	require.NoError(t, err)

	err = Decrypt("complex.enc.toml.yaml", "complex.dec.toml", env.keyFile, "", false)
	require.NoError(t, err)

	decContent, err := os.ReadFile("complex.dec.toml")
	require.NoError(t, err)
	content := string(decContent)

	// Verify nested tables, array tables, and mixed types preserved
	assert.Contains(t, content, "test-app")
	assert.Contains(t, content, "database")
	assert.Contains(t, content, "pool")
	assert.Contains(t, content, "routes")
	assert.Contains(t, content, "api.example.com")
	assert.Contains(t, content, "web.example.com")
	assert.Contains(t, content, "observability")
	assert.Contains(t, content, "sk-test-12345")
	assert.Contains(t, content, "abc123xyz")
}

func TestIntegration_TOML_WranglerConfig(t *testing.T) {
	skipIfNoRemarshal(t)
	env := setupIntegrationEnv(t)

	wranglerTOML := []byte(`name = "my-worker"
main = "src/index.ts"
compatibility_date = "2024-01-01"

[vars]
API_KEY = "sk-live-key-12345"
DATABASE_URL = "postgres://user:pass@host:5432/db"
SECRET_TOKEN = "super-secret-token"

[[routes]]
pattern = "api.example.com/*"
custom_domain = true

[[routes]]
pattern = "web.example.com/*"
custom_domain = false

[triggers]
crons = ["*/5 * * * *"]
`)
	writeTestFile(t, "wrangler.toml", wranglerTOML)

	err := Encrypt("wrangler.toml", "wrangler.enc.toml.yaml", env.keyFile, env.publicKey, "", false)
	require.NoError(t, err)

	err = Decrypt("wrangler.enc.toml.yaml", "wrangler.dec.toml", env.keyFile, "", false)
	require.NoError(t, err)

	decContent, err := os.ReadFile("wrangler.dec.toml")
	require.NoError(t, err)
	content := string(decContent)

	assert.Contains(t, content, "my-worker")
	assert.Contains(t, content, "sk-live-key-12345")
	assert.Contains(t, content, "postgres://")
	assert.Contains(t, content, "routes")
	assert.Contains(t, content, "api.example.com")
}

func TestIntegration_TOML_EncryptedOutputFormat(t *testing.T) {
	skipIfNoRemarshal(t)
	env := setupIntegrationEnv(t)

	writeTestFile(t, "config.toml", sampleTOML())

	err := Encrypt("config.toml", "config.enc.toml.yaml", env.keyFile, env.publicKey, "", false)
	require.NoError(t, err)

	encContent, err := os.ReadFile("config.enc.toml.yaml")
	require.NoError(t, err)
	content := string(encContent)

	// Verify output is YAML with SOPS age metadata
	assert.Contains(t, content, "sops:")
	assert.Contains(t, content, "age:")
	assert.Contains(t, content, "recipient:")
	assert.Contains(t, content, "enc:")

	// Verify plaintext values are encrypted
	assert.NotContains(t, content, "secret123")
	assert.NotContains(t, content, "localhost")
}

func TestIntegration_TOML_SpecialValues(t *testing.T) {
	skipIfNoRemarshal(t)
	env := setupIntegrationEnv(t)

	specialTOML := []byte(`[config]
multiline = "line one\nline two\nline three"
float_val = 3.14
bool_val = true
int_val = 42
negative = -100
`)
	writeTestFile(t, "special.toml", specialTOML)

	err := Encrypt("special.toml", "special.enc.toml.yaml", env.keyFile, env.publicKey, "", false)
	require.NoError(t, err)

	err = Decrypt("special.enc.toml.yaml", "special.dec.toml", env.keyFile, "", false)
	require.NoError(t, err)

	decContent, err := os.ReadFile("special.dec.toml")
	require.NoError(t, err)
	content := string(decContent)

	assert.Contains(t, content, "3.14")
	assert.Contains(t, content, "true")
	assert.Contains(t, content, "42")
}

// ============================================================================
// Group 2: TOML 批量操作（需要 remarshal）
// ============================================================================

func TestIntegration_TOML_BatchEncrypt_DirScan(t *testing.T) {
	skipIfNoRemarshal(t)
	env := setupIntegrationEnv(t)

	// Create 3 TOML files
	for i := 1; i <= 3; i++ {
		content := fmt.Sprintf("[config%d]\nkey = \"value%d\"\nsecret = \"secret%d\"\n", i, i, i)
		writeTestFile(t, fmt.Sprintf("batch%d.toml", i), []byte(content))
	}

	summary, err := BatchEncrypt(BatchOptions{
		InputDir:     ".",
		Pattern:      "batch*.toml",
		OutputSuffix: ".enc.toml.yaml",
		KeyFile:      env.keyFile,
		PublicKey:    env.publicKey,
	})
	require.NoError(t, err)

	assert.Equal(t, 3, summary.TotalFiles)
	assert.Equal(t, 3, summary.SuccessCount)
	assert.Equal(t, 0, summary.FailedCount)

	// Verify output files exist
	for i := 1; i <= 3; i++ {
		_, err := os.Stat(fmt.Sprintf("batch%d.enc.toml.yaml", i))
		assert.NoError(t, err, "batch%d.enc.toml.yaml should exist", i)
	}
}

func TestIntegration_TOML_BatchDecrypt_DirScan(t *testing.T) {
	skipIfNoRemarshal(t)
	env := setupIntegrationEnv(t)

	// Create and encrypt 3 TOML files
	for i := 1; i <= 3; i++ {
		content := fmt.Sprintf("[config%d]\nkey = \"value%d\"\nsecret = \"secret%d\"\n", i, i, i)
		writeTestFile(t, fmt.Sprintf("batch%d.toml", i), []byte(content))
	}

	_, err := BatchEncrypt(BatchOptions{
		InputDir:     ".",
		Pattern:      "batch*.toml",
		OutputSuffix: ".enc.toml.yaml",
		KeyFile:      env.keyFile,
		PublicKey:    env.publicKey,
	})
	require.NoError(t, err)

	// Remove originals so we can verify decryption creates them
	for i := 1; i <= 3; i++ {
		os.Remove(fmt.Sprintf("batch%d.toml", i))
	}

	// Batch decrypt
	summary, err := BatchDecrypt(BatchOptions{
		InputDir:     ".",
		Pattern:      "*.enc.toml.yaml",
		OutputSuffix: ".toml",
		KeyFile:      env.keyFile,
	})
	require.NoError(t, err)

	assert.Equal(t, 3, summary.TotalFiles)
	assert.Equal(t, 3, summary.SuccessCount)

	// Verify decrypted content
	for i := 1; i <= 3; i++ {
		decContent, err := os.ReadFile(fmt.Sprintf("batch%d.toml", i))
		require.NoError(t, err)
		assert.Contains(t, string(decContent), fmt.Sprintf("value%d", i))
		assert.Contains(t, string(decContent), fmt.Sprintf("secret%d", i))
	}
}

func TestIntegration_TOML_BatchEncrypt_FilePairs(t *testing.T) {
	skipIfNoRemarshal(t)
	env := setupIntegrationEnv(t)

	writeTestFile(t, "app.toml", sampleTOML())
	writeTestFile(t, "db.toml", []byte("[database]\nhost = \"db.example.com\"\nport = 3306\n"))

	filePairs := []config.FilePair{
		{PlaintextPath: "app.toml", EncryptedPath: "app.enc.toml.yaml"},
		{PlaintextPath: "db.toml", EncryptedPath: "db.enc.toml.yaml"},
	}

	summary, err := BatchEncrypt(BatchOptions{
		FilePairs: filePairs,
		KeyFile:   env.keyFile,
		PublicKey: env.publicKey,
	})
	require.NoError(t, err)

	assert.Equal(t, 2, summary.TotalFiles)
	assert.Equal(t, 2, summary.SuccessCount)

	for _, pair := range filePairs {
		_, err := os.Stat(pair.EncryptedPath)
		assert.NoError(t, err, "%s should exist", pair.EncryptedPath)
	}
}

func TestIntegration_TOML_BatchEncrypt_Parallel(t *testing.T) {
	skipIfNoRemarshal(t)
	env := setupIntegrationEnv(t)

	// Create 5 TOML files
	for i := 1; i <= 5; i++ {
		content := fmt.Sprintf("[config%d]\nkey = \"value%d\"\n", i, i)
		writeTestFile(t, fmt.Sprintf("par%d.toml", i), []byte(content))
	}

	summary, err := BatchEncrypt(BatchOptions{
		InputDir:     ".",
		Pattern:      "par*.toml",
		OutputSuffix: ".enc.toml.yaml",
		KeyFile:      env.keyFile,
		PublicKey:    env.publicKey,
		Parallel:     3,
	})
	require.NoError(t, err)

	assert.Equal(t, 5, summary.TotalFiles)
	assert.Equal(t, 5, summary.SuccessCount)
	assert.Equal(t, 0, summary.FailedCount)

	// Verify all output files exist
	for i := 1; i <= 5; i++ {
		_, err := os.Stat(fmt.Sprintf("par%d.enc.toml.yaml", i))
		assert.NoError(t, err, "par%d.enc.toml.yaml should exist", i)
	}
}

// ============================================================================
// Group 3: TOML Edit（需要 remarshal）
// ============================================================================

func TestIntegration_TOML_Edit_NoChange(t *testing.T) {
	skipIfNoRemarshal(t)
	env := setupIntegrationEnv(t)

	writeTestFile(t, "edit.toml", sampleTOML())

	err := Encrypt("edit.toml", "edit.enc.toml.yaml", env.keyFile, env.publicKey, "", false)
	require.NoError(t, err)

	// Record original encrypted content
	originalEnc, err := os.ReadFile("edit.enc.toml.yaml")
	require.NoError(t, err)

	// Create noop editor
	editor := mockEditorScript(t, "noop")

	// Edit with noop editor — should skip re-encryption
	err = Edit("edit.enc.toml.yaml", editor, env.keyFile)
	require.NoError(t, err)

	// Verify encrypted file is unchanged
	currentEnc, err := os.ReadFile("edit.enc.toml.yaml")
	require.NoError(t, err)
	assert.Equal(t, originalEnc, currentEnc)
}

func TestIntegration_TOML_Edit_WithChange(t *testing.T) {
	skipIfNoRemarshal(t)
	env := setupIntegrationEnv(t)

	writeTestFile(t, "edit.toml", sampleTOML())

	err := Encrypt("edit.toml", "edit.enc.toml.yaml", env.keyFile, env.publicKey, "", false)
	require.NoError(t, err)

	// Create modify editor
	editor := mockEditorScript(t, "modify")

	// Edit with modify editor — should re-encrypt
	err = Edit("edit.enc.toml.yaml", editor, env.keyFile)
	require.NoError(t, err)

	// Decrypt and verify the change
	err = Decrypt("edit.enc.toml.yaml", "edit.dec.toml", env.keyFile, "", false)
	require.NoError(t, err)

	decContent, err := os.ReadFile("edit.dec.toml")
	require.NoError(t, err)
	assert.Contains(t, string(decContent), "modified_value")
	assert.NotContains(t, string(decContent), "secret123")
}

// ============================================================================
// Group 4: TOML 端到端工作流（需要 remarshal）
// ============================================================================

func TestIntegration_TOML_FullWorkflow(t *testing.T) {
	skipIfNoRemarshal(t)
	env := setupIntegrationEnv(t)

	inputFile := "wrangler.toml"
	outputFile := "wrangler.enc.toml.yaml"

	// Step 1: setupAgeKey — already done in setupIntegrationEnv

	// Step 2: SavePublicKeyToConfig
	err := project.SavePublicKeyToConfig(env.publicKey, []config.FilePair{
		{PlaintextPath: inputFile, EncryptedPath: outputFile},
	})
	require.NoError(t, err)
	_, err = os.Stat(".yewseal.toml")
	require.NoError(t, err)

	// Step 3: UpdateSopsYaml
	err = project.UpdateSopsYaml(outputFile, env.publicKey, false)
	require.NoError(t, err)
	_, err = os.Stat(".sops.yaml")
	require.NoError(t, err)

	// Step 4: Encrypt
	writeTestFile(t, inputFile, sampleTOML())
	err = Encrypt(inputFile, outputFile, env.keyFile, env.publicKey, "", false)
	require.NoError(t, err)

	// Step 5: Decrypt
	decryptedFile := "wrangler.dec.toml"
	err = Decrypt(outputFile, decryptedFile, env.keyFile, "", false)
	require.NoError(t, err)

	// Step 6: Verify content integrity
	decContent, err := os.ReadFile(decryptedFile)
	require.NoError(t, err)
	assert.Contains(t, string(decContent), "localhost")
	assert.Contains(t, string(decContent), "secret123")
	assert.Contains(t, string(decContent), "5432")
}

func TestIntegration_TOML_ConfigDrivenBatch(t *testing.T) {
	skipIfNoRemarshal(t)
	env := setupIntegrationEnv(t)

	// Create test files
	writeTestFile(t, "app.toml", sampleTOML())
	writeTestFile(t, "db.toml", []byte("[database]\nhost = \"db.example.com\"\nport = 3306\npassword = \"dbpass\"\n"))

	// Create .yewseal.toml with file pairs
	configContent := `[encryption]

[[encryption.files]]
plaintext = "app.toml"
encrypted = "app.enc.toml.yaml"

[[encryption.files]]
plaintext = "db.toml"
encrypted = "db.enc.toml.yaml"

[key]
file_path = ".age/keys.txt"
public_key = "` + env.publicKey + `"
`
	writeTestFile(t, ".yewseal.toml", []byte(configContent))

	// Load config and use FilePairs
	cfg, err := config.LoadConfig()
	require.NoError(t, err)
	require.Len(t, cfg.GetFiles(), 2)

	// Batch encrypt using config-driven file pairs
	summary, err := BatchEncrypt(BatchOptions{
		FilePairs: cfg.GetFiles(),
		KeyFile:   env.keyFile,
		PublicKey: env.publicKey,
	})
	require.NoError(t, err)
	assert.Equal(t, 2, summary.SuccessCount)

	// Batch decrypt
	decSummary, err := BatchDecrypt(BatchOptions{
		FilePairs: cfg.GetFiles(),
		KeyFile:   env.keyFile,
	})
	require.NoError(t, err)
	assert.Equal(t, 2, decSummary.SuccessCount)

	// Verify decrypted content
	appContent, err := os.ReadFile("app.toml")
	require.NoError(t, err)
	assert.Contains(t, string(appContent), "secret123")

	dbContent, err := os.ReadFile("db.toml")
	require.NoError(t, err)
	assert.Contains(t, string(dbContent), "db.example.com")
	assert.Contains(t, string(dbContent), "dbpass")
}

// ============================================================================
// Group 5: 其他格式 Round-Trip（无外部依赖）
// ============================================================================

func TestIntegration_YAML_RoundTrip(t *testing.T) {
	env := setupIntegrationEnv(t)

	writeTestFile(t, "config.yaml", sampleYAML())

	err := Encrypt("config.yaml", "config.enc.yaml", env.keyFile, env.publicKey, "", false)
	require.NoError(t, err)

	err = Decrypt("config.enc.yaml", "config.dec.yaml", env.keyFile, "", false)
	require.NoError(t, err)

	decContent, err := os.ReadFile("config.dec.yaml")
	require.NoError(t, err)
	assert.Contains(t, string(decContent), "localhost")
	assert.Contains(t, string(decContent), "secret123")
}

func TestIntegration_JSON_RoundTrip(t *testing.T) {
	env := setupIntegrationEnv(t)

	writeTestFile(t, "config.json", sampleJSON())

	err := Encrypt("config.json", "config.enc.json", env.keyFile, env.publicKey, "", false)
	require.NoError(t, err)

	err = Decrypt("config.enc.json", "config.dec.json", env.keyFile, "", false)
	require.NoError(t, err)

	decContent, err := os.ReadFile("config.dec.json")
	require.NoError(t, err)
	assert.Contains(t, string(decContent), "localhost")
	assert.Contains(t, string(decContent), "secret123")
}

func TestIntegration_ENV_RoundTrip(t *testing.T) {
	env := setupIntegrationEnv(t)

	writeTestFile(t, "config.env", sampleENV())

	err := Encrypt("config.env", "config.enc.env", env.keyFile, env.publicKey, "", false)
	require.NoError(t, err)

	err = Decrypt("config.enc.env", "config.dec.env", env.keyFile, "", false)
	require.NoError(t, err)

	decContent, err := os.ReadFile("config.dec.env")
	require.NoError(t, err)
	assert.Contains(t, string(decContent), "DATABASE_HOST=localhost")
	assert.Contains(t, string(decContent), "DATABASE_PASSWORD=secret123")
}

func TestIntegration_INI_RoundTrip(t *testing.T) {
	env := setupIntegrationEnv(t)

	writeTestFile(t, "config.ini", sampleINI())

	err := Encrypt("config.ini", "config.enc.ini", env.keyFile, env.publicKey, "", false)
	require.NoError(t, err)

	err = Decrypt("config.enc.ini", "config.dec.ini", env.keyFile, "", false)
	require.NoError(t, err)

	decContent, err := os.ReadFile("config.dec.ini")
	require.NoError(t, err)
	assert.Contains(t, string(decContent), "localhost")
	assert.Contains(t, string(decContent), "secret123")
}

// ============================================================================
// Group 6: 通用边界和错误处理
// ============================================================================

func TestIntegration_Batch_PartialFailure(t *testing.T) {
	env := setupIntegrationEnv(t)

	// 2 valid YAML files + 1 invalid
	writeTestFile(t, "valid1.yaml", sampleYAML())
	writeTestFile(t, "valid2.yaml", sampleYAML())
	writeTestFile(t, "invalid.yaml", []byte("{{invalid yaml"))

	summary, err := BatchEncrypt(BatchOptions{
		InputDir:     ".",
		Pattern:      "*.yaml",
		OutputSuffix: ".enc.yaml",
		KeyFile:      env.keyFile,
		PublicKey:    env.publicKey,
	})

	assert.Error(t, err)
	require.NotNil(t, summary)
	assert.Equal(t, 3, summary.TotalFiles)
	assert.Equal(t, 2, summary.SuccessCount)
	assert.Equal(t, 1, summary.FailedCount)
}

func TestIntegration_Batch_EmptyDir(t *testing.T) {
	env := setupIntegrationEnv(t)

	require.NoError(t, os.MkdirAll("empty", 0755))

	_, err := BatchEncrypt(BatchOptions{
		InputDir:     "empty",
		Pattern:      "*.yaml",
		OutputSuffix: ".enc.yaml",
		KeyFile:      env.keyFile,
		PublicKey:    env.publicKey,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no files found")
}

func TestIntegration_UnsupportedFormat(t *testing.T) {
	env := setupIntegrationEnv(t)

	writeTestFile(t, "config.xml", []byte("<config><key>value</key></config>"))

	err := Encrypt("config.xml", "config.enc.xml", env.keyFile, env.publicKey, "", false)
	assert.Error(t, err)

	var unsupportedErr *errx.UnsupportedFormatError
	assert.ErrorAs(t, err, &unsupportedErr)
}

func TestIntegration_WrongKey(t *testing.T) {
	env := setupIntegrationEnv(t)

	writeTestFile(t, "config.yaml", sampleYAML())

	// Encrypt with env's key
	err := Encrypt("config.yaml", "config.enc.yaml", env.keyFile, env.publicKey, "", false)
	require.NoError(t, err)

	// Generate a different key pair using embedded age library
	identity2, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	wrongKeyDir := ".wrong-age"
	require.NoError(t, os.MkdirAll(wrongKeyDir, 0700))
	wrongKeyFile := filepath.Join(wrongKeyDir, "keys.txt")
	wrongKeyContent := fmt.Sprintf("# public key: %s\n%s\n",
		identity2.Recipient().String(),
		identity2.String())
	require.NoError(t, os.WriteFile(wrongKeyFile, []byte(wrongKeyContent), 0600))

	// Try to decrypt with wrong key
	err = Decrypt("config.enc.yaml", "config.dec.yaml", wrongKeyFile, "", false)
	assert.Error(t, err)
}

func TestIntegration_Edit_FileNotExist(t *testing.T) {
	_ = setupIntegrationEnv(t)

	err := Edit("nonexistent.enc.yaml", "vi", ".age/keys.txt")
	assert.Error(t, err)

	var notFoundErr *errx.NotFoundError
	assert.ErrorAs(t, err, &notFoundErr)
}

func TestGenerateOutputFilename(t *testing.T) {
	tests := []struct {
		name         string
		inputFile    string
		outputDir    string
		outputSuffix string
		mode         string
		expected     string
	}{
		// Encrypt mode
		{
			name:         "encrypt_toml",
			inputFile:    "config.toml",
			outputSuffix: ".enc.toml.yaml",
			mode:         "encrypt",
			expected:     "config.enc.toml.yaml",
		},
		{
			name:         "encrypt_yaml",
			inputFile:    "config.yaml",
			outputSuffix: ".enc.yaml",
			mode:         "encrypt",
			expected:     "config.enc.yaml",
		},
		{
			name:         "encrypt_json",
			inputFile:    "config.json",
			outputSuffix: ".enc.json",
			mode:         "encrypt",
			expected:     "config.enc.json",
		},
		{
			name:         "encrypt_with_output_dir",
			inputFile:    "config.toml",
			outputDir:    "encrypted",
			outputSuffix: ".enc.toml.yaml",
			mode:         "encrypt",
			expected:     filepath.Join("encrypted", "config.enc.toml.yaml"),
		},
		// Decrypt mode
		{
			name:         "decrypt_enc_toml_yaml",
			inputFile:    "config.enc.toml.yaml",
			outputSuffix: ".toml",
			mode:         "decrypt",
			expected:     "config.toml",
		},
		{
			name:         "decrypt_enc_yaml",
			inputFile:    "config.enc.yaml",
			outputSuffix: ".yaml",
			mode:         "decrypt",
			expected:     "config.yaml",
		},
		{
			name:         "decrypt_encrypted_yaml",
			inputFile:    "config.encrypted.yaml",
			outputSuffix: ".yaml",
			mode:         "decrypt",
			expected:     "config.yaml",
		},
		{
			name:         "decrypt_with_output_dir",
			inputFile:    "config.enc.toml.yaml",
			outputDir:    "decrypted",
			outputSuffix: ".toml",
			mode:         "decrypt",
			expected:     filepath.Join("decrypted", "config.toml"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateOutputFilename(tt.inputFile, tt.outputDir, tt.outputSuffix, tt.mode)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ============================================================================
// Group 7: InitProject 流程
// ============================================================================

func TestIntegration_InitProject_Artifacts(t *testing.T) {
	tempDir := t.TempDir()
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tempDir))
	t.Cleanup(func() { os.Chdir(oldWd) })

	// Run InitProject in non-interactive mode
	// createExampleFlag=true (flagSet=true, skips prompt, returns true — but file doesn't exist so just warns)
	// skipSopsConfigFlag=true (flagSet=true, returns false — skips sops config creation)
	err = project.InitProject(false, "wrangler.toml", "wrangler.enc.toml.yaml", true, true)
	require.NoError(t, err)

	// Verify .age/keys.txt was created with valid content
	keyContent, err := os.ReadFile(".age/keys.txt")
	require.NoError(t, err)
	assert.Contains(t, string(keyContent), "AGE-SECRET-KEY-")
	assert.Contains(t, string(keyContent), "# public key: age1")

	// Verify .yewseal.toml was created with correct fields
	configContent, err := os.ReadFile(".yewseal.toml")
	require.NoError(t, err)
	assert.Contains(t, string(configContent), "wrangler.toml")
	assert.Contains(t, string(configContent), "wrangler.enc.toml.yaml")
	assert.Contains(t, string(configContent), "public_key")

	// Verify .gitignore was created with YewSeal entries
	gitignoreContent, err := os.ReadFile(".gitignore")
	require.NoError(t, err)
	assert.Contains(t, string(gitignoreContent), "wrangler.toml")
	assert.Contains(t, string(gitignoreContent), ".age/keys.txt")
}

// ============================================================================
// Group 8: 格式 Override（非标准扩展名）
// ============================================================================

// TestIntegration_FormatOverride_ENVRoundTrip 验证 .dev.vars 这类非标准扩展名文件
// 通过 formatOverride="env" 正确加解密
func TestIntegration_FormatOverride_ENVRoundTrip(t *testing.T) {
	env := setupIntegrationEnv(t)

	writeTestFile(t, ".dev.vars", sampleENV())

	// 加密：扩展名是 .vars，SOPS 无法识别，必须手动指定 env
	err := Encrypt(".dev.vars", ".dev.vars.enc.yaml", env.keyFile, env.publicKey, "env", false)
	require.NoError(t, err)

	encContent, err := os.ReadFile(".dev.vars.enc.yaml")
	require.NoError(t, err)
	assert.Contains(t, string(encContent), "sops_version")
	assert.NotContains(t, string(encContent), "secret123")

	// 解密：输出文件扩展名同样非标准，指定 env
	err = Decrypt(".dev.vars.enc.yaml", ".dev.vars.dec", env.keyFile, "env", false)
	require.NoError(t, err)

	decContent, err := os.ReadFile(".dev.vars.dec")
	require.NoError(t, err)
	assert.Contains(t, string(decContent), "DATABASE_HOST=localhost")
	assert.Contains(t, string(decContent), "DATABASE_PASSWORD=secret123")
}

// TestIntegration_FormatOverride_DotenvAlias 验证 "dotenv" 别名与 "env" 等价
func TestIntegration_FormatOverride_DotenvAlias(t *testing.T) {
	env := setupIntegrationEnv(t)

	writeTestFile(t, "secrets.vars", sampleENV())

	err := Encrypt("secrets.vars", "secrets.vars.enc.yaml", env.keyFile, env.publicKey, "dotenv", false)
	require.NoError(t, err)

	err = Decrypt("secrets.vars.enc.yaml", "secrets.vars.dec", env.keyFile, "dotenv", false)
	require.NoError(t, err)

	decContent, err := os.ReadFile("secrets.vars.dec")
	require.NoError(t, err)
	assert.Contains(t, string(decContent), "DATABASE_PASSWORD=secret123")
}

// TestIntegration_FormatOverride_YAMLExtOverride 验证用 override 把 .conf 文件当 yaml 处理
func TestIntegration_FormatOverride_YAMLExtOverride(t *testing.T) {
	env := setupIntegrationEnv(t)

	writeTestFile(t, "app.conf", sampleYAML())

	err := Encrypt("app.conf", "app.conf.enc.yaml", env.keyFile, env.publicKey, "yaml", false)
	require.NoError(t, err)

	err = Decrypt("app.conf.enc.yaml", "app.conf.dec", env.keyFile, "yaml", false)
	require.NoError(t, err)

	decContent, err := os.ReadFile("app.conf.dec")
	require.NoError(t, err)
	assert.Contains(t, string(decContent), "localhost")
	assert.Contains(t, string(decContent), "secret123")
}

// TestIntegration_FormatOverride_UnknownOverrideFails 验证传入无效 format 时不影响正常错误路径
func TestIntegration_FormatOverride_UnknownOverrideFails(t *testing.T) {
	env := setupIntegrationEnv(t)

	writeTestFile(t, "config.vars", sampleENV())

	// "xml" 不是合法格式，ParseFormat 返回 FormatUnknown，fallback 到扩展名检测
	// .vars 也是 unknown，所以应该报 UnsupportedFormatError
	err := Encrypt("config.vars", "config.vars.enc.yaml", env.keyFile, env.publicKey, "xml", false)
	assert.Error(t, err)

	var unsupportedErr *errx.UnsupportedFormatError
	assert.ErrorAs(t, err, &unsupportedErr)
}

// TestIntegration_FormatOverride_Batch 验证批量模式下 FilePair.Format 生效
func TestIntegration_FormatOverride_Batch(t *testing.T) {
	env := setupIntegrationEnv(t)

	writeTestFile(t, ".dev.vars", sampleENV())
	writeTestFile(t, "config.yaml", sampleYAML())

	// 批量加密：两个文件，一个需要 format override
	_, err := BatchEncrypt(BatchOptions{
		FilePairs: []config.FilePair{
			{PlaintextPath: ".dev.vars", EncryptedPath: ".dev.vars.enc.yaml", Format: "env"},
			{PlaintextPath: "config.yaml", EncryptedPath: "config.enc.yaml"},
		},
		KeyFile:   env.keyFile,
		PublicKey: env.publicKey,
	})
	require.NoError(t, err)

	// 验证两个文件都被加密
	enc1, err := os.ReadFile(".dev.vars.enc.yaml")
	require.NoError(t, err)
	assert.Contains(t, string(enc1), "sops_version")
	assert.NotContains(t, string(enc1), "secret123")

	enc2, err := os.ReadFile("config.enc.yaml")
	require.NoError(t, err)
	assert.Contains(t, string(enc2), "sops:")

	// 批量解密：FilePair 与加密时相同，BatchDecrypt 内部会自动互换 Input/Output
	_, err = BatchDecrypt(BatchOptions{
		FilePairs: []config.FilePair{
			{PlaintextPath: ".dev.vars", EncryptedPath: ".dev.vars.enc.yaml", Format: "env"},
			{PlaintextPath: "config.yaml", EncryptedPath: "config.enc.yaml"},
		},
		KeyFile: env.keyFile,
	})
	require.NoError(t, err)

	dec1, err := os.ReadFile(".dev.vars")
	require.NoError(t, err)
	assert.Contains(t, string(dec1), "DATABASE_PASSWORD=secret123")

	dec2, err := os.ReadFile("config.yaml")
	require.NoError(t, err)
	assert.Contains(t, string(dec2), "secret123")
}
