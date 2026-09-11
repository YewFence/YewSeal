package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"filippo.io/age"
	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/sopsx"
	toml "github.com/pelletier/go-toml/v2"
	"github.com/stretchr/testify/require"
)

func TestCLIProcessingOutcomes(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "yews.exe")
	output, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput()
	require.NoError(t, err, "%s", output)
	for _, name := range []string{"AGE_KEY_FILE", "YEWSEAL_AGE_IDENTITIES", "SOPS_AGE_KEY", "SOPS_AGE_KEY_FILE", "SOPS_AGE_KEY_CMD", "SOPS_OUTPUT_FILE", "YEWSEAL_STRICT"} {
		t.Setenv(name, "")
		require.NoError(t, os.Unsetenv(name))
	}
	for _, tc := range []struct {
		name, command, scenario, strictEnv string
		flags                              []string
		code                               int
		different                          bool
	}{
		{name: "decrypt-complete-strict", command: "decrypt", scenario: "complete", flags: []string{"--strict"}},
		{name: "decrypt-partial", command: "decrypt"},
		{name: "decrypt-strict", command: "decrypt", flags: []string{"--strict"}, code: 1},
		{name: "decrypt-env-strict", command: "decrypt", strictEnv: "true", code: 1},
		{name: "decrypt-env-override", command: "decrypt", strictEnv: "true", flags: []string{"--strict=false"}},
		{name: "decrypt-all-skipped", command: "decrypt", scenario: "all-skipped", code: 1},
		{name: "decrypt-error-continues", command: "decrypt", scenario: "broken", code: 1},
		{name: "diff-partial-same", command: "diff"},
		{name: "diff-complete-same", command: "diff", scenario: "complete"},
		{name: "diff-complete-different", command: "diff", scenario: "complete", different: true},
		{name: "diff-strict-complete-different", command: "diff", scenario: "complete", flags: []string{"--strict"}, different: true},
		{name: "diff-partial-different", command: "diff", different: true},
		{name: "diff-strict", command: "diff", flags: []string{"--strict"}, code: 1},
		{name: "diff-strict-partial-different", command: "diff", flags: []string{"--strict"}, different: true, code: 1},
		{name: "diff-env-strict", command: "diff", strictEnv: "true", code: 1},
		{name: "diff-explicit-overrides-invalid", command: "diff", strictEnv: "bad", flags: []string{"--strict=false"}},
		{name: "diff-all-skipped", command: "diff", scenario: "all-skipped"},
		{name: "diff-all-skipped-strict", command: "diff", scenario: "all-skipped", flags: []string{"--strict"}, code: 1},
		{name: "diff-errors-dominate-differences", command: "diff", scenario: "broken", different: true, code: 1},
		{name: "diff-missing-plaintext", command: "diff", scenario: "missing-plaintext"},
		{name: "diff-missing-plaintext-strict", command: "diff", scenario: "missing-plaintext", flags: []string{"--strict"}},
		{name: "diff-plaintext-discovery", command: "diff", scenario: "group-plaintext", flags: []string{"--strict"}},
		{name: "diff-verbose", command: "diff", flags: []string{"--verbose"}},
		{name: "diff-bad-env", command: "diff", strictEnv: "bad", code: 1},
		{name: "decrypt-bad-env", command: "decrypt", strictEnv: "bad", code: 1},
		{name: "diff-bad-flag", command: "diff", flags: []string{"--unknown-option"}, code: 1},
		{name: "view-does-not-skip", command: "view", flags: []string{"other.enc.yaml"}, strictEnv: "bad", code: 1},
		{name: "view-verbose-clean-stdout", command: "view", flags: []string{"good.enc.yaml", "--verbose"}},
		{name: "edit-does-not-skip", command: "edit", flags: []string{"--file", "other.enc.yaml"}, strictEnv: "bad", code: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0755))
			require.NoError(t, os.Mkdir(filepath.Join(dir, ".age"), 0700))
			owner, err := age.GenerateX25519Identity()
			require.NoError(t, err)
			other, err := age.GenerateX25519Identity()
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(filepath.Join(dir, ".age/keys.txt"), []byte(owner.String()), 0600))
			plain := []byte("token: value\n")
			for name, recipient := range map[string]string{"good.enc.yaml": owner.Recipient().String(), "other.enc.yaml": other.Recipient().String()} {
				data, err := sopsx.Encrypt(plain, "yaml", []string{recipient})
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(filepath.Join(dir, name), data, 0600))
			}
			pairs := []config.FilePair{
				{PlaintextPath: "good.yaml", EncryptedPath: "good.enc.yaml", Format: "yaml"},
				{PlaintextPath: "private/other.yaml", EncryptedPath: "other.enc.yaml", Format: "yaml"},
			}
			if tc.command == "diff" && tc.scenario != "group-plaintext" && tc.scenario != "missing-plaintext" {
				require.NoError(t, os.Mkdir(filepath.Join(dir, "private"), 0700))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "private/other.yaml"), plain, 0600))
			}
			if tc.command == "diff" && tc.scenario != "missing-plaintext" && tc.scenario != "all-skipped" {
				local := plain
				if tc.different {
					local = []byte("token: changed\n")
				}
				require.NoError(t, os.WriteFile(filepath.Join(dir, "good.yaml"), local, 0600))
			}
			if tc.scenario == "all-skipped" {
				pairs = pairs[1:]
			}
			if tc.scenario == "complete" || tc.scenario == "missing-plaintext" {
				pairs = pairs[:1]
			}
			if tc.scenario == "broken" {
				if tc.command == "diff" {
					require.NoError(t, os.WriteFile(filepath.Join(dir, "broken.yaml"), plain, 0600))
				}
				require.NoError(t, os.WriteFile(filepath.Join(dir, "broken.enc.yaml"), []byte("broken"), 0600))
				pairs = append([]config.FilePair{{PlaintextPath: "broken.yaml", EncryptedPath: "broken.enc.yaml", Format: "yaml"}}, pairs...)
			}
			defaults := []string{"owner"}
			cfg := config.Config{Encryption: config.EncryptionConfig{Files: pairs}, Recipients: config.RecipientConfig{Defaults: &defaults, Registry: map[string]string{"owner": owner.Recipient().String()}}}
			if tc.scenario == "group-plaintext" {
				cfg.Encryption.Files = nil
				cfg.Encryption.Groups = []config.GroupConfig{{Patterns: []string{"*.yaml"}}}
				require.NoError(t, os.WriteFile(filepath.Join(dir, "new.yaml"), plain, 0600))
			}
			data, err := toml.Marshal(cfg)
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(filepath.Join(dir, ".yewseal.toml"), data, 0600))
			cmd := exec.Command(binary, append([]string{tc.command}, tc.flags...)...)
			cmd.Dir = dir
			if tc.strictEnv != "" {
				cmd.Env = append(os.Environ(), "YEWSEAL_STRICT="+tc.strictEnv)
			}
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			err = cmd.Run()
			if tc.code == 0 {
				require.NoError(t, err, "%s\n%s", &stdout, &stderr)
			} else {
				var exit *exec.ExitError
				require.ErrorAs(t, err, &exit, "%s\n%s", &stdout, &stderr)
				require.Equal(t, tc.code, exit.ExitCode(), "%s\n%s", &stdout, &stderr)
			}
			if tc.name == "diff-bad-env" || tc.name == "decrypt-bad-env" || tc.name == "diff-bad-flag" {
				require.NotContains(t, stderr.String(), "Summary")
				return
			}
			if tc.command == "view" || tc.command == "edit" {
				if tc.code != 0 {
					require.Contains(t, stderr.String(), "no matching age identity")
				} else {
					require.Equal(t, plain, stdout.Bytes())
					require.Contains(t, stderr.String(), "Selected 1")
					require.NoFileExists(t, filepath.Join(dir, "good.yaml"))
					require.NoFileExists(t, filepath.Join(dir, ".gitignore"))
				}
				require.NotContains(t, stderr.String(), "invalid YEWSEAL_STRICT")
				return
			}
			if tc.command == "diff" {
				if tc.scenario == "complete" || tc.scenario == "missing-plaintext" || tc.scenario == "group-plaintext" {
					require.NotContains(t, stderr.String(), "Comparison incomplete")
					if tc.scenario == "complete" {
						require.Contains(t, stderr.String(), "0 skipped")
					} else {
						require.Contains(t, stderr.String(), "1 missing input, 0 no matching identity")
					}
				} else {
					require.Contains(t, stderr.String(), "Comparison incomplete")
					require.Contains(t, stderr.String(), "other.enc.yaml")
					require.Contains(t, stderr.String(), "1 skipped")
				}
				require.NotContains(t, stdout.String(), "Summary")
				if tc.different {
					require.Contains(t, stdout.String(), "--- ")
				} else {
					require.Empty(t, stdout.String())
				}
				if tc.scenario == "group-plaintext" {
					require.Contains(t, stderr.String(), "1 compared, 1 skipped (1 missing input, 0 no matching identity), 0 failed (2 selected)")
					require.NotContains(t, stderr.String(), "other.enc.yaml")
				}
				require.NoFileExists(t, filepath.Join(dir, ".gitignore"))
				require.NoFileExists(t, filepath.Join(dir, ".sops.yaml"))
			} else {
				require.NoDirExists(t, filepath.Join(dir, "private"))
				ignore, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
				require.NoError(t, err)
				if tc.scenario != "complete" {
					require.Contains(t, string(ignore), "private/other.yaml")
				}
				if tc.scenario != "all-skipped" {
					written, err := os.ReadFile(filepath.Join(dir, "good.yaml"))
					require.NoError(t, err)
					require.Equal(t, plain, written)
				}
			}
		})
	}
}
