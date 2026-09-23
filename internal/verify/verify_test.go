package verify_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"filippo.io/age"
	"github.com/YewFence/YewSeal/internal/agekey"
	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/sopsconfig"
	"github.com/YewFence/YewSeal/internal/sopsx"
	"github.com/YewFence/YewSeal/internal/verify"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── helpers ────────────────────────────────────────────────────────────────

type testKey struct {
	identity  string
	recipient string
	bundle    agekey.IdentityBundle
}

func newTestKey(t *testing.T) testKey {
	t.Helper()
	id, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	secret := id.String()
	bundle, err := agekey.NewIdentityBundle([]string{secret})
	require.NoError(t, err)
	return testKey{identity: secret, recipient: id.Recipient().String(), bundle: bundle}
}

func makeEncrypted(t *testing.T, dir, name, format string, plain []byte, recipients []string) string {
	t.Helper()
	encData, err := sopsx.Encrypt(plain, format, recipients)
	require.NoError(t, err)
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, encData, 0644))
	return path
}

func makePlaintext(t *testing.T, dir, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, data, 0644))
	return path
}

func writeKeyFile(t *testing.T, dir string, key testKey) string {
	t.Helper()
	kf := filepath.Join(dir, "keys.txt")
	content := fmt.Sprintf("# created: %s\n# public key: %s\n%s\n",
		time.Now().UTC().Format(time.RFC3339), key.recipient, key.identity)
	require.NoError(t, os.WriteFile(kf, []byte(content), 0600))
	return kf
}

func resolvedPair(plain, enc, format string, recipients []string) config.ResolvedFilePair {
	return config.ResolvedFilePair{
		PlaintextPath: plain,
		EncryptedPath: enc,
		Format:        format,
		Recipients:    recipients,
	}
}

func minimalSelection(pairs ...config.ResolvedFilePair) config.ResolvedSelection {
	all := append([]config.ResolvedFilePair(nil), pairs...)
	return config.ResolvedSelection{FilePairs: pairs, AllConfigPairs: all}
}

func requireFinding(t *testing.T, report *verify.Report, code string, sev verify.Severity) {
	t.Helper()
	for _, f := range report.Findings {
		if f.Code == code && f.Severity == sev {
			return
		}
	}
	var got []string
	for _, f := range report.Findings {
		got = append(got, fmt.Sprintf("%s(%s)", f.Code, f.Severity))
	}
	t.Errorf("expected finding %s(%s); got: %v", code, sev, got)
}

func requireNoFinding(t *testing.T, report *verify.Report, code string) {
	t.Helper()
	for _, f := range report.Findings {
		if f.Code == code {
			t.Errorf("unexpected finding %s: %s", code, f.Message)
		}
	}
}

// isolateVCSConfig keeps the developer's global git/jj configuration (commit
// signing, hooks, identity) out of the fixtures, and applies to the git and jj
// processes spawned by verify.Check as well.
func isolateVCSConfig(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	gitConfig := filepath.Join(home, "gitconfig")
	require.NoError(t, os.WriteFile(gitConfig, []byte("[user]\n\tname = Test\n\temail = test@example.com\n[commit]\n\tgpgsign = false\n[init]\n\tdefaultBranch = main\n"), 0600))
	jjConfig := filepath.Join(home, "jjconfig.toml")
	require.NoError(t, os.WriteFile(jjConfig, []byte("[user]\nname = \"Test\"\nemail = \"test@example.com\"\n"), 0600))
	t.Setenv("GIT_CONFIG_GLOBAL", gitConfig)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("JJ_CONFIG", jjConfig)
	t.Setenv("HOME", home)
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v failed:\n%s", args, out)
}

func runJJ(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("jj", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "jj %v failed:\n%s", args, out)
}

func requireJJ(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("jj"); err == nil {
		return
	}
	if os.Getenv("CI") != "" {
		t.Fatal("jj is required in CI; it is declared in mise.toml")
	}
	t.Skip("jj is not installed")
}

// ── ciphertext static layer ────────────────────────────────────────────────

func TestCiphertextMissing(t *testing.T) {
	dir := t.TempDir()
	sel := minimalSelection(resolvedPair(
		filepath.Join(dir, "config.yaml"),
		filepath.Join(dir, "config.enc.yaml"),
		"yaml", nil,
	))
	report, err := verify.Check(sel, dir, verify.Options{CheckSOPSConfig: false})
	require.NoError(t, err)
	requireFinding(t, report, "ciphertext_missing", verify.SeverityError)
}

