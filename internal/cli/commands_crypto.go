package cli

import (
	yewsapp "github.com/YewFence/YewSeal/internal/app"
	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/presentation"

	"github.com/spf13/cobra"
)

func encryptCommand(load configLoader) *cobra.Command {
	opts := encryptOptions{
		Parallel:       1,
		SyncSOPSConfig: true,
	}
	var resolver *optionResolver

	cmd := &cobra.Command{
		Use:     "encrypt [command options] [path-or-pattern]...",
		Aliases: []string{"e"},
		Short:   "Encrypt configuration file (supports .toml, .yaml, .yml, .json, .env, .ini, and binary output)",
		Long: `Encrypt registered configuration files with SOPS and Age. Supported
formats: .toml, .yaml, .yml, .json, .env, .ini, and binary output; every
format is encrypted natively by the embedded SOPS engine and the
ciphertext keeps the original format.

Target selection (no argument: every file and group in .yewseal.toml
within the current directory scope):
  - a registered plaintext or encrypted path selects that single mapping;
  - an existing directory selects mappings whose plaintext side is inside
    it (groups always scan by their own config directory, never by the
    target directory);
  - arguments containing *, ?, and similar metacharacters are patterns
    matched against registered plaintext paths (a leading / anchors to
    the current working directory, ** is supported);
  - multiple arguments take the union; patterns only include, excludes
    come from group "patterns" in the config; any argument matching
    nothing is an error.

Recipients come strictly from alias resolution in .yewseal.toml (file
"recipients" > group > top-level recipients.defaults); there is no
--public-key flag. An empty final set, an unknown alias, or groups
disagreeing on the same path fails the whole batch before any ciphertext
is written.

When ciphertext already exists, encrypt uses an available private identity
to verify and update it in place: unchanged files remain byte-identical,
unchanged values retain their ciphertext, and recipient-only changes only
rewrap the existing data key. If no identity is available, or none matches a
specific file, encrypt warns and replaces that ciphertext from the current
plaintext. --force always performs this fresh encryption and rotates the data
key without reading the old ciphertext. Missing plaintext is reported and
skipped without creating an output directory.

--output only changes the location, never the format. There is no
--format flag: non-standard extensions are declared via "format" or
"format_rules" in the config. Group results get the format's standard
.enc.* path; --output applies to single file targets only, never to
config-wide or directory-driven batches.

After processing, --sync-sops-config (enabled by default) rewrites
.sops.yaml from the complete resolved project policy, not only the selected
targets. Encryption always finishes before synchronization is attempted.
A synchronization failure leaves completed ciphertext work in place but
makes the command fail.

Exit codes: 0 on success (skips without real errors allowed); 1 when
any file fails, an all-skipped batch occurs, .sops.yaml synchronization
fails, or output delivery fails. Exit 1 does not imply that no ciphertext
was written; 2 for calling errors: invalid arguments, a missing or invalid
.yewseal.toml, selection or authorization failure, or an unusable
identity source.

Output: ciphertext goes to files and stdout stays empty; warnings,
per-file failure reasons, and the summary go to stderr (--verbose adds
selection info and per-file success). --json replaces stdout with the
batch report (summary and per-file outcomes); stderr diagnostics stay
unchanged, and when the run never starts (a calling error, exit 2)
stdout stays empty.

To encrypt an unregistered file ad hoc without a project config, use
the SOPS CLI directly.

See also: "yews plan" to audit mappings and authorization (not an
encrypt dry run), "yews diff" to compare plaintext with the stored
ciphertext.

Documentation: ` + docsTargetSelect,
		Example: `  # Encrypt every file registered in the config
  yews encrypt

  # Encrypt one registered mapping (either side locates it)
  yews encrypt config.toml
  yews encrypt config.enc.toml

  # Temporarily override the output path of a single target
  yews encrypt config.toml -o review/config.enc.toml

  # Select registered .toml plaintext under ./configs with a pattern
  yews encrypt './configs/*.toml'

  # Encrypt a batch across four parallel workers
  yews encrypt ./configs --parallel 4

  # Print the batch report for scripts (diagnostics stay on stderr)
  yews encrypt --json > report.json`,
		Args: func(cmd *cobra.Command, args []string) error {
			return validateBatchArgs(args, opts.Parallel, resolver.IsSet("output"))
		},
		RunE: withConfig(load, func(cmd *cobra.Command, args []string, cfg *config.Config) error {
			return yewsapp.EncryptFiles(cfg, yewsapp.EncryptRequest{
				Presentation:          presentation.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), opts.Verbose),
				KeyFile:               opts.KeyFile,
				Output:                opts.Output,
				OutputSet:             resolver.IsSet("output"),
				Targets:               args,
				Parallel:              opts.Parallel,
				Force:                 opts.Force,
				JSON:                  opts.JSON,
				UpdateProjectMetadata: true,
				SyncSOPSConfig:        opts.SyncSOPSConfig,
			})
		}),
	}
	addEncryptFlags(cmd.Flags(), &opts)
	resolver = newOptionResolver(cmd, &opts)
	cmd.Args = resolver.before(cmd.Args)
	return cmd
}

