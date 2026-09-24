package verify

import (
	"os"
	"path/filepath"

	"github.com/YewFence/YewSeal/internal/agekey"
	"github.com/YewFence/YewSeal/internal/config"
)

// Options controls one verify run.
type Options struct {
	KeyFile         string
	DecryptMode     DecryptBehavior
	CheckSOPSConfig bool
	// RecipientAliases maps canonical public keys to registry aliases.
	RecipientAliases map[string]string
}

// ErrIdentityRequired is returned when DecryptRequired finds no identity.
type ErrIdentityRequired struct{}

func (ErrIdentityRequired) Error() string {
	return "--decrypt requires an Age identity, but no identity source provided one"
}

// Check runs every verification layer over the resolved selection. cwd is the
// directory relative identity paths and .sops.yaml resolve against. The
// returned error means verify could not run; problems found in the project
// are findings in the report.
func Check(selection config.ResolvedSelection, cwd string, opts Options) (*Report, error) {
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
		return nil, ErrIdentityRequired{}
	}

	labels := recipientLabels(opts.RecipientAliases)
	for _, pair := range selection.FilePairs {
		usable := checkCiphertext(report, pair, labels)
		switch {
		case !usable:
			report.AddSkip("ciphertext unusable; decryption not attempted for those files")
		case opts.DecryptMode == DecryptDisabled:
			report.AddSkip("decrypt layer disabled by --no-decrypt; decryption, MAC, and plaintext consistency not checked")
		case !hasIdentity:
			report.AddSkip("no Age identity available; decryption, MAC, and plaintext consistency not checked")
		default:
			checkDecrypt(report, pair, bundle, opts.DecryptMode)
		}
	}

	checkVersionControl(report, selection.FilePairs, cwd, opts.KeyFile)

	if opts.CheckSOPSConfig {
		checkSOPSDrift(report, selection.AllConfigPairs, cwd)
	} else {
		report.AddSkip(".sops.yaml is not managed (--sync-sops-config=false); drift not checked")
	}

	report.Sort()
	return report, nil
}

func checkVersionControl(report *Report, pairs []config.ResolvedFilePair, cwd, keyFile string) {
	root, adapter := detectVCS(cwd)
	if adapter == nil {
		report.AddSkip("no VCS repository found; version-control exposure not checked")
		return
	}
	state, err := queryVCS(root, adapter)
	if err != nil {
		report.Add(Finding{
			Code:     "vcs_query_failed",
			Severity: SeverityError,
			Message:  "failed to query " + adapter.name() + " repository " + root + ": " + err.Error(),
			Hint:     "make sure " + adapter.name() + " is installed and the repository is readable",
		})
		return
	}
	for _, pair := range pairs {
		state.classify(report, pair.PlaintextPath, plaintextSubject, Finding{
			PlaintextPath: pair.PlaintextPath,
			EncryptedPath: pair.EncryptedPath,
		})
	}
	keyPaths, valueSources := identityFileSources(cwd, keyFile)
	for _, path := range keyPaths {
		state.classify(report, path, keySubject, Finding{})
	}
	for _, source := range valueSources {
		report.AddSkip("identity source " + source + " has no file; version-control exposure not checked")
	}
}

// identityFileSources lists every configured file-backed identity source,
// present or shadowed: a key file committed to the repository leaks whether
// or not it wins identity resolution. It never reads source contents or runs
// key commands, so it is safe under --no-decrypt.
func identityFileSources(cwd, keyFile string) (paths []string, valueSources []string) {
	seen := make(map[string]bool)
	add := func(path string) {
		if !filepath.IsAbs(path) {
			path = filepath.Join(cwd, path)
		}
		path = filepath.Clean(path)
		if !seen[path] {
			seen[path] = true
			paths = append(paths, path)
		}
	}
	if keyFile != "" {
		add(keyFile)
	}
	if path := os.Getenv("SOPS_AGE_KEY_FILE"); path != "" {
		add(path)
	}
	if _, err := os.Lstat(filepath.Join(cwd, ".age", "keys.txt")); err == nil {
		add(filepath.Join(".age", "keys.txt"))
	}
	for _, name := range []string{"YEWSEAL_AGE_IDENTITIES", "SOPS_AGE_KEY", "YEWSEAL_AGE_KEY_CMD", "SOPS_AGE_KEY_CMD"} {
		if os.Getenv(name) != "" {
			valueSources = append(valueSources, "env:"+name)
		}
	}
	return paths, valueSources
}
