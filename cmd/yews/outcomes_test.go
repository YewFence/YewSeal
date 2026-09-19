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
	// 场景输入与期望全部声明在表行：scenario/flags/stdin/code/different 驱动
	// fixture 与调用，stdout/stderr/文件三族字段驱动统一断言。
	const plain = "token: value\n"
	const changed = "token: changed\n"
	noMetadata := []string{".gitignore", ".sops.yaml"}
	for _, tc := range []struct {
		name, command, scenario, strictEnv string
		flags                              []string
		stdin                              string
		stdout, stdoutHas                  string
		wantIn, gone, kept                 []string
		wantOut                            []string
		wantFiles                          map[string]string
		wantIgnoreIn                       string
		code                               int
		different                          bool
	}{
		{name: "decrypt-complete-strict", command: "decrypt", scenario: "complete", flags: []string{"--strict"},
			wantIgnoreIn: "good.yaml", wantFiles: map[string]string{"good.yaml": plain}, gone: []string{"private"}},
		{name: "decrypt-partial", command: "decrypt",
			wantIgnoreIn: "private/other.yaml", wantFiles: map[string]string{"good.yaml": plain}, gone: []string{"private"}},
		{name: "decrypt-strict", command: "decrypt", flags: []string{"--strict"}, code: 1,
			wantIgnoreIn: "private/other.yaml", wantFiles: map[string]string{"good.yaml": plain}, gone: []string{"private"}},
		{name: "decrypt-env-strict", command: "decrypt", strictEnv: "true", code: 1,
			wantIgnoreIn: "private/other.yaml", wantFiles: map[string]string{"good.yaml": plain}, gone: []string{"private"}},
		{name: "decrypt-env-override", command: "decrypt", strictEnv: "true", flags: []string{"--strict=false"},
			wantIgnoreIn: "private/other.yaml", wantFiles: map[string]string{"good.yaml": plain}, gone: []string{"private"}},
		{name: "decrypt-all-skipped", command: "decrypt", scenario: "all-skipped",
			wantIgnoreIn: "private/other.yaml", gone: []string{"good.yaml", "private"}},
		{name: "decrypt-error-continues", command: "decrypt", scenario: "broken", code: 1,
			wantIgnoreIn: "private/other.yaml", wantFiles: map[string]string{"good.yaml": plain}, gone: []string{"private"}},
		{name: "diff-partial-same", command: "diff",
			wantIn: []string{"Comparison incomplete", "other.enc.yaml", "1 skipped"}, gone: noMetadata},
		{name: "diff-complete-same", command: "diff", scenario: "complete",
			wantIn: []string{"0 skipped"}, wantOut: []string{"Comparison incomplete"}, gone: noMetadata},
		{name: "diff-complete-different", command: "diff", scenario: "complete", different: true, stdoutHas: "--- ",
			wantIn: []string{"0 skipped"}, wantOut: []string{"Comparison incomplete"}, gone: noMetadata},
		{name: "diff-partial-different", command: "diff", different: true, stdoutHas: "--- ",
			wantIn: []string{"Comparison incomplete", "other.enc.yaml", "1 skipped"}, gone: noMetadata},
		{name: "diff-ignores-strict-env", command: "diff", strictEnv: "true",
			wantIn: []string{"Comparison incomplete", "other.enc.yaml", "1 skipped"}, gone: noMetadata},
		{name: "diff-ignores-invalid-strict-env", command: "diff", strictEnv: "bad",
			wantIn: []string{"Comparison incomplete", "other.enc.yaml", "1 skipped"}, gone: noMetadata},
		{name: "diff-all-skipped", command: "diff", scenario: "all-skipped",
			wantIn: []string{"Comparison incomplete", "other.enc.yaml", "1 skipped"}, gone: noMetadata},
		{name: "diff-errors-dominate-differences", command: "diff", scenario: "broken", different: true, code: 1, stdoutHas: "--- ",
			wantIn: []string{"Comparison incomplete", "other.enc.yaml", "1 skipped"}, gone: noMetadata},
		{name: "diff-missing-plaintext", command: "diff", scenario: "missing-plaintext",
			wantIn: []string{"1 missing input, 0 no matching identity"}, wantOut: []string{"Comparison incomplete"}, gone: noMetadata},
		{name: "diff-plaintext-discovery", command: "diff", scenario: "group-plaintext",
			wantIn:  []string{"1 compared, 1 skipped (1 missing input, 0 no matching identity), 0 failed (2 selected)"},
			wantOut: []string{"Comparison incomplete", "other.enc.yaml"}, gone: noMetadata},
		{name: "diff-verbose", command: "diff", flags: []string{"--verbose"},
			wantIn: []string{"Comparison incomplete", "other.enc.yaml", "1 skipped"}, gone: noMetadata},
		{name: "decrypt-bad-env", command: "decrypt", strictEnv: "bad", code: 2, wantOut: []string{"Summary"}},
		{name: "diff-bad-flag", command: "diff", flags: []string{"--unknown-option"}, code: 2, wantOut: []string{"Summary"}},
		{name: "view-does-not-skip", command: "view", flags: []string{"other.enc.yaml"}, strictEnv: "bad", code: 1,
			wantIn: []string{"no matching age identity"}, wantOut: []string{"invalid YEWSEAL_DECRYPT_STRICT"}},
		{name: "view-verbose-clean-stdout", command: "view", flags: []string{"good.enc.yaml", "--verbose"},
			stdout: plain, wantIn: []string{"Selected 1"}, gone: []string{"good.yaml", ".gitignore"}},
		{name: "edit-does-not-skip", command: "edit", flags: []string{"--file", "other.enc.yaml"}, strictEnv: "bad", code: 1,
			wantIn: []string{"no matching age identity"}, wantOut: []string{"invalid YEWSEAL_DECRYPT_STRICT"}},
		{name: "clean-removes-matching", command: "clean", scenario: "complete",
			wantIn: []string{"REMOVED good.yaml", "1 removed, 0 already absent, 0 retained, 0 failed (1 selected)"},
			gone:   []string{"good.yaml", ".gitignore", ".sops.yaml"}, kept: []string{"good.enc.yaml"}},
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
			wantIn: []string{"RETAINED good.yaml"}, wantOut: []string{"Delete the local plaintext anyway?"}, kept: []string{"good.yaml"}},
		{name: "clean-force-removes-difference", command: "clean", scenario: "complete", different: true, flags: []string{"--force"},
			wantIn: []string{"REMOVED good.yaml"}, wantOut: []string{"Delete the local plaintext anyway?"}, gone: []string{"good.yaml"}},
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
			for name, recipient := range map[string]string{"good.enc.yaml": owner.Recipient().String(), "other.enc.yaml": other.Recipient().String()} {
				data, err := sopsx.Encrypt([]byte(plain), "yaml", []string{recipient})
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(filepath.Join(dir, name), data, 0600))
			}
			pairs := []config.FilePair{
				{PlaintextPath: "good.yaml", EncryptedPath: "good.enc.yaml", Format: "yaml"},
				{PlaintextPath: "private/other.yaml", EncryptedPath: "other.enc.yaml", Format: "yaml"},
			}
			if tc.command == "diff" && tc.scenario != "group-plaintext" && tc.scenario != "missing-plaintext" {
				require.NoError(t, os.Mkdir(filepath.Join(dir, "private"), 0700))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "private/other.yaml"), []byte(plain), 0600))
			}
			if tc.command == "clean" || tc.command == "c" {
				if tc.scenario == "" {
					require.NoError(t, os.Mkdir(filepath.Join(dir, "private"), 0700))
					require.NoError(t, os.WriteFile(filepath.Join(dir, "private/other.yaml"), []byte(plain), 0600))
				}
			}
			if tc.command == "diff" || tc.command == "clean" || tc.command == "c" {
				if tc.scenario != "missing-plaintext" && tc.scenario != "all-skipped" {
					local := plain
					if tc.different {
						local = changed
					}
					require.NoError(t, os.WriteFile(filepath.Join(dir, "good.yaml"), []byte(local), 0600))
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
					require.NoError(t, os.WriteFile(filepath.Join(dir, "broken.yaml"), []byte(plain), 0600))
				}
				require.NoError(t, os.WriteFile(filepath.Join(dir, "broken.enc.yaml"), []byte("broken"), 0600))
				pairs = append([]config.FilePair{{PlaintextPath: "broken.yaml", EncryptedPath: "broken.enc.yaml", Format: "yaml"}}, pairs...)
			}
			defaults := []string{"owner"}
			cfg := config.Config{Encryption: config.EncryptionConfig{Files: pairs}, Recipients: config.RecipientConfig{Defaults: &defaults, Registry: map[string]string{"owner": owner.Recipient().String()}}}
			if tc.scenario == "group-plaintext" {
				cfg.Encryption.Files = nil
				cfg.Encryption.Groups = []config.GroupConfig{{Patterns: []string{"*.yaml"}}}
				require.NoError(t, os.WriteFile(filepath.Join(dir, "new.yaml"), []byte(plain), 0600))
			}
			data, err := toml.Marshal(cfg)
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(filepath.Join(dir, ".yewseal.toml"), data, 0600))
			ctx, cancel := context.WithTimeout(t.Context(), subprocessTimeout)
			defer cancel()
			cmd := exec.CommandContext(ctx, binary, append([]string{tc.command}, tc.flags...)...)
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
			// 期望与输入同屏：stdout/stderr/文件三族正交断言，全部来自表行。
			if tc.stdoutHas != "" {
				require.Contains(t, stdout.String(), tc.stdoutHas)
			} else {
				require.Equal(t, tc.stdout, stdout.String())
			}
			require.NotContains(t, stdout.String(), "Summary", "summaries belong on stderr")
			for _, want := range tc.wantIn {
				require.Contains(t, stderr.String(), want)
			}
			for _, want := range tc.wantOut {
				require.NotContains(t, stderr.String(), want)
			}
			for _, name := range tc.gone {
				require.NoFileExists(t, filepath.Join(dir, name))
			}
			for _, name := range tc.kept {
				require.FileExists(t, filepath.Join(dir, name))
			}
			for name, want := range tc.wantFiles {
				written, err := os.ReadFile(filepath.Join(dir, name))
				require.NoError(t, err)
				require.Equal(t, want, string(written))
			}
			if tc.wantIgnoreIn != "" {
				ignore, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
				require.NoError(t, err)
				require.Contains(t, string(ignore), tc.wantIgnoreIn)
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