func decryptCommand(load configLoader) *cobra.Command {
	opts := decryptOptions{
		Parallel: 1,
	}
	var resolver *optionResolver

	cmd := &cobra.Command{
		Use:     "decrypt [command options] [path-or-pattern]...",
		Aliases: []string{"d"},
		Short:   "Decrypt encrypted file to its configured plaintext path",
		Long: `Decrypt registered SOPS-encrypted files to their configured plaintext
paths. The format comes from the project config or the registered file
path; runtime format overrides and cross-format conversion are not
supported.

Target selection (no argument: every registered file and group result
whose ciphertext side is within the current directory scope):
  - a registered plaintext or encrypted path selects that single mapping;
  - an existing directory selects mappings whose encrypted side is inside
    it (groups always scan by their own config directory, never by the
    target directory);
  - arguments containing *, ?, and similar metacharacters are patterns
    matched against registered encrypted paths (a leading / anchors to
    the current working directory, ** is supported);
  - multiple arguments take the union; patterns only include, excludes
    come from group "patterns" in the config; any argument matching
    nothing is an error.

The config still governs plaintext/ciphertext paths and formats, but the
recipients actually used for decryption come from the ciphertext's SOPS
metadata: when an alias referenced by the current config no longer
exists, decrypt warns on stderr and continues with the identity bundle.

Overwrite protection: an existing plaintext file whose content differs
from the decryption result is not overwritten unless --force is set.

Exit codes: by default, files whose keys do not match the current
identity are skipped; partial success with no real error exits 0, while
an all-skipped batch, any real error, or an output delivery failure
exits 1. With --strict, any skip also exits 1, but remaining files are
still processed and successful results are kept; --strict=false
overrides YEWSEAL_DECRYPT_STRICT. Calling errors (invalid arguments, a
missing or invalid .yewseal.toml, selection failure, or an unusable
identity source) exit 2.

TOML ciphertext is decrypted natively by the embedded TOML store without
format conversion; the output is normalized TOML (single-quoted literal
strings, comments preserved, equivalent content, possibly different
layout from the handwritten original).

Output: plaintext goes to files and stdout stays empty; warnings,
per-file skip and failure reasons, and the summary go to stderr
(--verbose adds selection info and per-file success). --json replaces
stdout with the batch report (summary and per-file outcomes); stderr
diagnostics stay unchanged, and when the run never starts (a calling
error, exit 2) stdout stays empty.

To decrypt an unregistered file ad hoc without a project config, use
the SOPS CLI directly (a fork build with the native TOML store is needed
for native TOML ciphertext).

See also: "yews plan" to audit mappings and authorization (not a decrypt
dry run, and it does not verify that the current identity can decrypt),
"yews view" to print plaintext to stdout, "yews diff" to compare
plaintext with the ciphertext.

Documentation: ` + docsTargetSelect + `
Result classification and exit codes: ` + docsDecryptResults,
		Example: `  # Decrypt every file registered in the config
  yews decrypt

  # Decrypt one registered encrypted file to its configured plaintext
  yews decrypt config.enc.toml

  # Decrypt a single target to an explicit output path
  yews decrypt config.enc.toml -o config.toml

  # Select registered ciphertext under ./configs with a pattern
  yews decrypt './configs/*.enc.toml'

  # Overwrite a plaintext file that differs from the decrypted content
  yews decrypt config.enc.toml --force

  # Fail on any skipped file (or set YEWSEAL_DECRYPT_STRICT=true)
  yews decrypt --strict

  # Print the batch report for scripts (diagnostics stay on stderr)
  yews decrypt --json > report.json`,
		Args: func(cmd *cobra.Command, args []string) error {
			return validateBatchArgs(args, opts.Parallel, resolver.IsSet("output"))
		},
		RunE: withConfig(load, func(cmd *cobra.Command, args []string, cfg *config.Config) error {
			return yewsapp.DecryptFiles(cfg, yewsapp.DecryptRequest{
				Presentation:          presentation.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), opts.Verbose),
				KeyFile:               opts.KeyFile,
				Output:                opts.Output,
				OutputSet:             resolver.IsSet("output"),
				Targets:               args,
				Parallel:              opts.Parallel,
				Force:                 opts.Force,
				Strict:                opts.Strict,
				JSON:                  opts.JSON,
				UpdateProjectMetadata: true,
			})
		}),
	}
	addDecryptFlags(cmd.Flags(), &opts)
	resolver = newOptionResolver(cmd, &opts)
	cmd.Args = resolver.before(cmd.Args)
	return cmd
}

