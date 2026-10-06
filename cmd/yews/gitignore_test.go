package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCLIUpdateGitignore(t *testing.T) {
	binary := buildYews(t)
	clearCommandEnvironment(t)
	for _, command := range []string{"init", "encrypt", "decrypt"} {
		for _, existing := range []bool{false, true} {
			for _, tc := range []struct {
				name    string
				env     string
				flags   []string
				update  bool
				sopsOff bool
				code    int
			}{
				{name: "default", update: true},
				{name: "flag-off", flags: []string{"--update-gitignore=false"}},
				{name: "env-off", env: "false"},
				{name: "flag-on-over-env-off", env: "false", flags: []string{"--update-gitignore=true"}, update: true},
				{name: "flag-off-over-env-on", env: "true", flags: []string{"--update-gitignore=false"}},
				{name: "invalid-env", env: "invalid", code: 2},
				{name: "flag-over-invalid-env", env: "invalid", flags: []string{"--update-gitignore=false"}},
				{name: "ignore-on-sops-off", flags: []string{"--sync-sops-config=false"}, update: true, sopsOff: true},
			} {
				if command == "decrypt" && tc.sopsOff {
					continue
				}
				t.Run(command+"/"+tc.name+"/existing="+strconv.FormatBool(existing), func(t *testing.T) {
					dir := t.TempDir()
					plain := []byte("token: fixture\n")
					require.NoError(t, os.WriteFile(filepath.Join(dir, "config.yaml"), plain, 0600))
					if command != "init" {
						_, initOutput, initCode := runMetadataCommand(t, binary, dir, nil, "init", "--input", "config.yaml")
						require.Zero(t, initCode, initOutput)
						_, encryptOutput, encryptCode := runMetadataCommand(t, binary, dir, nil, "encrypt")
						require.Zero(t, encryptCode, encryptOutput)
						require.NoError(t, os.Remove(filepath.Join(dir, ".gitignore")))
						if command == "encrypt" {
							require.NoError(t, os.WriteFile(filepath.Join(dir, ".sops.yaml"), []byte("# stale SOPS rules\n"), 0600))
						}
						if command == "decrypt" {
							require.NoError(t, os.Remove(filepath.Join(dir, "config.yaml")))
						}
					}
					ignorePath := filepath.Join(dir, ".gitignore")
					custom := []byte("# Custom ignore rules\nbuild/\n")
					if existing {
						require.NoError(t, os.WriteFile(ignorePath, custom, 0600))
					}
					args := []string{command}
					if command == "init" {
						args = append(args, "--input", "config.yaml")
					}
					args = append(args, tc.flags...)
					env := []string{
						"YEWSEAL_UPDATE_GITIGNORE=" + tc.env,
						"YEWSEAL_" + strings.ToUpper(command) + "_UPDATE_GITIGNORE=invalid",
					}
					stdout, stderr, code := runMetadataCommand(t, binary, dir, env, args...)
					require.Equal(t, tc.code, code, "%s\n%s", stdout, stderr)
					require.Empty(t, stdout)
					if tc.code != 0 {
						require.Contains(t, stderr, "invalid YEWSEAL_UPDATE_GITIGNORE")
					} else {
						if command == "init" && tc.sopsOff {
							require.NoFileExists(t, filepath.Join(dir, ".sops.yaml"))
						} else {
							require.FileExists(t, filepath.Join(dir, ".sops.yaml"))
						}
						if command == "encrypt" {
							policy, err := os.ReadFile(filepath.Join(dir, ".sops.yaml"))
							require.NoError(t, err)
							if tc.sopsOff {
								require.Equal(t, "# stale SOPS rules\n", string(policy))
							} else {
								require.Contains(t, string(policy), `path_regex: ^config\.enc\.yaml$`)
								require.NotContains(t, string(policy), "stale SOPS rules")
							}
						}
						if command != "init" {
							require.FileExists(t, filepath.Join(dir, "config.enc.yaml"))
							if command == "decrypt" {
								decoded, err := os.ReadFile(filepath.Join(dir, "config.yaml"))
								require.NoError(t, err)
								require.Equal(t, plain, decoded)
							}
						}
					}
					if tc.update {
						data, err := os.ReadFile(ignorePath)
						require.NoError(t, err)
						require.Contains(t, string(data), "\nconfig.yaml\n")
						require.Contains(t, string(data), "\n.age/keys.txt\n")
						if existing {
							require.True(t, bytes.HasPrefix(data, custom))
						}
					} else if existing {
						data, err := os.ReadFile(ignorePath)
						require.NoError(t, err)
						require.Equal(t, custom, data)
					} else {
						require.NoFileExists(t, ignorePath)
					}
				})
			}
		}
	}
}

func TestCLIUpdateGitignoreDisabledDoesNotReadIgnore(t *testing.T) {
	binary := buildYews(t)
	clearCommandEnvironment(t)
	for _, command := range []string{"init", "encrypt", "decrypt"} {
		t.Run(command, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("token: fixture\n"), 0600))
			if command != "init" {
				_, initOutput, initCode := runMetadataCommand(t, binary, dir, nil, "init", "--input", "config.yaml")
				require.Zero(t, initCode, initOutput)
				_, encryptOutput, encryptCode := runMetadataCommand(t, binary, dir, nil, "encrypt")
				require.Zero(t, encryptCode, encryptOutput)
				require.NoError(t, os.Remove(filepath.Join(dir, ".gitignore")))
			}
			require.NoError(t, os.Mkdir(filepath.Join(dir, ".gitignore"), 0700))
			args := []string{command, "--update-gitignore=false"}
			if command == "init" {
				args = append(args, "--input", "config.yaml", "--force")
			}
			_, stderr, code := runMetadataCommand(t, binary, dir, nil, args...)
			require.Zero(t, code, stderr)
			require.DirExists(t, filepath.Join(dir, ".gitignore"))
		})
	}
}

func TestVerifyReportsExposureWithGitignoreUpdatesDisabled(t *testing.T) {
	binary := buildYews(t)
	clearCommandEnvironment(t)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	dir := t.TempDir()
	output, err := exec.Command("git", "init", "-q", dir).CombinedOutput()
	require.NoError(t, err, "%s", output)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("token: [密钥]"), 0600))
	env := []string{"YEWSEAL_UPDATE_GITIGNORE=false"}
	for _, args := range [][]string{{"init", "--input", "config.yaml"}, {"encrypt"}} {
		_, stderr, code := runMetadataCommand(t, binary, dir, env, args...)
		require.Zero(t, code, stderr)
	}
	stdout, stderr, code := runMetadataCommand(t, binary, dir, env, "verify", "--no-decrypt", "--json")
	require.Zero(t, code, stderr)
	var report struct {
		Findings []struct {
			Code string `json:"code"`
		} `json:"findings"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &report))
	codes := make([]string, 0, len(report.Findings))
	for _, finding := range report.Findings {
		codes = append(codes, finding.Code)
	}
	require.Contains(t, codes, "plaintext_not_ignored")
	require.Contains(t, codes, "key_not_ignored")
	require.NoFileExists(t, filepath.Join(dir, ".gitignore"))
}

func runMetadataCommand(t *testing.T, binary, dir string, env []string, args ...string) (string, string, int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), subprocessTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	if err == nil {
		return stdout.String(), stderr.String(), 0
	}
	var exit *exec.ExitError
	require.ErrorAs(t, err, &exit, "%s\n%s", &stdout, &stderr)
	return stdout.String(), stderr.String(), exit.ExitCode()
}
