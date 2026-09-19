package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"filippo.io/age"
	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/sopsx"
	toml "github.com/pelletier/go-toml/v2"
	"github.com/stretchr/testify/require"
)

// buildYews 构建一次真实二进制供所有子进程级 e2e 场景复用。
func buildYews(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "yews.exe")
	ctx, cancel := context.WithTimeout(context.Background(), subprocessTimeout)
	defer cancel()
	output, err := exec.CommandContext(ctx, "go", "build", "-o", binary, ".").CombinedOutput()
	require.NoError(t, err, "%s", output)
	return binary
}

func TestCLIProcessingOutcomes(t *testing.T) {
	binary := buildYews(t)
	clearCommandEnvironment(t)
	for _, tc := range []struct {
		name, command, scenario, strictEnv string
		flags                              []string
		stdin, wantOut                     string
		wantIn, gone, kept                 []string
		code                               int
		different                          bool
	}{
		{name: "decrypt-complete-strict", command: "decrypt", scenario: "complete", flags: []string{"--strict"}},
		{name: "decrypt-partial", command: "decrypt"},
		{name: "decrypt-strict", command: "decrypt", flags: []string{"--strict"}, code: 1},
		{name: "decrypt-env-strict", command: "decrypt", strictEnv: "true", code: 1},
		{name: "decrypt-env-override", command: "decrypt", strictEnv: "true", flags: []string{"--strict=false"}},
		{name: "decrypt-all-skipped", command: "decrypt", scenario: "all-skipped"},
		{name: "decrypt-error-continues", command: "decrypt", scenario: "broken", code: 1},
		{name: "diff-partial-same", command: "diff"},
		{name: "diff-complete-same", command: "diff", scenario: "complete"},
		{name: "diff-complete-different", command: "diff", scenario: "complete", different: true},
		{name: "diff-partial-different", command: "diff", different: true},
		{name: "diff-ignores-strict-env", command: "diff", strictEnv: "true"},
		{name: "diff-ignores-invalid-strict-env", command: "diff", strictEnv: "bad"},
		{name: "diff-all-skipped", command: "diff", scenario: "all-skipped"},
		{name: "diff-errors-dominate-differences", command: "diff", scenario: "broken", different: true, code: 1},
		{name: "diff-missing-plaintext", command: "diff", scenario: "missing-plaintext"},
		{name: "diff-plaintext-discovery", command: "diff", scenario: "group-plaintext"},
		{name: "diff-verbose", command: "diff", flags: []string{"--verbose"}},
		{name: "decrypt-bad-env", command: "decrypt", strictEnv: "bad", code: 2},
		{name: "diff-bad-flag", command: "diff", flags: []string{"--unknown-option"}, code: 2},
		{name: "view-does-not-skip", command: "view", flags: []string{"other.enc.yaml"}, strictEnv: "bad", code: 1},
		{name: "view-verbose-clean-stdout", command: "view", flags: []string{"good.enc.yaml", "--verbose"}},
		{name: "edit-does-not-skip", command: "edit", flags: []string{"--file", "other.enc.yaml"}, strictEnv: "bad", code: 1},
		{name: "clean-removes-matching", command: "clean", scenario: "complete",
			wantIn: []string{"REMOVED good.yaml", "1 removed, 0 already absent, 0 retained, 0 failed (1 selected)"},
			gone:   []string{"good.yaml"}, kept: []string{"good.enc.yaml"}},
		{name: "clean-alias-c", command: "c", scenario: "complete",
			wantIn: []string{"REMOVED good.yaml"}, gone: []string{"good.yaml"}},
		{name: "clean-prompt-no-retains", command: "clean", scenario: "complete", different: true, stdin: "n\n",
			wantIn: []string{"Delete the local plaintext anyway? [y/N]:", "RETAINED good.yaml: plaintext differs from encrypted content"},
			kept:   []string{"good.yaml"}},
		{name: "clean-prompt-yes-removes", command: "clean", scenario: "complete", different: true, stdin: "y\n",
			wantIn: []string{"REMOVED good.yaml"}, gone: []string{"good.yaml"}},
		{name: "clean-prompt-eof-fails", command: "clean", scenario: "complete", different: true, code: 1,
			wantIn: []string{"FAILED good.yaml"}, kept: []string{"good.yaml"}},
		{name: "clean-skip-different-keeps", command: "clean", scenario: "complete", different: true, flags: []string{"--skip-different"},
			wantIn: []string{"RETAINED good.yaml"}, wantOut: "Delete the local plaintext anyway?", kept: []string{"good.yaml"}},
		{name: "clean-force-removes-difference", command: "clean", scenario: "complete", different: true, flags: []string{"--force"},
			wantIn: []string{"REMOVED good.yaml"}, wantOut: "Delete the local plaintext anyway?", gone: []string{"good.yaml"}},
		{name: "clean-force-keeps-unrecoverable", command: "clean", scenario: "broken", flags: []string{"--force"}, code: 1,
			wantIn: []string{"REMOVED good.yaml", "FAILED broken.yaml"}, gone: []string{"good.yaml"}, kept: []string{"broken.yaml"}},
		{name: "clean-error-continues", command: "clean", scenario: "broken", code: 1,
			wantIn: []string{"REMOVED good.yaml", "FAILED broken.yaml"}, gone: []string{"good.yaml"}, kept: []string{"broken.yaml"}},
		{name: "clean-missing-plaintext", command: "clean", scenario: "missing-plaintext", flags: []string{"--verbose"},
			wantIn: []string{"ALREADY ABSENT good.yaml", "0 removed, 1 already absent, 0 retained, 0 failed (1 selected)"},
			kept:   []string{"good.enc.yaml"}},
		{name: "clean-identity-mismatch-fails", command: "clean", code: 1,
			wantIn: []string{"REMOVED good.yaml", "FAILED private/other.yaml"}, gone: []string{"good.yaml"}, kept: []string{"private/other.yaml"}},
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
			if tc.command == "clean" || tc.command == "c" {
				if tc.scenario == "" {
					require.NoError(t, os.Mkdir(filepath.Join(dir, "private"), 0700))
					require.NoError(t, os.WriteFile(filepath.Join(dir, "private/other.yaml"), plain, 0600))
				}
			}
			if tc.command == "diff" || tc.command == "clean" || tc.command == "c" {
				if tc.scenario != "missing-plaintext" && tc.scenario != "all-skipped" {
					local := plain
					if tc.different {
						local = []byte("token: changed\n")
					}
					require.NoError(t, os.WriteFile(filepath.Join(dir, "good.yaml"), local, 0600))
				}
			}
			if tc.scenario == "all-skipped" {
				pairs = pairs[1:]
			}
			if tc.scenario == "complete" || tc.scenario == "missing-plaintext" {
				pairs = pairs[:1]
			}
			if tc.scenario == "broken" {
				if tc.command == "diff" || tc.command == "clean" || tc.command == "c" {
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
			if tc.stdin != "" {
				cmd.Stdin = strings.NewReader(tc.stdin)
			}
			if tc.strictEnv != "" {
				cmd.Env = append(os.Environ(), "YEWSEAL_DECRYPT_STRICT="+tc.strictEnv)
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
			if tc.name == "decrypt-bad-env" || tc.name == "diff-bad-flag" {
				require.NotContains(t, stderr.String(), "Summary")
				return
			}
			if tc.command == "clean" || tc.command == "c" {
				require.Empty(t, stdout.String(), "clean keeps stdout empty")
				for _, want := range tc.wantIn {
					require.Contains(t, stderr.String(), want)
				}
				if tc.wantOut != "" {
					require.NotContains(t, stderr.String(), tc.wantOut)
				}
				for _, name := range tc.gone {
					require.NoFileExists(t, filepath.Join(dir, name))
				}
				for _, name := range tc.kept {
					require.FileExists(t, filepath.Join(dir, name))
				}
				require.NoFileExists(t, filepath.Join(dir, ".gitignore"), "clean never touches project metadata")
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
				require.NotContains(t, stderr.String(), "invalid YEWSEAL_DECRYPT_STRICT")
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

func TestCLIExitCodes(t *testing.T) {
	binary := buildYews(t)
	clearCommandEnvironment(t)
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0755))

	for _, tc := range []struct {
		name string
		args []string
		code int
	}{
		{name: "unknown-command", args: []string{"bogus"}, code: 2},
		{name: "unknown-flag", args: []string{"encrypt", "--nope"}, code: 2},
		{name: "missing-config", args: []string{"encrypt"}, code: 2},
		{name: "wrong-arg-count", args: []string{"view"}, code: 2},
		{name: "unregistered-target", args: []string{"view", "nowhere.enc.yaml"}, code: 2},
		{name: "invalid-pattern", args: []string{"encrypt", "["}, code: 2},
		{name: "clean-strategy-conflict", args: []string{"clean", "--force", "--skip-different"}, code: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(binary, tc.args...)
			cmd.Dir = dir
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			var exit *exec.ExitError
			require.ErrorAs(t, cmd.Run(), &exit, "%s\n%s", &stdout, &stderr)
			require.Equal(t, tc.code, exit.ExitCode(), "%s\n%s", &stdout, &stderr)
			require.Empty(t, stdout.String(), "calling errors must not write to stdout")
		})
	}
}

func TestCLIDecryptAliasUsesInlineIdentityEnvironment(t *testing.T) {
	binary := buildYews(t)
	clearCommandEnvironment(t)
	for _, envName := range []string{"YEWSEAL_AGE_IDENTITIES", "SOPS_AGE_KEY"} {
		t.Run(envName, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0o755))
			owner, err := age.GenerateX25519Identity()
			require.NoError(t, err)
			plain := []byte("token: value\n")
			ciphertext, err := sopsx.Encrypt(plain, "yaml", []string{owner.Recipient().String()})
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "secret.enc.yaml"), ciphertext, 0o600))
			defaults := []string{"owner"}
			cfg := config.Config{
				Encryption: config.EncryptionConfig{Files: []config.FilePair{{PlaintextPath: "secret.yaml", EncryptedPath: "secret.enc.yaml", Format: "yaml"}}},
				Recipients: config.RecipientConfig{Defaults: &defaults, Registry: map[string]string{"owner": owner.Recipient().String()}},
			}
			data, err := toml.Marshal(cfg)
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(filepath.Join(dir, ".yewseal.toml"), data, 0o600))

			cmd := exec.Command(binary, "d")
			cmd.Dir = dir
			cmd.Env = append(os.Environ(), envName+"="+owner.String())
			output, err := cmd.CombinedOutput()
			require.NoError(t, err, "%s", output)
			decrypted, err := os.ReadFile(filepath.Join(dir, "secret.yaml"))
			require.NoError(t, err)
			require.Equal(t, plain, decrypted)
		})
	}
}