func TestCiphertextNotRegularFile(t *testing.T) {
	dir := t.TempDir()
	encPath := filepath.Join(dir, "config.enc.yaml")
	require.NoError(t, os.MkdirAll(encPath, 0755))
	sel := minimalSelection(resolvedPair(
		filepath.Join(dir, "config.yaml"), encPath, "yaml", nil,
	))
	report, err := verify.Check(sel, dir, verify.Options{CheckSOPSConfig: false})
	require.NoError(t, err)
	requireFinding(t, report, "ciphertext_not_regular", verify.SeverityError)
}

func TestCiphertextNotParseable(t *testing.T) {
	dir := t.TempDir()
	encPath := filepath.Join(dir, "config.enc.yaml")
	require.NoError(t, os.WriteFile(encPath, []byte("not sops ciphertext\n"), 0644))
	sel := minimalSelection(resolvedPair(
		filepath.Join(dir, "config.yaml"), encPath, "yaml", nil,
	))
	report, err := verify.Check(sel, dir, verify.Options{CheckSOPSConfig: false})
	require.NoError(t, err)
	requireFinding(t, report, "ciphertext_parse_error", verify.SeverityError)
}

func TestCiphertextRecipientMissing(t *testing.T) {
	dir := t.TempDir()
	key1 := newTestKey(t)
	key2 := newTestKey(t)
	encPath := makeEncrypted(t, dir, "config.enc.yaml", "yaml",
		[]byte("token: secret\n"), []string{key1.recipient})
	sel := minimalSelection(resolvedPair(
		filepath.Join(dir, "config.yaml"), encPath, "yaml",
		[]string{key1.recipient, key2.recipient},
	))
	report, err := verify.Check(sel, dir, verify.Options{CheckSOPSConfig: false})
	require.NoError(t, err)
	requireFinding(t, report, "recipient_missing", verify.SeverityError)
}

func TestCiphertextRecipientExtra(t *testing.T) {
	dir := t.TempDir()
	key1 := newTestKey(t)
	key2 := newTestKey(t)
	encPath := makeEncrypted(t, dir, "config.enc.yaml", "yaml",
		[]byte("token: secret\n"), []string{key1.recipient, key2.recipient})
	sel := minimalSelection(resolvedPair(
		filepath.Join(dir, "config.yaml"), encPath, "yaml",
		[]string{key1.recipient},
	))
	report, err := verify.Check(sel, dir, verify.Options{CheckSOPSConfig: false})
	require.NoError(t, err)
	requireFinding(t, report, "recipient_extra", verify.SeverityError)
}

func TestCiphertextRecipientMatchPass(t *testing.T) {
	dir := t.TempDir()
	key := newTestKey(t)
	encPath := makeEncrypted(t, dir, "config.enc.yaml", "yaml",
		[]byte("token: secret\n"), []string{key.recipient})
	sel := minimalSelection(resolvedPair(
		filepath.Join(dir, "config.yaml"), encPath, "yaml",
		[]string{key.recipient},
	))
	report, err := verify.Check(sel, dir, verify.Options{CheckSOPSConfig: false})
	require.NoError(t, err)
	requireNoFinding(t, report, "recipient_missing")
	requireNoFinding(t, report, "recipient_extra")
}

// ── decrypt layer ──────────────────────────────────────────────────────────

func TestDecryptAutoSkipsWhenNoIdentity(t *testing.T) {
	dir := t.TempDir()
	key := newTestKey(t)
	encPath := makeEncrypted(t, dir, "config.enc.yaml", "yaml",
		[]byte("token: secret\n"), []string{key.recipient})
	sel := minimalSelection(resolvedPair(
		filepath.Join(dir, "config.yaml"), encPath, "yaml", []string{key.recipient},
	))
	report, err := verify.Check(sel, dir, verify.Options{
		DecryptMode:     verify.DecryptAuto,
		CheckSOPSConfig: false,
	})
	require.NoError(t, err)
	assert.Greater(t, report.SkipCount, 0, "should skip decrypt layer when no identity")
	requireNoFinding(t, report, "decrypt_failed")
}

func TestDecryptRequiredErrorsWhenNoIdentity(t *testing.T) {
	dir := t.TempDir()
	key := newTestKey(t)
	encPath := makeEncrypted(t, dir, "config.enc.yaml", "yaml",
		[]byte("token: secret\n"), []string{key.recipient})
	sel := minimalSelection(resolvedPair(
		filepath.Join(dir, "config.yaml"), encPath, "yaml", []string{key.recipient},
	))
	_, err := verify.Check(sel, dir, verify.Options{
		DecryptMode:     verify.DecryptRequired,
		CheckSOPSConfig: false,
	})
	require.Error(t, err)
}

