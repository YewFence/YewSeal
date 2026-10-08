package cli

import (
	"fmt"

	yewsapp "github.com/YewFence/YewSeal/internal/app"
	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/errx"
	"github.com/YewFence/YewSeal/internal/presentation"

	"github.com/spf13/cobra"
)

func verifyCommand(load configLoader) *cobra.Command {
	opts := verifyOptions{SyncSOPSConfig: true}

	cmd := &cobra.Command{
		Use:   "verify [command options] [path-or-pattern]...",
		Short: "Check ciphertext, recipients, plaintext consistency, version-control exposure, and .sops.yaml drift",
		Long: `Check the health of registered encrypted files without modifying any
project file. verify never prints plaintext, private keys, or data keys.

With no arguments, verify selects mappings with either path under the current
directory and its subdirectories. Use file paths, directories, or patterns
to select targets; directories and patterns match either path.
Missing ciphertext is a finding.

Checks:
  configuration   the same strict authorization as plan and encrypt; an
                  invalid config, unknown alias, empty recipient set, or
                  group conflict is a calling error, not a finding.
  ciphertext      no private key needed: the encrypted file exists, is a
                  regular file, parses as SOPS, contains exactly one key
                  group with no non-Age keys, and its Age recipients equal
                  the configured public keys
                  (order ignored; renaming an alias without changing its
                  key is not drift).
  decryption      runs when an Age identity is available: decrypts, checks
                  the MAC, and compares an existing local plaintext with
                  decrypted content by parsed structure. Without an identity it is
                  skipped. --decrypt requires an identity and treats a
                  file no identity can open as an error; --no-decrypt reads
                  no identity source. A missing plaintext is skipped.
  version control selected plaintext and file-backed Age keys in the nearest
                  git or jj repository (.jj wins when both exist): tracked
                  files and unignored files awaiting a commit are errors.
                  In git, staged files count as tracked; in jj, files in a
                  parent of @ count as tracked and unignored files only in
                  @ are also errors. Ignoring a tracked file does not
                  clear the error. Symlinks and their targets are both
                  checked. Past commits are not audited. Outside a repository
                  this check is skipped; a failed VCS query is an error.
                  With jj, verification inspects an uncommitted snapshot
                  without updating the working-copy commit.
  .sops.yaml      checked against the managed-file rules, independently of
                  selected targets. A difference is an error; an absent file
                  is skipped. --sync-sops-config=false skips the comparison.

Finding codes are stable: ciphertext_missing, ciphertext_not_regular,
ciphertext_stat_error, ciphertext_read_error, ciphertext_parse_error,
ciphertext_no_recipients, key_groups_unsupported, recipient_unsupported, recipient_missing, recipient_extra,
recipient_duplicate, decrypt_failed, mac_mismatch,
decrypt_no_matching_identity (warning, error with --decrypt),
plaintext_read_error, plaintext_parse_error, plaintext_drift, plaintext_tracked,
plaintext_not_ignored, key_tracked, key_not_ignored,
vcs_query_failed, sops_config_read_error, sops_config_generate_error,
sops_config_drift. All are errors unless marked.

Output: stdout carries a summary line, one line per skipped check kind,
then each finding with a hint. The summary counts individual checks
(each file contributes one check per layer): pass and skipped count
checks, warning and error count findings. --json prints only
{"ok", "summary", "skipped", "findings"} on stdout, where "skipped" lists
the skip reasons. Errors go to stderr.

Exit codes: 0 when no finding is an error (warnings and skips allowed);
1 when at least one finding is an error ("fix the repository"); 2 when
verify cannot run or report ("fix the environment"): invalid arguments
(including --decrypt with --no-decrypt), a missing or invalid
.yewseal.toml, selection or authorization failure, an unreadable explicit
key file, a failed key command, --decrypt without any identity, or a
report that cannot be written.

See also: "yews plan" for mappings and authorization only, "yews encrypt"
to repair recipient drift, "yews diff" to inspect a plaintext difference.

Target selection: ` + docsTargetSelect + `
Managed-file rules: ` + docsManagedFiles + `
Verification workflows: ` + docsVerify,
		Example: `  # Check registered mappings under the current directory
  yews verify

  # Check one registered mapping
  yews verify config/production.yaml

  # CI without private keys: static checks only
  yews verify --no-decrypt

  # CI with a key: fail unless every file can be decrypted
  yews verify --decrypt --key-file /run/secrets/yewseal-identities

  # Machine-readable report
  yews verify --json > verify.json`,
		Args: func(_ *cobra.Command, args []string) error {
			if opts.Decrypt && opts.NoDecrypt {
				return fmt.Errorf("--decrypt and --no-decrypt are mutually exclusive")
			}
			for _, arg := range args {
				if err := validateTargetArg(arg); err != nil {
					return err
				}
			}
			return nil
		},
		RunE: withConfig(load, func(cmd *cobra.Command, args []string, cfg *config.Config) error {
			report, err := yewsapp.Verify(cfg, yewsapp.VerifyRequest{
				Targets:        args,
				KeyFile:        opts.KeyFile,
				Decrypt:        opts.Decrypt,
				NoDecrypt:      opts.NoDecrypt,
				SyncSOPSConfig: opts.SyncSOPSConfig,
			})
			if err != nil {
				return err
			}
			out := presentation.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), false)
			out.SetDirectory(config.CurrentDir(cfg))
			// An undeliverable report means verify could not report its result.
			if err := out.Finish(out.VerifyReport(report, opts.JSON)); err != nil {
				return errx.Usage(err)
			}
			if !report.OK() {
				return errx.NewFindingsError()
			}
			return nil
		}),
	}
	cmd.Flags().BoolVar(&opts.Decrypt, "decrypt", false, "Require an Age identity and treat undecryptable files as errors")
	cmd.Flags().BoolVar(&opts.NoDecrypt, "no-decrypt", false, "Skip decryption checks without reading any identity source")
	cmd.Flags().BoolVar(&opts.SyncSOPSConfig, "sync-sops-config", opts.SyncSOPSConfig, "Check .sops.yaml using the managed-file rules; false skips the comparison")
	markSharedEnv(cmd.Flags().Lookup("sync-sops-config"), syncSOPSConfigEnv)
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Print the verify report as JSON on stdout (errors stay on stderr)")
	resolver := newOptionResolver(cmd, &opts)
	cmd.Args = resolver.before(cmd.Args)
	return cmd
}
