package app

import (
	"fmt"
	"io"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/crypto"
)

type DiffResult struct {
	Different bool
}

func DiffPlaintextAgainstEncryptedTargets(w io.Writer, cfg *config.Config, target, keyFile, cliFormat string, verbose bool) (DiffResult, error) {
	filePairs, err := ResolveTargetFilePairs(cfg, target)
	if err != nil {
		return DiffResult{}, err
	}

	different := false
	for _, filePair := range filePairs {
		formatOverride, err := ResolveFormatOverride(cliFormat, filePair)
		if err != nil {
			return DiffResult{}, err
		}

		result, err := crypto.DiffPlaintextAgainstEncrypted(
			filePair.PlaintextPath,
			filePair.EncryptedPath,
			keyFile,
			formatOverride,
			verbose,
		)
		if err != nil {
			return DiffResult{}, err
		}
		if result.Different {
			different = true
			if _, err := fmt.Fprint(w, result.Diff); err != nil {
				return DiffResult{}, err
			}
		}
	}

	return DiffResult{Different: different}, nil
}