func TestDecryptLayerPassesWithMatchingIdentity(t *testing.T) {
	dir := t.TempDir()
	key := newTestKey(t)
	plain := []byte("token: secret\n")
	encPath := makeEncrypted(t, dir, "config.enc.yaml", "yaml", plain, []string{key.recipient})
	kf := writeKeyFile(t, dir, key)
	sel := minimalSelection(resolvedPair(
		filepath.Join(dir, "config.yaml"), encPath, "yaml", []string{key.recipient},
	))
	report, err := verify.Check(sel, dir, verify.Options{
		KeyFile:         kf,
		DecryptMode:     verify.DecryptAuto,
		CheckSOPSConfig: false,
	})
	require.NoError(t, err)
	requireNoFinding(t, report, "decrypt_failed")
	requireNoFinding(t, report, "decrypt_no_matching_identity")
	requireNoFinding(t, report, "mac_mismatch")
}

func TestDecryptNoMatchingIdentityIsWarningInAutoMode(t *testing.T) {
	dir := t.TempDir()
	key1 := newTestKey(t)
	key2 := newTestKey(t)
	encPath := makeEncrypted(t, dir, "config.enc.yaml", "yaml",
		[]byte("token: secret\n"), []string{key1.recipient})
	kf := writeKeyFile(t, dir, key2) // wrong key
	sel := minimalSelection(resolvedPair(
		filepath.Join(dir, "config.yaml"), encPath, "yaml", []string{key1.recipient},
	))
	report, err := verify.Check(sel, dir, verify.Options{
		KeyFile:         kf,
		DecryptMode:     verify.DecryptAuto,
		CheckSOPSConfig: false,
	})
	require.NoError(t, err)
	requireFinding(t, report, "decrypt_no_matching_identity", verify.SeverityWarning)
}

func TestDecryptNoMatchingIdentityIsErrorInRequiredMode(t *testing.T) {
	dir := t.TempDir()
	key1 := newTestKey(t)
	key2 := newTestKey(t)
	encPath := makeEncrypted(t, dir, "config.enc.yaml", "yaml",
		[]byte("token: secret\n"), []string{key1.recipient})
	kf := writeKeyFile(t, dir, key2) // wrong key
	sel := minimalSelection(resolvedPair(
		filepath.Join(dir, "config.yaml"), encPath, "yaml", []string{key1.recipient},
	))
	report, err := verify.Check(sel, dir, verify.Options{
		KeyFile:         kf,
		DecryptMode:     verify.DecryptRequired,
		CheckSOPSConfig: false,
	})
	require.NoError(t, err)
	requireFinding(t, report, "decrypt_no_matching_identity", verify.SeverityError)
}

func TestDecryptLayerSkippedWithNoDecrypt(t *testing.T) {
	dir := t.TempDir()
	key := newTestKey(t)
	kf := writeKeyFile(t, dir, key)
	encPath := makeEncrypted(t, dir, "config.enc.yaml", "yaml",
		[]byte("token: secret\n"), []string{key.recipient})
	sel := minimalSelection(resolvedPair(
		filepath.Join(dir, "config.yaml"), encPath, "yaml", []string{key.recipient},
	))
	report, err := verify.Check(sel, dir, verify.Options{
		KeyFile:         kf,
		DecryptMode:     verify.DecryptDisabled,
		CheckSOPSConfig: false,
	})
	require.NoError(t, err)
	requireNoFinding(t, report, "decrypt_failed")
	assert.Greater(t, report.SkipCount, 0)
}

func TestPlaintextDriftDetected(t *testing.T) {
	dir := t.TempDir()
	key := newTestKey(t)
	encPath := makeEncrypted(t, dir, "config.enc.yaml", "yaml",
		[]byte("token: original\n"), []string{key.recipient})
	plainPath := makePlaintext(t, dir, "config.yaml", []byte("token: modified\n"))
	kf := writeKeyFile(t, dir, key)
	sel := minimalSelection(resolvedPair(plainPath, encPath, "yaml", []string{key.recipient}))
	report, err := verify.Check(sel, dir, verify.Options{
		KeyFile:         kf,
		DecryptMode:     verify.DecryptAuto,
		CheckSOPSConfig: false,
	})
	require.NoError(t, err)
	requireFinding(t, report, "plaintext_drift", verify.SeverityError)
}

