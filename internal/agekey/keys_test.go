package agekey

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withAgekeyWorkingDir(t *testing.T, dir string) {
	t.Helper()

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(oldWd))
	})
}

func TestExtractPublicKey_Valid(t *testing.T) {
	output := "# public key: age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p"
	result := ExtractPublicKey(output)
	assert.Equal(t, "age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p", result)
}

func TestExtractPublicKey_MultiLine(t *testing.T) {
	output := `# created: 2024-01-01T00:00:00Z
# public key: age1abc123xyz456
AGE-SECRET-KEY-1QQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQ`

	result := ExtractPublicKey(output)
	assert.Equal(t, "age1abc123xyz456", result)
}

func TestExtractPublicKey_WithWhitespace(t *testing.T) {
	output := "  # public key: age1testkey  "
	result := ExtractPublicKey(output)
	assert.Equal(t, "age1testkey", result)
}

func TestExtractPublicKey_NoKey(t *testing.T) {
	output := `# created: 2024-01-01T00:00:00Z
AGE-SECRET-KEY-1QQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQ`

	result := ExtractPublicKey(output)
	assert.Empty(t, result)
}

func TestExtractPublicKey_Empty(t *testing.T) {
	result := ExtractPublicKey("")
	assert.Empty(t, result)
}

func TestExtractPublicKey_OnlyComments(t *testing.T) {
	output := `# some comment
# another comment
# not a public key line`

	result := ExtractPublicKey(output)
	assert.Empty(t, result)
}

func TestExtractPublicKey_MalformedPrefix(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{
			name:   "wrong prefix",
			input:  "# publickey: age1test",
			expect: "",
		},
		{
			name:   "missing colon",
			input:  "# public key age1test",
			expect: "",
		},
		{
			name:   "correct prefix",
			input:  "# public key: age1correct",
			expect: "age1correct",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractPublicKey(tt.input)
			assert.Equal(t, tt.expect, result)
		})
	}
}

// =============================================
// readKeyFile tests
// =============================================

func TestReadKeyFile_Success(t *testing.T) {
	tmpDir := t.TempDir()
	keyFile := filepath.Join(tmpDir, "keys.txt")

	content := `# created: 2024-01-01T00:00:00Z
# public key: age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p
AGE-SECRET-KEY-1QQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQ`

	err := os.WriteFile(keyFile, []byte(content), 0600)
	require.NoError(t, err)

	key, err := GetAgeKey(keyFile)
	require.NoError(t, err)
	assert.Equal(t, "AGE-SECRET-KEY-1QQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQ", key)
}

func TestReadKeyFile_NotExists(t *testing.T) {
	_, err := readKeyFile("/nonexistent/path/to/keys.txt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read key file")
}

