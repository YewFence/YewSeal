package verify_test

import (
	"path/filepath"
	"testing"

	"github.com/YewFence/YewSeal/internal/verify"
	"github.com/stretchr/testify/require"
)

func TestVCSLayerJJUnignoredKeyIsError(t *testing.T) {
	requireJJ(t)
	dir := t.TempDir()
	isolateVCSConfig(t)
	runJJ(t, dir, "git", "init")
	key := newTestKey(t)
	keyFile := writeKeyFile(t, dir, key)
	encrypted := makeEncrypted(t, dir, "config.enc.yaml", "yaml", []byte("value: fixture\n"), []string{key.recipient})
	selection := minimalSelection(resolvedPair(filepath.Join(dir, "config.yaml"), encrypted, "yaml", []string{key.recipient}))
	report, err := verify.Check(selection, dir, verify.Options{
		KeyFile:     keyFile,
		DecryptMode: verify.DecryptDisabled,
	})
	require.NoError(t, err)
	requireFinding(t, report, "key_not_ignored", verify.SeverityError)
	require.False(t, report.OK())
}