func TestPlaintextAbsentDecryptPass(t *testing.T) {
	dir := t.TempDir()
	key := newTestKey(t)
	encPath := makeEncrypted(t, dir, "config.enc.yaml", "yaml",
		[]byte("token: secret\n"), []string{key.recipient})
	kf := writeKeyFile(t, dir, key)
	sel := minimalSelection(resolvedPair(
		filepath.Join(dir, "config.yaml"), encPath, "yaml", []string{key.recipient},
	))
	report, err := verify.Check(sel, dir, verify.Options{
		KeyFile:         kf,
		DecryptMode:     verify.DecryptAuto,
		CheckSOPSConfig: false,
	})
	require.NoError(t, err)
	requireNoFinding(t, report, "plaintext_drift")
	requireNoFinding(t, report, "decrypt_failed")
}

func TestMACMismatchIsSeparateFromDecryptFailure(t *testing.T) {
	dir := t.TempDir()
	key := newTestKey(t)
	encData, err := sopsx.Encrypt([]byte("a: one\nb: two\n"), "yaml", []string{key.recipient})
	require.NoError(t, err)
	var kept []string
	for _, line := range strings.Split(string(encData), "\n") {
		if !strings.HasPrefix(line, "b: ") {
			kept = append(kept, line)
		}
	}
	// Removing a whole value keeps every remaining value decryptable, so only the MAC can catch it.
	encPath := filepath.Join(dir, "config.enc.yaml")
	require.NoError(t, os.WriteFile(encPath, []byte(strings.Join(kept, "\n")), 0644))
	sel := minimalSelection(resolvedPair(filepath.Join(dir, "config.yaml"), encPath, "yaml", []string{key.recipient}))
	report, err := verify.Check(sel, dir, verify.Options{KeyFile: writeKeyFile(t, dir, key)})
	require.NoError(t, err)
	requireFinding(t, report, "mac_mismatch", verify.SeverityError)
	requireNoFinding(t, report, "decrypt_failed")
}

func TestCorruptedValueIsDecryptFailure(t *testing.T) {
	dir := t.TempDir()
	key := newTestKey(t)
	encData, err := sopsx.Encrypt([]byte("token: secret\n"), "yaml", []string{key.recipient})
	require.NoError(t, err)
	corrupted := strings.Replace(string(encData), "ENC[AES256_GCM,data:", "ENC[AES256_GCM,data:A", 1)
	encPath := filepath.Join(dir, "config.enc.yaml")
	require.NoError(t, os.WriteFile(encPath, []byte(corrupted), 0644))
	sel := minimalSelection(resolvedPair(filepath.Join(dir, "config.yaml"), encPath, "yaml", []string{key.recipient}))
	report, err := verify.Check(sel, dir, verify.Options{KeyFile: writeKeyFile(t, dir, key)})
	require.NoError(t, err)
	requireFinding(t, report, "decrypt_failed", verify.SeverityError)
	requireNoFinding(t, report, "mac_mismatch")
}

func TestUnusableCiphertextSkipsDecryption(t *testing.T) {
	dir := t.TempDir()
	key := newTestKey(t)
	sel := minimalSelection(resolvedPair(filepath.Join(dir, "config.yaml"), filepath.Join(dir, "config.enc.yaml"), "yaml", []string{key.recipient}))
	report, err := verify.Check(sel, dir, verify.Options{KeyFile: writeKeyFile(t, dir, key)})
	require.NoError(t, err)
	requireFinding(t, report, "ciphertext_missing", verify.SeverityError)
	requireNoFinding(t, report, "decrypt_failed")
	assert.Equal(t, 1, report.ErrorCount, "a missing ciphertext is reported once")
}

func TestRecipientDuplicateInMetadata(t *testing.T) {
	dir := t.TempDir()
	key := newTestKey(t)
	encPath := makeEncrypted(t, dir, "config.enc.yaml", "yaml", []byte("token: secret\n"), []string{key.recipient, key.recipient})
	sel := minimalSelection(resolvedPair(filepath.Join(dir, "config.yaml"), encPath, "yaml", []string{key.recipient}))
	report, err := verify.Check(sel, dir, verify.Options{})
	require.NoError(t, err)
	requireFinding(t, report, "recipient_duplicate", verify.SeverityError)
	requireNoFinding(t, report, "recipient_missing")
	requireNoFinding(t, report, "recipient_extra")
}

func TestRecipientMessagesUseAliasesAndCarryFullKey(t *testing.T) {
	dir := t.TempDir()
	configured, extra := newTestKey(t), newTestKey(t)
	encPath := makeEncrypted(t, dir, "config.enc.yaml", "yaml", []byte("token: secret\n"), []string{extra.recipient})
	sel := minimalSelection(resolvedPair(filepath.Join(dir, "config.yaml"), encPath, "yaml", []string{configured.recipient}))
	report, err := verify.Check(sel, dir, verify.Options{RecipientAliases: map[string]string{configured.recipient: "owner"}})
	require.NoError(t, err)
	for _, f := range report.Findings {
		switch f.Code {
		case "recipient_missing":
			assert.Contains(t, f.Message, "owner")
			assert.Equal(t, configured.recipient, f.Recipient)
		case "recipient_extra":
			assert.NotContains(t, f.Message, extra.recipient, "unknown keys are shortened in messages")
			assert.Equal(t, extra.recipient, f.Recipient)
		}
	}
}

