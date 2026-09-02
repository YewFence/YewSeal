package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/YewFence/YewSeal/internal/task"
	"github.com/stretchr/testify/require"
)

func TestSelectFilePairsExplicitFileOverridesGroupPair(t *testing.T) {
	root := t.TempDir()
	plaintext := filepath.Join(root, "config.yaml")
	encrypted := filepath.Join(root, "config.enc.yaml")
	require.NoError(t, os.WriteFile(plaintext, []byte("secret: value\n"), 0644))

	fileRecipients := []string{"file"}
	cfg := &Config{
		CurrentDir: root,
		UserConfig: true,
		Encryption: EncryptionConfig{
			Files:  []FilePair{{PlaintextPath: plaintext, EncryptedPath: encrypted, Recipients: &fileRecipients}},
			Groups: []GroupConfig{{Patterns: []string{"config.yaml"}, ConfigDir: root}},
		},
	}

	result, err := SelectFilePairs(cfg, SelectionOptions{Command: task.ModeEncrypt, Target: plaintext})
	require.NoError(t, err)
	require.Len(t, result.FilePairs, 1)
	require.NotNil(t, result.FilePairs[0].Recipients)
	require.Equal(t, fileRecipients, *result.FilePairs[0].Recipients)
}