func TestReadKeyFile_InvalidContent(t *testing.T) {
	tmpDir := t.TempDir()
	keyFile := filepath.Join(tmpDir, "keys.txt")

	// Write file without a valid AGE-SECRET-KEY line
	content := `# created: 2024-01-01T00:00:00Z
# public key: age1test
some random content`

	err := os.WriteFile(keyFile, []byte(content), 0600)
	require.NoError(t, err)

	_, err = GetAgeKey(keyFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no valid Age secret key found")
}

func TestReadKeyFile_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	keyFile := filepath.Join(tmpDir, "keys.txt")

	err := os.WriteFile(keyFile, []byte(""), 0600)
	require.NoError(t, err)

	_, err = GetAgeKey(keyFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no valid Age secret key found")
}

func TestReadKeyFile_MultipleKeys(t *testing.T) {
	tmpDir := t.TempDir()
	keyFile := filepath.Join(tmpDir, "keys.txt")

	// File with multiple keys - should return the first one
	content := `# public key: age1first
AGE-SECRET-KEY-1FIRSTFIRSTFIRSTFIRSTFIRSTFIRSTFIRSTFIRSTFIRSTFIRSTFIRST
# public key: age1second
AGE-SECRET-KEY-1SECONDSECONDSECONDSECONDSECONDSECONDSECONDSECONDSECONDSECOND`

	err := os.WriteFile(keyFile, []byte(content), 0600)
	require.NoError(t, err)

	key, err := GetAgeKey(keyFile)
	require.NoError(t, err)
	assert.Equal(t, "AGE-SECRET-KEY-1FIRSTFIRSTFIRSTFIRSTFIRSTFIRSTFIRSTFIRSTFIRSTFIRSTFIRST", key)
}

// =============================================
// GetAgeKey tests
// =============================================

func TestGetAgeKey_FromKeyFile(t *testing.T) {
	tmpDir := t.TempDir()
	keyFile := filepath.Join(tmpDir, "keys.txt")

	content := `# public key: age1test
AGE-SECRET-KEY-1TESTTESTTESTTESTTESTTESTTESTTESTTESTTESTTESTTESTTEST`

	err := os.WriteFile(keyFile, []byte(content), 0600)
	require.NoError(t, err)

	key, err := GetAgeKey(keyFile)
	require.NoError(t, err)
	assert.Equal(t, "AGE-SECRET-KEY-1TESTTESTTESTTESTTESTTESTTESTTESTTESTTESTTESTTESTTEST", key)
}

func TestGetAgeKey_FromEnvVar(t *testing.T) {
	expectedKey := "AGE-SECRET-KEY-1ENVENVENVENVENVENVENVENVENVENVENVENVENVENV"
	t.Setenv("SOPS_AGE_KEY", expectedKey)

	key, err := GetAgeKey("")
	require.NoError(t, err)
	assert.Equal(t, expectedKey, key)
}

func TestGetAgeKey_FromEnvFilePath(t *testing.T) {
	t.Setenv("SOPS_AGE_KEY", "")

	tmpDir := t.TempDir()
	keyFile := filepath.Join(tmpDir, "keys.txt")

	content := `# public key: age1test
AGE-SECRET-KEY-1FILEFILEFILEFILEFILEFILEFILEFILEFILEFILEFILEFILE`

	err := os.WriteFile(keyFile, []byte(content), 0600)
	require.NoError(t, err)

	t.Setenv("SOPS_AGE_KEY_FILE", keyFile)

	key, err := GetAgeKey("")
	require.NoError(t, err)
	assert.Equal(t, "AGE-SECRET-KEY-1FILEFILEFILEFILEFILEFILEFILEFILEFILEFILEFILEFILE", key)
}

func TestGetAgeKey_MissingKeyFileFallsBackToEnvVar(t *testing.T) {
	expectedKey := "AGE-SECRET-KEY-1ENVENVENVENVENVENVENVENVENVENVENVENVENVENV"
	t.Setenv("SOPS_AGE_KEY", expectedKey)

	key, err := GetAgeKey("/nonexistent/path/to/keys.txt")
	require.NoError(t, err)
	assert.Equal(t, expectedKey, key)
}

func TestGetAgeKey_MissingKeyFileFallsBackToEnvFilePath(t *testing.T) {
	t.Setenv("SOPS_AGE_KEY", "")

	tmpDir := t.TempDir()
	envKeyFile := filepath.Join(tmpDir, "env-keys.txt")
	content := `AGE-SECRET-KEY-1ENVFILEENVFILEENVFILEENVFILEENVFILEENVFILE`
	require.NoError(t, os.WriteFile(envKeyFile, []byte(content), 0600))
	t.Setenv("SOPS_AGE_KEY_FILE", envKeyFile)

	key, err := GetAgeKey(filepath.Join(tmpDir, "missing-keys.txt"))
	require.NoError(t, err)
	assert.Equal(t, "AGE-SECRET-KEY-1ENVFILEENVFILEENVFILEENVFILEENVFILEENVFILE", key)
}

func TestGetAgeKey_Priority(t *testing.T) {
	t.Setenv("SOPS_AGE_KEY", "AGE-SECRET-KEY-1ENVENVENVENVENVENVENVENVENVENVENVENVENVENV")

	// Create key file
	tmpDir := t.TempDir()
	keyFile := filepath.Join(tmpDir, "keys.txt")
	content := `AGE-SECRET-KEY-1FILEPARAMFILEPARAMFILEPARAMFILEPARAMFILEPARAMFILEPARAMFILE`
	err := os.WriteFile(keyFile, []byte(content), 0600)
	require.NoError(t, err)

	// CLI parameter should take priority over env var
	key, err := GetAgeKey(keyFile)
	require.NoError(t, err)
	assert.Equal(t, "AGE-SECRET-KEY-1FILEPARAMFILEPARAMFILEPARAMFILEPARAMFILEPARAMFILEPARAMFILE", key)
}

func TestGetAgeKey_InvalidKeyFileDoesNotFallBackToEnvVar(t *testing.T) {
	t.Setenv("SOPS_AGE_KEY", "AGE-SECRET-KEY-1ENVENVENVENVENVENVENVENVENVENVENVENVENVENV")

	tmpDir := t.TempDir()
	keyFile := filepath.Join(tmpDir, "keys.txt")
	require.NoError(t, os.WriteFile(keyFile, []byte("invalid key content"), 0600))

	_, err := GetAgeKey(keyFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no valid Age secret key found")
}

func TestGetAgeKey_NoKeyFound(t *testing.T) {
	envVars := []string{"SOPS_AGE_KEY", "SOPS_AGE_KEY_FILE", "SOPS_AGE_KEY_CMD"}
	for _, v := range envVars {
		t.Setenv(v, "")
	}

	withAgekeyWorkingDir(t, t.TempDir())

	_, err := GetAgeKey("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no Age key found")
}

// =============================================
// GetPublicKey tests
// =============================================

func TestGetPublicKey_FromParam(t *testing.T) {
	providedKey := "age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p"

	key, err := GetPublicKey(providedKey, "", false)
	require.NoError(t, err)
	assert.Equal(t, providedKey, key)
}

func TestGetPublicKey_FromEnv(t *testing.T) {
	expectedKey := "age1envenvenvenvenvenvenvenvenvenvenvenvenvenvenvenvenvenvenvenv"
	t.Setenv("SOPS_AGE_RECIPIENTS", expectedKey)

	key, err := GetPublicKey("", "", false)
	require.NoError(t, err)
	assert.Equal(t, expectedKey, key)
}

func TestGetPublicKey_FromKeyFile(t *testing.T) {
	t.Setenv("SOPS_AGE_RECIPIENTS", "")

	tmpDir := t.TempDir()
	withAgekeyWorkingDir(t, tmpDir)

	// Create key file with public key in comment
	keyFile := filepath.Join(tmpDir, "keys.txt")
	content := `# created: 2024-01-01T00:00:00Z
# public key: age1filekeyfilekeyfilekeyfilekeyfilekeyfilekeyfilekeyfilekey
AGE-SECRET-KEY-1TESTTESTTESTTESTTESTTESTTESTTESTTESTTESTTESTTESTTEST`

	err := os.WriteFile(keyFile, []byte(content), 0600)
	require.NoError(t, err)

	key, err := GetPublicKey("", keyFile, false)
	require.NoError(t, err)
	assert.Equal(t, "age1filekeyfilekeyfilekeyfilekeyfilekeyfilekeyfilekeyfilekey", key)
}

func TestGetPublicKey_Priority_ParamOverEnv(t *testing.T) {
	t.Setenv("SOPS_AGE_RECIPIENTS", "age1envenvenvenvenvenvenvenvenvenvenvenvenvenvenvenvenvenvenvenv")

	// Param should have higher priority
	providedKey := "age1paramprovided"
	key, err := GetPublicKey(providedKey, "", false)
	require.NoError(t, err)
	assert.Equal(t, providedKey, key)
}

func TestGetPublicKey_NoKeyFound(t *testing.T) {
	t.Setenv("SOPS_AGE_RECIPIENTS", "")

	withAgekeyWorkingDir(t, t.TempDir())

	_, err := GetPublicKey("", "", false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no public key found")
}

func TestGetPublicKey_VerboseMode(t *testing.T) {
	providedKey := "age1verbosetest"

	// Should not error with verbose mode
	key, err := GetPublicKey(providedKey, "", true)
	require.NoError(t, err)
	assert.Equal(t, providedKey, key)
}

func TestGetPublicKey_InvalidKeyFile(t *testing.T) {
	t.Setenv("SOPS_AGE_RECIPIENTS", "")

	tmpDir := t.TempDir()
	withAgekeyWorkingDir(t, tmpDir)

	// Create key file without public key comment
	keyFile := filepath.Join(tmpDir, "keys.txt")
	content := `# created: 2024-01-01T00:00:00Z
AGE-SECRET-KEY-1TESTTESTTESTTESTTESTTESTTESTTESTTESTTESTTESTTESTTEST`

	err := os.WriteFile(keyFile, []byte(content), 0600)
	require.NoError(t, err)

	_, err = GetPublicKey("", keyFile, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no public key found")
}