// ── .sops.yaml drift ──────────────────────────────────────────────────────

func TestSOPSDriftPassesForSynchronizedFile(t *testing.T) {
	dir := t.TempDir()
	key := newTestKey(t)
	encPath := makeEncrypted(t, dir, "config.enc.yaml", "yaml", []byte("token: secret\n"), []string{key.recipient})
	pair := resolvedPair(filepath.Join(dir, "config.yaml"), encPath, "yaml", []string{key.recipient})
	rendered, err := sopsconfig.Render(config.DisplayResolvedFilePairs([]config.ResolvedFilePair{pair}, dir))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".sops.yaml"), rendered, 0644))
	report, err := verify.Check(minimalSelection(pair), dir, verify.Options{CheckSOPSConfig: true})
	require.NoError(t, err)
	requireNoFinding(t, report, "sops_config_drift")
}

func TestSOPSDriftDetectedAfterManualEdit(t *testing.T) {
	dir := t.TempDir()
	key := newTestKey(t)
	encPath := makeEncrypted(t, dir, "config.enc.yaml", "yaml",
		[]byte("token: secret\n"), []string{key.recipient})
	// Write a .sops.yaml with a wrong recipient
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".sops.yaml"),
		[]byte("creation_rules:\n  - path_regex: '.*'\n    age: age1wrongkey\n"), 0644))
	sel := config.ResolvedSelection{
		FilePairs: []config.ResolvedFilePair{
			resolvedPair(filepath.Join(dir, "config.yaml"), encPath, "yaml", []string{key.recipient}),
		},
		AllConfigPairs: []config.ResolvedFilePair{
			resolvedPair(filepath.Join(dir, "config.yaml"), encPath, "yaml", []string{key.recipient}),
		},
	}
	report, err := verify.Check(sel, dir, verify.Options{CheckSOPSConfig: true})
	require.NoError(t, err)
	requireFinding(t, report, "sops_config_drift", verify.SeverityError)
}

func TestSOPSDriftSkippedWhenFileAbsent(t *testing.T) {
	dir := t.TempDir()
	sel := config.ResolvedSelection{AllConfigPairs: nil}
	report, err := verify.Check(sel, dir, verify.Options{CheckSOPSConfig: true})
	require.NoError(t, err)
	requireNoFinding(t, report, "sops_config_drift")
	assert.Greater(t, report.SkipCount, 0)
}

func TestSOPSDriftSkippedWhenCheckDisabled(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".sops.yaml"),
		[]byte("creation_rules:\n  - path_regex: '.*'\n    age: age1wrongkey\n"), 0644))
	sel := config.ResolvedSelection{AllConfigPairs: nil}
	report, err := verify.Check(sel, dir, verify.Options{CheckSOPSConfig: false})
	require.NoError(t, err)
	requireNoFinding(t, report, "sops_config_drift")
}

// ── output: sort stability ─────────────────────────────────────────────────

func TestFindingsSortedDeterministically(t *testing.T) {
	dir := t.TempDir()
	key1 := newTestKey(t)
	key2 := newTestKey(t)
	// Both encrypted files are missing — deterministic sort by path
	pairs := []config.ResolvedFilePair{
		resolvedPair(filepath.Join(dir, "b.yaml"), filepath.Join(dir, "b.enc.yaml"), "yaml", []string{key2.recipient}),
		resolvedPair(filepath.Join(dir, "a.yaml"), filepath.Join(dir, "a.enc.yaml"), "yaml", []string{key1.recipient}),
	}
	sel := minimalSelection(pairs...)
	report1, err := verify.Check(sel, dir, verify.Options{CheckSOPSConfig: false})
	require.NoError(t, err)
	report2, err := verify.Check(sel, dir, verify.Options{CheckSOPSConfig: false})
	require.NoError(t, err)
	require.Equal(t, len(report1.Findings), len(report2.Findings))
	for i := range report1.Findings {
		assert.Equal(t, report1.Findings[i].Code, report2.Findings[i].Code)
		assert.Equal(t, report1.Findings[i].EncryptedPath, report2.Findings[i].EncryptedPath)
	}
	if len(report1.Findings) >= 2 {
		assert.True(t, report1.Findings[0].EncryptedPath < report1.Findings[1].EncryptedPath,
			"findings must be sorted by encrypted path")
	}
}

