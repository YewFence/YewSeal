package verify

import (
	"os"

	"github.com/YewFence/YewSeal/internal/agekey"
	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/task"
)

// Options controls the verify run.
type Options struct {
	KeyFile        string
	DecryptMode    DecryptBehavior
	SyncSOPSConfig bool // when true, skip .sops.yaml drift check
}

// Check runs all four verification layers against the resolved selection.
// It is the primary entry point for internal/app.
func Check(selection config.ResolvedSelection, cwd string, opts Options) (*Report, error) {
	_ = task.ModeVerify // ensure constant is reachable

	report := &Report{}

	var bundle agekey.IdentityBundle
	if opts.DecryptMode != DecryptDisabled {
		var err error
		bundle, err = agekey.GetIdentityBundle(opts.KeyFile)
		if err != nil {
			return nil, err
		}
	}

	hasIdentity := len(bundle.Identities()) > 0

	if !hasIdentity && opts.DecryptMode == DecryptRequired {
		return nil, &identityRequiredErr{}
	}

	repoRoot, adapter, err := detectVCS(cwd)
	if err != nil {
		return nil, err
	}
	vcsAvailable := adapter != nil

	keyFilePaths := resolveKeyFilePaths(opts.KeyFile)

	for _, pair := range selection.FilePairs {
		// Layer 2: static ciphertext check (no key required).
		checkCiphertext(report, pair)

		// Layer 3: decrypt layer.
		switch {
		case opts.DecryptMode == DecryptDisabled:
			report.AddSkip("decrypt layer disabled via --no-decrypt")
		case !hasIdentity:
			report.AddSkip("no identity available; decrypt and plaintext consistency layers skipped")
		default:
			checkDecrypt(report, pair, bundle, opts.DecryptMode)
		}

		// Layer 4: VCS check for this plaintext.
		if !vcsAvailable {
			report.AddSkip("no VCS repository detected; VCS layer skipped")
		} else {
			if len(keyFilePaths) > 0 {
				for _, kp := range keyFilePaths {
					checkVCS(report, pair.PlaintextPath, kp, repoRoot, adapter)
				}
			} else {
				checkVCS(report, pair.PlaintextPath, "", repoRoot, adapter)
			}
		}
	}

	// Layer 5: .sops.yaml drift.
	if opts.SyncSOPSConfig {
		report.AddSkip(".sops.yaml sync is active; drift check skipped")
	} else {
		orig, _ := os.Getwd()
		if cwd != "" && cwd != orig {
			_ = os.Chdir(cwd)
		}
		checkSOPSDrift(report, selection.AllConfigPairs)
		if cwd != "" && cwd != orig {
			_ = os.Chdir(orig)
		}
	}

	report.Sort()
	return report, nil
}

func resolveKeyFilePaths(explicitKeyFile string) []string {
	if explicitKeyFile != "" {
		return []string{explicitKeyFile}
	}
	if _, err := os.Stat(".age/keys.txt"); err == nil {
		return []string{".age/keys.txt"}
	}
	return nil
}

type identityRequiredErr struct{}

func (e *identityRequiredErr) Error() string {
	return "--decrypt requires an Age identity; none is available from any configured source"
}
