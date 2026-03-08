package crypto

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"filippo.io/age"
	"github.com/stretchr/testify/require"
)

// testEnvironment holds the test environment configuration
type testEnvironment struct {
	tempDir    string
	publicKey  string
	privateKey string
	keyFile    string
}

// setupIntegrationEnv creates an isolated test environment with Age keys
// using the embedded filippo.io/age library (no external age-keygen needed)
func setupIntegrationEnv(t *testing.T) *testEnvironment {
	t.Helper()

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

	// Generate Age key using embedded library
	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err, "Failed to generate age key")

	publicKey := identity.Recipient().String()
	privateKey := identity.String()

	// Write key file in standard age-keygen format
	keyFile := filepath.Join(ageDir, "keys.txt")
	keyContent := fmt.Sprintf("# created: %s\n# public key: %s\n%s\n",
		time.Now().UTC().Format(time.RFC3339),
		publicKey,
		privateKey)
	err = os.WriteFile(keyFile, []byte(keyContent), 0600)
	require.NoError(t, err)

	return &testEnvironment{
		tempDir:    tempDir,
		publicKey:  publicKey,
		privateKey: privateKey,
		keyFile:    keyFile,
	}
}

// skipIfNoRemarshal skips the test if toml2yaml/yaml2toml are not available
func skipIfNoRemarshal(t *testing.T) {
	t.Helper()
	for _, tool := range []string{"toml2yaml", "yaml2toml"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("Skipping test: %s is not installed (install remarshal for TOML support)", tool)
		}
	}
}

// ============================================================================
// Sample data functions
// ============================================================================

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

// sampleYAML returns a sample YAML content for testing
func sampleYAML() []byte {
	return []byte(`database:
  host: localhost
  port: 5432
  password: secret123
server:
  name: test
  enabled: true
`)
}

// sampleJSON returns a sample JSON content for testing
func sampleJSON() []byte {
	return []byte(`{
  "database": {
    "host": "localhost",
    "port": 5432,
    "password": "secret123"
  },
  "server": {
    "name": "test",
    "enabled": true
  }
}`)
}

// sampleENV returns a sample .env content for testing
func sampleENV() []byte {
	return []byte(`DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_PASSWORD=secret123
SERVER_NAME=test
`)
}

// sampleINI returns a sample INI content for testing
func sampleINI() []byte {
	return []byte(`[database]
host = localhost
port = 5432
password = secret123

[server]
name = test
enabled = true
`)
}

// writeTestFile creates a test file with the given name and content
func writeTestFile(t *testing.T, name string, content []byte) {
	t.Helper()
	dir := filepath.Dir(name)
	if dir != "." && dir != "" {
		require.NoError(t, os.MkdirAll(dir, 0755))
	}
	require.NoError(t, os.WriteFile(name, content, 0644))
}

// mockEditorScript creates a mock editor script and returns the editor command string.
// mode: "noop" (no changes) or "modify" (replaces secret123 with modified_value)
func mockEditorScript(t *testing.T, mode string) string {
	t.Helper()

	var scriptName, scriptContent, editorCmd string

	if runtime.GOOS == "windows" {
		scriptName = "mock-editor.ps1"
		switch mode {
		case "noop":
			scriptContent = "exit 0"
		case "modify":
			scriptContent = "$file = $args[0]\n" +
				"$content = Get-Content $file -Raw\n" +
				"$content = $content -replace 'secret123', 'modified_value'\n" +
				"Set-Content -Path $file -Value $content -NoNewline"
		default:
			t.Fatalf("unknown editor mode: %s", mode)
		}
		require.NoError(t, os.WriteFile(scriptName, []byte(scriptContent), 0755))
		absPath, err := filepath.Abs(scriptName)
		require.NoError(t, err)
		editorCmd = "pwsh -File " + absPath
	} else {
		scriptName = "mock-editor.sh"
		switch mode {
		case "noop":
			scriptContent = "#!/bin/sh\nexit 0"
		case "modify":
			scriptContent = "#!/bin/sh\nsed -i 's/secret123/modified_value/' \"$1\""
		default:
			t.Fatalf("unknown editor mode: %s", mode)
		}
		require.NoError(t, os.WriteFile(scriptName, []byte(scriptContent), 0755))
		absPath, err := filepath.Abs(scriptName)
		require.NoError(t, err)
		editorCmd = "sh " + absPath
	}

	return editorCmd
}