// ── VCS layer: no repo ─────────────────────────────────────────────────────

func TestVCSLayerSkippedOutsideRepo(t *testing.T) {
	dir := t.TempDir()
	key := newTestKey(t)
	encPath := makeEncrypted(t, dir, "config.enc.yaml", "yaml",
		[]byte("token: secret\n"), []string{key.recipient})
	sel := minimalSelection(resolvedPair(
		filepath.Join(dir, "config.yaml"), encPath, "yaml", []string{key.recipient},
	))
	report, err := verify.Check(sel, dir, verify.Options{CheckSOPSConfig: false})
	require.NoError(t, err)
	hasVCSSkip := false
	for _, reason := range report.SkipReasons {
		if strings.Contains(reason, "VCS") {
			hasVCSSkip = true
		}
	}
	assert.True(t, hasVCSSkip, "expected a VCS skip message outside a repo")
}

// ── VCS layer: git ─────────────────────────────────────────────────────────

func TestVCSLayerGitTrackedPlaintextIsError(t *testing.T) {
	dir := t.TempDir()
	isolateVCSConfig(t)
	runGit(t, dir, "init", "-q")
	key := newTestKey(t)
	plain := []byte("token: secret\n")
	plainPath := makePlaintext(t, dir, "config.yaml", plain)
	encPath := makeEncrypted(t, dir, "config.enc.yaml", "yaml", plain, []string{key.recipient})
	runGit(t, dir, "add", "config.yaml")
	runGit(t, dir, "commit", "-q", "-m", "add plaintext")
	sel := minimalSelection(resolvedPair(plainPath, encPath, "yaml", []string{key.recipient}))
	report, err := verify.Check(sel, dir, verify.Options{CheckSOPSConfig: false})
	require.NoError(t, err)
	requireFinding(t, report, "plaintext_tracked", verify.SeverityError)
}

func TestVCSLayerGitUntrackedNotIgnoredIsWarning(t *testing.T) {
	dir := t.TempDir()
	isolateVCSConfig(t)
	runGit(t, dir, "init", "-q")
	runGit(t, dir, "commit", "-q", "--allow-empty", "-m", "init")
	key := newTestKey(t)
	plain := []byte("token: secret\n")
	plainPath := makePlaintext(t, dir, "config.yaml", plain)
	encPath := makeEncrypted(t, dir, "config.enc.yaml", "yaml", plain, []string{key.recipient})
	sel := minimalSelection(resolvedPair(plainPath, encPath, "yaml", []string{key.recipient}))
	report, err := verify.Check(sel, dir, verify.Options{CheckSOPSConfig: false})
	require.NoError(t, err)
	requireFinding(t, report, "plaintext_not_ignored", verify.SeverityWarning)
}

func TestVCSLayerGitTrackedAfterIgnoreStillError(t *testing.T) {
	dir := t.TempDir()
	isolateVCSConfig(t)
	runGit(t, dir, "init", "-q")
	key := newTestKey(t)
	plain := []byte("token: secret\n")
	plainPath := makePlaintext(t, dir, "config.yaml", plain)
	encPath := makeEncrypted(t, dir, "config.enc.yaml", "yaml", plain, []string{key.recipient})
	runGit(t, dir, "add", "config.yaml")
	runGit(t, dir, "commit", "-q", "-m", "add plaintext")
	// Add to .gitignore after commit — file is still in the index
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("config.yaml\n"), 0644))
	sel := minimalSelection(resolvedPair(plainPath, encPath, "yaml", []string{key.recipient}))
	report, err := verify.Check(sel, dir, verify.Options{CheckSOPSConfig: false})
	require.NoError(t, err)
	requireFinding(t, report, "plaintext_tracked", verify.SeverityError)
}

func TestVCSLayerGitIgnoredIsPass(t *testing.T) {
	dir := t.TempDir()
	isolateVCSConfig(t)
	runGit(t, dir, "init", "-q")
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("config.yaml\n"), 0644))
	runGit(t, dir, "add", ".gitignore")
	runGit(t, dir, "commit", "-q", "-m", "add gitignore")
	key := newTestKey(t)
	plain := []byte("token: secret\n")
	plainPath := makePlaintext(t, dir, "config.yaml", plain)
	encPath := makeEncrypted(t, dir, "config.enc.yaml", "yaml", plain, []string{key.recipient})
	sel := minimalSelection(resolvedPair(plainPath, encPath, "yaml", []string{key.recipient}))
	report, err := verify.Check(sel, dir, verify.Options{CheckSOPSConfig: false})
	require.NoError(t, err)
	requireNoFinding(t, report, "plaintext_tracked")
	requireNoFinding(t, report, "plaintext_not_ignored")
}

