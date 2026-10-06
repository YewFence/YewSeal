package verify_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/sopsconfig"
	"github.com/YewFence/YewSeal/internal/verify"
	"github.com/stretchr/testify/require"
)

func TestSOPSDriftChecksCompleteCurrentDirectoryPolicy(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "service")
	require.NoError(t, os.Mkdir(dir, 0755))
	key := newTestKey(t)
	encPath := makeEncrypted(t, dir, "secret.enc.yaml", "yaml", []byte("token: fixture\n"), []string{key.recipient})
	selected := resolvedPair(filepath.Join(dir, "secret.yaml"), encPath, "yaml", []string{key.recipient})
	unselected := resolvedPair(filepath.Join(dir, "second.yaml"), filepath.Join(dir, "second.enc.yaml"), "yaml", []string{key.recipient})
	outside := resolvedPair(filepath.Join(root, "root.yaml"), filepath.Join(root, "root.enc.yaml"), "yaml", []string{key.recipient})
	selection := config.ResolvedSelection{
		FilePairs:      []config.ResolvedFilePair{selected},
		AllConfigPairs: []config.ResolvedFilePair{outside, selected, unselected},
	}
	localPairs := []config.ResolvedFilePair{selected, unselected}
	rendered, err := sopsconfig.Render(localPairs, dir)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".sops.yaml"), rendered, 0644))
	report, err := verify.Check(selection, dir, verify.Options{
		DecryptMode:     verify.DecryptDisabled,
		CheckSOPSConfig: true,
	})
	require.NoError(t, err)
	requireNoFinding(t, report, "sops_config_drift")

	rendered, err = sopsconfig.Render(localPairs[:1], dir)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".sops.yaml"), rendered, 0644))
	report, err = verify.Check(selection, dir, verify.Options{
		DecryptMode:     verify.DecryptDisabled,
		CheckSOPSConfig: true,
	})
	require.NoError(t, err)
	requireFinding(t, report, "sops_config_drift", verify.SeverityError)
}
