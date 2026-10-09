package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"filippo.io/age"
	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/sopsx"
	"github.com/stretchr/testify/require"
)

func TestDecryptDeliveryArgumentErrorsPrecedeConfig(t *testing.T) {
	for _, tc := range []struct {
		name    string
		args    []string
		env     map[string]string
		message string
	}{
		{"flags-force", []string{"--output", "out", "--force"}, nil, "conflicts"},
		{"env-output", []string{"--force"}, map[string]string{"YEWSEAL_DECRYPT_OUTPUT": "out"}, "conflicts"},
		{"env-force", []string{"--output", "out"}, map[string]string{"YEWSEAL_DECRYPT_FORCE": "true"}, "conflicts"},
		{"both-env", nil, map[string]string{"YEWSEAL_DECRYPT_OUTPUT": "out", "YEWSEAL_DECRYPT_FORCE": "true"}, "conflicts"},
		{"flags-inplace", []string{"--output", "out", "--inplace"}, nil, "conflicts"},
		{"env-inplace", []string{"--output", "out"}, map[string]string{"YEWSEAL_DECRYPT_INPLACE": "true"}, "conflicts"},
		{"output-env-inplace-flag", []string{"--inplace"}, map[string]string{"YEWSEAL_DECRYPT_OUTPUT": "out"}, "conflicts"},
		{"empty-output", []string{"--output="}, nil, "requires a directory"},
		{"removed-short-option", []string{"-o", "out"}, nil, "unknown shorthand flag"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for key, value := range tc.env {
				t.Setenv(key, value)
			}
			root := newRootCommand("test", func() (*config.Config, error) {
				t.Fatal("argument errors must precede config loading")
				return nil, nil
			})
			quietCommand(root, append([]string{"decrypt"}, tc.args...))
			err := root.Execute()
			require.ErrorContains(t, err, tc.message)
			require.Equal(t, 2, ExitCode(root, err))
		})
	}
}

func TestDecryptDeliveryEnvironmentPrecedenceAndFalseFlags(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		env  map[string]string
	}{
		{"env-output", nil, nil},
		{"explicit-false", []string{"--force=false", "--strict=false", "--inplace=false"}, map[string]string{"YEWSEAL_DECRYPT_FORCE": "true", "YEWSEAL_DECRYPT_STRICT": "true", "YEWSEAL_DECRYPT_INPLACE": "true"}},
		{"env-false", nil, map[string]string{"YEWSEAL_DECRYPT_FORCE": "false", "YEWSEAL_DECRYPT_INPLACE": "false"}},
		{"flag-overrides-output", []string{"--output", "delivery"}, map[string]string{"YEWSEAL_DECRYPT_OUTPUT": "missing"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cwd := t.TempDir()
			t.Chdir(cwd)
			require.NoError(t, os.Mkdir("delivery", 0755))
			t.Setenv("YEWSEAL_DECRYPT_OUTPUT", "delivery")
			for key, value := range tc.env {
				t.Setenv(key, value)
			}
			loads := 0
			root := newRootCommand("test", func() (*config.Config, error) {
				loads++
				return &config.Config{CurrentDir: cwd, ProjectRoot: cwd}, nil
			})
			quietCommand(root, append([]string{"decrypt"}, tc.args...))
			require.NoError(t, root.Execute())
			require.Equal(t, 1, loads)
			require.NoFileExists(t, ".gitignore")
		})
	}
}

func TestDecryptDeliveryCLIInplaceEnvironmentAndStrictExit(t *testing.T) {
	cwd := t.TempDir()
	t.Chdir(cwd)
	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	ciphertext, err := sopsx.Encrypt([]byte("token: secret\n"), "yaml", []string{identity.Recipient().String()})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile("secret.enc.yaml", ciphertext, 0644))
	cfg := &config.Config{CurrentDir: cwd, ProjectRoot: cwd, Encryption: config.EncryptionConfig{Files: []config.FilePair{{PlaintextPath: "secret.yaml", EncryptedPath: "secret.enc.yaml", PlaintextMode: "delivery"}}}}
	for _, strict := range []bool{false, true} {
		t.Setenv("YEWSEAL_DECRYPT_STRICT", "true")
		args := []string{"decrypt", "--output", t.TempDir()}
		if !strict {
			args = append(args, "--strict=false")
		}
		t.Setenv("YEWSEAL_AGE_IDENTITIES", "")
		t.Setenv("SOPS_AGE_KEY", "")
		t.Setenv("SOPS_AGE_KEY_FILE", "")
		t.Setenv("YEWSEAL_KEY_FILE", "")
		t.Setenv("SOPS_AGE_KEY_CMD", "")
		root := newRootCommand("test", func() (*config.Config, error) { return cfg, nil })
		var stdout, stderr bytes.Buffer
		root.SetOut(&stdout)
		root.SetErr(&stderr)
		root.SetArgs(args)
		executed, err := root.ExecuteC()
		if strict {
			require.Error(t, err)
			require.Equal(t, 1, ExitCode(executed, err))
		} else {
			require.NoError(t, err)
		}
		require.Empty(t, stdout.String())
		require.Contains(t, stderr.String(), "no age identity is available")
	}
	t.Setenv("YEWSEAL_DECRYPT_INPLACE", "true")
	t.Setenv("SOPS_AGE_KEY", identity.String())
	root := newRootCommand("test", func() (*config.Config, error) { return cfg, nil })
	quietCommand(root, []string{"decrypt"})
	require.NoError(t, root.Execute())
	require.FileExists(t, filepath.Join(cwd, "secret.yaml"))
}