// ── VCS layer: jj ─────────────────────────────────────────────────────────

func TestVCSLayerJJScratchIsWarning(t *testing.T) {
	requireJJ(t)
	dir := t.TempDir()
	isolateVCSConfig(t)
	runJJ(t, dir, "git", "init")
	key := newTestKey(t)
	plain := []byte("token: secret\n")
	plainPath := makePlaintext(t, dir, "config.yaml", plain)
	encPath := makeEncrypted(t, dir, "config.enc.yaml", "yaml", plain, []string{key.recipient})
	sel := minimalSelection(resolvedPair(plainPath, encPath, "yaml", []string{key.recipient}))
	report, err := verify.Check(sel, dir, verify.Options{CheckSOPSConfig: false})
	require.NoError(t, err)
	// In jj, a new file in @ (scratch, not yet committed) is a warning
	requireFinding(t, report, "plaintext_not_ignored", verify.SeverityWarning)
}

func TestVCSLayerJJCommittedIsError(t *testing.T) {
	requireJJ(t)
	dir := t.TempDir()
	isolateVCSConfig(t)
	runJJ(t, dir, "git", "init")
	key := newTestKey(t)
	plain := []byte("token: secret\n")
	plainPath := makePlaintext(t, dir, "config.yaml", plain)
	encPath := makeEncrypted(t, dir, "config.enc.yaml", "yaml", plain, []string{key.recipient})
	runJJ(t, dir, "commit", "-m", "add plaintext")
	sel := minimalSelection(resolvedPair(plainPath, encPath, "yaml", []string{key.recipient}))
	report, err := verify.Check(sel, dir, verify.Options{CheckSOPSConfig: false})
	require.NoError(t, err)
	requireFinding(t, report, "plaintext_tracked", verify.SeverityError)
}

func TestVCSLayerJJIgnoredIsPass(t *testing.T) {
	requireJJ(t)
	dir := t.TempDir()
	isolateVCSConfig(t)
	runJJ(t, dir, "git", "init")
	// jj respects .gitignore
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("config.yaml\n"), 0644))
	runJJ(t, dir, "commit", "-m", "add gitignore")
	key := newTestKey(t)
	plain := []byte("token: secret\n")
	plainPath := makePlaintext(t, dir, "config.yaml", plain)
	encPath := makeEncrypted(t, dir, "config.enc.yaml", "yaml", plain, []string{key.recipient})
	sel := minimalSelection(resolvedPair(plainPath, encPath, "yaml", []string{key.recipient}))
	report, err := verify.Check(sel, dir, verify.Options{CheckSOPSConfig: false})
	require.NoError(t, err)
	requireNoFinding(t, report, "plaintext_tracked")
	requireNoFinding(t, report, "plaintext_not_ignored")
}

func TestVCSLayerJJColocatedPreferredOverGit(t *testing.T) {
	requireJJ(t)
	dir := t.TempDir()
	isolateVCSConfig(t)
	runJJ(t, dir, "git", "init", "--colocate")
	key := newTestKey(t)
	plain := []byte("token: secret\n")
	plainPath := makePlaintext(t, dir, "config.yaml", plain)
	encPath := makeEncrypted(t, dir, "config.enc.yaml", "yaml", plain, []string{key.recipient})
	// Commit via jj so it's in @-
	runJJ(t, dir, "commit", "-m", "add plaintext")
	sel := minimalSelection(resolvedPair(plainPath, encPath, "yaml", []string{key.recipient}))
	report, err := verify.Check(sel, dir, verify.Options{CheckSOPSConfig: false})
	require.NoError(t, err)
	requireFinding(t, report, "plaintext_tracked", verify.SeverityError)
}

// ── JSON output: no sensitive data ────────────────────────────────────────

func TestFindingsContainNoPlaintextOrKeyMaterial(t *testing.T) {
	dir := t.TempDir()
	key := newTestKey(t)
	plain := []byte("token: supersecret\n")
	encPath := makeEncrypted(t, dir, "config.enc.yaml", "yaml", plain, []string{key.recipient})
	kf := writeKeyFile(t, dir, key)
	plainPath := makePlaintext(t, dir, "config.yaml", plain)
	sel := minimalSelection(resolvedPair(plainPath, encPath, "yaml", []string{key.recipient}))
	report, err := verify.Check(sel, dir, verify.Options{
		KeyFile:         kf,
		DecryptMode:     verify.DecryptAuto,
		CheckSOPSConfig: false,
	})
	require.NoError(t, err)
	var buf bytes.Buffer
	for _, f := range report.Findings {
		fmt.Fprintf(&buf, "%s %s %s", f.Message, f.Hint, f.Code)
	}
	out := buf.String()
	assert.NotContains(t, out, "supersecret", "findings must not embed plaintext content")
	assert.NotContains(t, out, key.identity, "findings must not embed private key material")
}

