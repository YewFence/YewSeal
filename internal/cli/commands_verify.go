package cli

import (
	"fmt"
	"os"

	yewsapp "github.com/YewFence/YewSeal/internal/app"
	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/errx"
	"github.com/YewFence/YewSeal/internal/presentation"

	"github.com/spf13/cobra"
)

func verifyCommand(load configLoader) *cobra.Command {
	opts := verifyOptions{}
	var resolver *optionResolver

	cmd := &cobra.Command{
		Use:   "verify [command options] [path-or-pattern]...",
		Short: "Check project health: ciphertext integrity, recipient drift, plaintext safety, and VCS exposure",
		Long: `Inspect a project's health across four independent layers without
modifying any project file:

  1. Configuration layer: resolves selection and authorization using the
     same strict path as encrypt and plan; bad config or unknown aliases
     fail the command before any per-file check runs.

  2. Ciphertext static layer (no private key required): for each selected
     file verify that the encrypted path exists as a regular file, that
     SOPS can parse it, that SOPS metadata contains at least one Age
     recipient, and that the recipient set matches the canonical public
     keys resolved from the current config. Recipient alias renames that
     keep the same public key do not produce a finding.

  3. Decrypt layer (requires a private key): decrypt each ciphertext and
     verify its MAC. When a local plaintext exists, compare its bytes
     against the decrypted content. Without any available identity, this
     layer is skipped and the skip is stated in the output. With
     --no-decrypt, this layer is always skipped. With --decrypt, a missing
     identity or an identity that cannot open a file is an error.

  4. VCS layer: detect the repository type (.jj beats .git for colocated
     repos) and classify each plaintext file and any file-backed Age key
     against the dual-list contract — files already in history are errors,
     files that are one commit away are warnings. Repos with both .jj and
     .git are treated as jj repos because jj controls the working copy.
     A non-VCS directory makes this layer skip with a notice.

  5. .sops.yaml drift: compare the on-disk .sops.yaml against what the
     current resolved policy would generate. A mismatch is an error; an
     absent .sops.yaml is a skip. This layer is skipped when
     YEWSEAL_SYNC_SOPS_CONFIG is true (sync is active, so drift is
     expected to be repaired on the next encrypt).

Target selection is directionless, matching either side of each mapping,
and follows the same rules as plan: no argument means the current
directory scope; a registered path or an existing directory selects
matching pairs; patterns with metacharacters are matched against both
sides; any argument matching nothing is an error.

Exit codes:
  0  verify completed with no error-severity findings
  1  verify completed but produced at least one error finding
  2  verify could not run: config invalid, target selection failed,
     --decrypt with no identity, --decrypt and --no-decrypt together,
     or report delivery failed

CI usage: run without --decrypt for pure static analysis. Add --decrypt
with a key available in the environment for full content verification.

Output: summary and findings to stdout; warnings and diagnostics to
stderr. --json prints only a JSON object on stdout; errors stay on stderr.
verify never prints plaintext, private keys, data keys, or plaintext
digests; recipients are shown as aliases or short public keys.

See also: "yews plan" to audit mappings and authorization without touching
ciphertext, "yews encrypt" to repair recipient drift, "yews diff" to
inspect content differences.

Documentation: ` + docsVerify,
		Example: `  # Run all static checks (no private key needed)
  yews verify

  # Verify a single registered file
  yews verify config/production.yaml

  # Force the decrypt layer; fail if no identity is available
  yews verify --decrypt --key-file /run/secrets/age-identities

  # Static-only even if an identity is configured
  yews verify --no-decrypt

  # Machine-readable output for CI scripts (errors on stderr)
  yews verify --json > health.json`,
		Args: func(cmd *cobra.Command, args []string) error {
			if opts.Decrypt && opts.NoDecrypt {
				return errx.Usage(fmt.Errorf("--decrypt and --no-decrypt are mutually exclusive"))
			}
			for _, arg := range args {
				if err := validateTargetArg(arg); err != nil {
					return err
				}
			}
			return nil
		},
		RunE: withConfig(load, func(cmd *cobra.Command, args []string, cfg *config.Config) error {
			syncSOPSConfig := opts.SyncSOPSConfig
			if !resolver.IsSet("sync-sops-config") {
				// Inherit from the YEWSEAL_SYNC_SOPS_CONFIG top-level env if not explicitly set.
				if v := os.Getenv("YEWSEAL_SYNC_SOPS_CONFIG"); v == "true" || v == "1" {
					syncSOPSConfig = true
				}
			}
			out := presentation.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), opts.Verbose)
			result, err := yewsapp.Verify(cfg, yewsapp.VerifyRequest{
				Targets:        args,
				KeyFile:        opts.KeyFile,
				Decrypt:        opts.Decrypt,
				NoDecrypt:      opts.NoDecrypt,
				SyncSOPSConfig: syncSOPSConfig,
				JSON:           opts.JSON,
				Verbose:        opts.Verbose,
			})
			if err != nil {
				return err
			}
			reportErr := out.Finish(out.VerifyReport(result.Report, presentation.VerifyPrintOptions{
				JSON:    opts.JSON,
				Verbose: opts.Verbose,
				CWD:     config.CurrentDir(cfg),
			}))
			if reportErr != nil {
				return reportErr
			}
			if result.ExitCode == 1 {
				return errx.NewFindingsError()
			}
			return nil
		}),
	}
	cmd.Flags().BoolVar(&opts.Decrypt, "decrypt", false, "Require a private identity; mismatch or missing identity exits 2 (env YEWSEAL_VERIFY_DECRYPT)")
	cmd.Flags().BoolVar(&opts.NoDecrypt, "no-decrypt", false, "Skip the decrypt layer entirely, even if an identity is available")
	cmd.Flags().BoolVar(&opts.SyncSOPSConfig, "sync-sops-config", false, "Skip the .sops.yaml drift check (set when sync is already active)")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Print the verify report as JSON on stdout (diagnostics stay on stderr)")
	cmd.Flags().BoolVarP(&opts.Verbose, "verbose", "v", false, "Enable verbose output")
	resolver = newOptionResolver(cmd, &opts)
	cmd.Args = resolver.before(cmd.Args)
	return cmd
}