func planCommand(load configLoader) *cobra.Command {
	opts := planOptions{}

	cmd := &cobra.Command{
		Use:   "plan [command options] [path-or-pattern]...",
		Short: "Check configured file mappings, formats, and recipient authorization without writing files",
		Long: `Inspect registered file mappings, formats, and current-config
authorization without encrypting, decrypting, or writing any file. This
is a directionless mapping check, not an encrypt/decrypt dry run:
success does not guarantee that files can be encrypted, that ciphertext
can be opened, or that output paths are writable.

Target selection (no argument: mappings with either side within the
current directory scope):
  - a registered plaintext or encrypted path selects that single mapping;
  - an existing directory selects mappings with either side inside it;
  - arguments containing *, ?, and similar metacharacters are patterns
    matched against either side of registered mappings;
  - multiple arguments take the union; patterns only include; any
    argument matching nothing is an error.
Paths only select mappings; they never imply an operation direction.

Groups always scan by their own config directory, and plan uses the
union of plaintext-side and ciphertext-side discovery: files present on
only one side still show up (for example config.yml next to
config.enc.yaml keeps the discovered real plaintext path). Competing
mappings must be resolved by an explicit file entry or plan reports a
conflict. Explicit entries do not require the files to exist.

plan applies the same strict authorization semantics as encrypt to
every mapping resolved from the loaded config, including unselected
ones: unknown aliases, empty recipient sets, and group conflicts fail
the run. The report includes recipient aliases, canonical recipients,
registry origins, and the effective authorization source. plan does not
load an identity bundle and does not read ciphertext content or
metadata, so it has no historical-decrypt tolerance.

Output: a table report on stdout (config count, selection scope, file
mappings); --json prints only JSON; errors go to stderr and never mix
into the report. plan defines no output or worker flags, reads no
output-related environment variables, and its report contains no
metadata write plan.

Exit codes: 0 on success; calling errors (invalid patterns, a missing
or invalid .yewseal.toml, or authorization conflicts) exit 2.

See also: "yews encrypt" and "yews decrypt" share the registered-mapping
selection, with different discovery sides and historical-decrypt
authorization handling.

Documentation: ` + docsConfiguration,
		Example: `  # Inspect the mappings within the current directory scope
  yews plan

  # Either side of a mapping selects it
  yews plan config.toml
  yews plan config.enc.toml

  # Filter registered mappings with a pattern
  yews plan './configs/*.toml'

  # Print JSON for scripts (errors stay on stderr)
  yews plan --json > plan.json`,
		Args: func(cmd *cobra.Command, args []string) error {
			for _, arg := range args {
				if err := validateTargetArg(arg); err != nil {
					return err
				}
			}
			return nil
		},
		RunE: withConfig(load, func(cmd *cobra.Command, args []string, cfg *config.Config) error {
			return yewsapp.PrintPlan(cmd.OutOrStdout(), cfg, yewsapp.PlanRequest{
				Targets: args,
			}, presentation.PlanPrintOptions{
				JSON:    opts.JSON,
				Verbose: opts.Verbose,
			})
		}),
	}
	addPlanFlags(cmd.Flags(), &opts)
	resolver := newOptionResolver(cmd, &opts)
	cmd.Args = resolver.before(cmd.Args)
	return cmd
}