func TestVCSLayerKeyFileTrackedIsError(t *testing.T) {
	dir := t.TempDir()
	isolateVCSConfig(t)
	runGit(t, dir, "init", "-q")
	key := newTestKey(t)
	kf := writeKeyFile(t, dir, key)
	runGit(t, dir, "add", "keys.txt")
	runGit(t, dir, "commit", "-q", "-m", "leak key")
	encPath := makeEncrypted(t, dir, "config.enc.yaml", "yaml", []byte("token: secret\n"), []string{key.recipient})
	sel := minimalSelection(resolvedPair(filepath.Join(dir, "config.yaml"), encPath, "yaml", []string{key.recipient}))
	report, err := verify.Check(sel, dir, verify.Options{KeyFile: kf, DecryptMode: verify.DecryptDisabled})
	require.NoError(t, err)
	requireFinding(t, report, "key_tracked", verify.SeverityError)
}

func TestVCSLayerShadowedSOPSAgeKeyFileIsChecked(t *testing.T) {
	dir := t.TempDir()
	isolateVCSConfig(t)
	runGit(t, dir, "init", "-q")
	key := newTestKey(t)
	kf := writeKeyFile(t, dir, key)
	t.Setenv("SOPS_AGE_KEY_FILE", kf)
	encPath := makeEncrypted(t, dir, "config.enc.yaml", "yaml", []byte("token: secret\n"), []string{key.recipient})
	sel := minimalSelection(resolvedPair(filepath.Join(dir, "config.yaml"), encPath, "yaml", []string{key.recipient}))
	report, err := verify.Check(sel, dir, verify.Options{DecryptMode: verify.DecryptDisabled})
	require.NoError(t, err)
	requireFinding(t, report, "key_not_ignored", verify.SeverityWarning)
}

func TestVCSLayerSpecialFileNamesAreLiteral(t *testing.T) {
	dir := t.TempDir()
	isolateVCSConfig(t)
	runGit(t, dir, "init", "-q")
	key := newTestKey(t)
	plain := []byte("token: secret\n")
	plainPath := makePlaintext(t, dir, "we ird*[name].yaml", plain)
	runGit(t, dir, "add", "--", "we ird*[name].yaml")
	runGit(t, dir, "commit", "-q", "-m", "special name")
	encPath := makeEncrypted(t, dir, "special.enc.yaml", "yaml", plain, []string{key.recipient})
	sel := minimalSelection(resolvedPair(plainPath, encPath, "yaml", []string{key.recipient}))
	report, err := verify.Check(sel, dir, verify.Options{})
	require.NoError(t, err)
	requireFinding(t, report, "plaintext_tracked", verify.SeverityError)
}

func TestVCSLayerQueryFailureIsError(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0755)) // not a valid repository
	key := newTestKey(t)
	encPath := makeEncrypted(t, dir, "config.enc.yaml", "yaml", []byte("token: secret\n"), []string{key.recipient})
	sel := minimalSelection(resolvedPair(filepath.Join(dir, "config.yaml"), encPath, "yaml", []string{key.recipient}))
	report, err := verify.Check(sel, dir, verify.Options{})
	require.NoError(t, err)
	requireFinding(t, report, "vcs_query_failed", verify.SeverityError)
}

func TestVCSLayerJJCommitThenIgnoreStillError(t *testing.T) {
	requireJJ(t)
	dir := t.TempDir()
	isolateVCSConfig(t)
	runJJ(t, dir, "git", "init", "--no-colocate")
	key := newTestKey(t)
	plain := []byte("token: secret\n")
	plainPath := makePlaintext(t, dir, "config.yaml", plain)
	runJJ(t, dir, "commit", "-m", "add plaintext")
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("config.yaml\n"), 0644))
	encPath := makeEncrypted(t, dir, "config.enc.yaml", "yaml", plain, []string{key.recipient})
	sel := minimalSelection(resolvedPair(plainPath, encPath, "yaml", []string{key.recipient}))
	report, err := verify.Check(sel, dir, verify.Options{})
	require.NoError(t, err)
	requireFinding(t, report, "plaintext_tracked", verify.SeverityError)
}
