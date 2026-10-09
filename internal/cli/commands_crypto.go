package cli

import (
	"fmt"
	"strings"

	yewsapp "github.com/YewFence/YewSeal/internal/app"
	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/presentation"

	"github.com/spf13/cobra"
)

func encryptCommand(load configLoader) *cobra.Command {
	opts := encryptOptions{
		Parallel:        1,
		SyncSOPSConfig:  true,
		UpdateGitignore: true,
	}
	var resolver *optionResolver

	cmd := &cobra.Command{
		Use:     "encrypt [command options] [path-or-pattern]...",
		Aliases: []string{"e"},
		Short:   "Encrypt configuration file (supports .toml, .yaml, .yml, .json, .env, .ini, and binary output)",
		Long: `Encrypt registered configuration files with SOPS and Age,
preserving their format. Supported formats: TOML, YAML, JSON, ENV, INI, and binary.

With no arguments, encrypt selects registered plaintext under the current
directory and its subdirectories. Use file paths, directories, or patterns
to select targets; directories and patterns match plaintext paths.

Recipients come from .yewseal.toml. An empty recipient set, an unknown
alias, or conflicting group authorization fails the batch before any
ciphertext is written.

When ciphertext already exists, encrypt uses an available private identity
to verify and update it in place: unchanged files remain byte-identical,
unchanged values retain their ciphertext, and recipient-only changes only
rewrap the existing data key. If no identity is available, or none matches a
specific file, encrypt warns and replaces that ciphertext from the current
plaintext. --force always performs this fresh encryption and rotates the data
key without reading the old ciphertext. Missing plaintext is reported and
skipped.

--output changes the destination of a single file target, preserving its
configured format. Declare non-standard extensions with "format" or
"format_rules" in .yewseal.toml.

Before encryption, --update-gitignore (enabled by default) adds all configured
plaintext entries using the managed-file rules. After encryption,
--sync-sops-config (enabled by default) rewrites .sops.yaml using those rules.
A synchronization failure leaves completed ciphertext work in place and
makes the command fail.

Exit codes: 0 on success (skips without real errors allowed, including
a fully skipped batch); 1 when any file fails, .sops.yaml synchronization
fails, or output delivery fails. Exit 1 does not imply that no ciphertext
was written; 2 for calling errors: invalid arguments, a missing or invalid
.yewseal.toml, selection or authorization failure, or an unusable
identity source (an unreadable explicit file or a failed key command).

Output: ciphertext goes to files and stdout stays empty; warnings,
per-file failure reasons, and the summary go to stderr (--verbose adds
selection info and per-file success). --json replaces stdout with the
batch report (summary and per-file outcomes); stderr diagnostics stay
unchanged, and when the run never starts (a calling error, exit 2)
stdout stays empty.

To encrypt an unregistered file ad hoc without a project config, use
the SOPS CLI directly.

See also: "yews plan" to inspect mappings and authorization, "yews diff"
to compare plaintext with stored ciphertext.

Target selection: ` + docsTargetSelect + `
Managed-file rules: ` + docsManagedFiles,
		Example: `  # Encrypt registered plaintext under the current directory
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
				Presentation:    presentation.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), opts.Verbose),
				KeyFile:         opts.KeyFile,
				Output:          opts.Output,
				OutputSet:       resolver.IsSet("output"),
				Targets:         args,
				Parallel:        opts.Parallel,
				Force:           opts.Force,
				JSON:            opts.JSON,
				UpdateGitignore: opts.UpdateGitignore,
				SyncSOPSConfig:  opts.SyncSOPSConfig,
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
		Parallel:        1,
		UpdateGitignore: true,
	}
	var resolver *optionResolver

	cmd := &cobra.Command{
		Use:     "decrypt [command options] [path-or-pattern]...",
		Aliases: []string{"d"},
		Short:   "Decrypt registered files in place or deliver a mirrored plaintext tree",
		Long: `Decrypt registered SOPS-encrypted files to their configured plaintext
paths, or into a caller-supplied mirrored tree with --output. The
format comes from the project config or the registered file path;
runtime format overrides and cross-format conversion are not
supported.

With no arguments, decrypt selects registered ciphertext under the current
directory and its subdirectories. Use file paths, directories, or patterns
to select targets; directories and patterns match encrypted paths. Selectors
choose what to decrypt; --output chooses where to deliver the same selection.

--output DIR mirrors plaintext paths relative to the repository root (or the
config discovery root outside a repository), directly under DIR. A single
selected file keeps its project-relative path, too. Selected plaintext paths
outside the project root and conflicting destinations are rejected before any
write. Every destination is a regular file at the logical registered path.
DIR resolves relative to the calling directory and must already exist as a
real, empty directory — a file, symlink, or any existing entry, including a
hidden one, is rejected. New subdirectories use 0700 and plaintext files use
0600; root permissions stay unchanged. Use a trusted, one-use directory.
--output is mutually exclusive with --force=true and --inplace=true. Delivery
writes the tree without consulting .gitignore or .sops.yaml; an output root
inside the repository is allowed, and the caller owns accidental commit risk
and output-tree cleanup, including after failure.

Mappings with plaintext_mode=delivery require --output DIR or explicit
--inplace consent; otherwise the whole selection is rejected before writing.
--inplace allows these mappings at their configured plaintext paths and is
mutually exclusive with --output. The default plaintext_mode is inplace.
Delivery classification is an accident-prevention default; use separate
recipients and identities for access control. view and encrypt keep their
existing behavior under this classification.

The config still governs plaintext/ciphertext paths and formats, but the
recipients actually used for decryption come from the ciphertext's SOPS
metadata: when an alias referenced by the current config no longer
exists, decrypt warns on stderr and continues with the identity bundle.

Overwrite protection: an existing plaintext file whose content differs
from the decryption result is not overwritten unless --force is set.
For configured-path writes, --update-gitignore (enabled by default) adds plaintext
entries for selected mappings, or all configured mappings when no target is
given, using the managed-file rules.

Exit codes: by default, files are skipped when no Age identity is
available (outcome "no-identity") or when the available identities do
not match ("no-matching-identity"); even a fully skipped batch exits 0,
so lenient callers can treat unavailable decryption access as a
degradable condition. Real errors (a missing or corrupted ciphertext,
read or write failures, overwrite conflicts) and output delivery
failures exit 1. With --strict, any skip also exits 1, but remaining
files are still processed and successful results are kept. --output does not
implicitly enable strict: use --strict for complete runtime snapshots. Failures
can leave partial plaintext trees; successful writes are never rolled back.
--strict=false overrides YEWSEAL_DECRYPT_STRICT. Calling errors
(invalid arguments, a missing or invalid .yewseal.toml, selection
failure, an unreadable explicit key file, or a failed key command) exit 2.

TOML output uses normalized formatting: single-quoted literal strings,
preserved comments, and equivalent content with possibly different layout.

Output: plaintext goes to files and stdout stays empty; warnings,
per-file skip and failure reasons, and the summary go to stderr
(--verbose adds selection info and per-file success). --json replaces
stdout with the batch report (summary and per-file outcomes); stderr
diagnostics stay unchanged, and when the run never starts (a calling
error, exit 2) stdout stays empty.

For unregistered files, see the SOPS interoperability guide.

See also: "yews plan" to inspect mappings and authorization, "yews view"
to print plaintext, "yews diff" to compare it with stored ciphertext.

Target selection: ` + docsTargetSelect + `
Managed-file rules: ` + docsManagedFiles + `
SOPS interoperability: ` + docsSOPS + `
Result classification and exit codes: ` + docsDecryptResults + `
Plaintext delivery and caller-owned lifecycle: ` + docsPlaintextDelivery,
		Example: `  # Decrypt registered ciphertext under the current directory
  yews decrypt

  # Decrypt one registered encrypted file to its configured plaintext
  yews decrypt config.enc.toml

  # Deliver a complete mirrored tree into a caller-owned empty directory
  delivery=$(mktemp -d)
  yews decrypt --strict --output "$delivery"

  # Explicitly permit delivery mappings at configured plaintext paths
  yews decrypt --inplace

  # Select registered ciphertext under ./configs with a pattern
  yews decrypt './configs/*.enc.toml'

  # Overwrite a plaintext file that differs from the decrypted content
  yews decrypt config.enc.toml --force

  # Fail on any skipped file (or set YEWSEAL_DECRYPT_STRICT=true)
  yews decrypt --strict

  # Print the batch report for scripts (diagnostics stay on stderr)
  yews decrypt --json > report.json`,
		Args: func(cmd *cobra.Command, args []string) error {
			if err := validateBatchArgs(args, opts.Parallel, false); err != nil {
				return err
			}
			if resolver.IsSet("output") {
				if strings.TrimSpace(opts.Output) == "" {
					return fmt.Errorf("--output requires a directory")
				}
				if opts.Force || opts.Inplace {
					return fmt.Errorf("--output conflicts with --force=true or --inplace=true")
				}
			}
			return nil
		},
		RunE: withConfig(load, func(cmd *cobra.Command, args []string, cfg *config.Config) error {
			return yewsapp.DecryptFiles(cfg, yewsapp.DecryptRequest{
				Presentation:    presentation.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), opts.Verbose),
				KeyFile:         opts.KeyFile,
				OutputDir:       opts.Output,
				Inplace:         opts.Inplace,
				Targets:         args,
				Parallel:        opts.Parallel,
				Force:           opts.Force,
				Strict:          opts.Strict,
				JSON:            opts.JSON,
				UpdateGitignore: opts.UpdateGitignore,
			})
		}),
	}
	addDecryptFlags(cmd.Flags(), &opts)
	resolver = newOptionResolver(cmd, &opts)
	cmd.Args = resolver.before(cmd.Args)
	return cmd
}

func cleanCommand(load configLoader) *cobra.Command {
	opts := cleanOptions{}
	var resolver *optionResolver

	cmd := &cobra.Command{
		Use:     "clean [command options] [path-or-pattern]...",
		Aliases: []string{"c"},
		Short:   "Safely remove registered local plaintext files",
		Long: `Remove registered local plaintext, by default only after proving that its
ciphertext can be decrypted with the current Age identities. clean never
encrypts files, changes recipients, repairs project metadata, removes
directories, or displays plaintext or diff content.

With no arguments, clean selects registered plaintext under the current
directory and its subdirectories. Use file paths, directories, or patterns
to select targets; directories and patterns match plaintext paths.
A missing ciphertext makes cleanup fail; a missing plaintext is reported
as already absent.

Matching decrypted bytes are removed automatically. Different bytes prompt
once per file with No as the safe default. --skip-different keeps every
difference without reading stdin; --remove-different removes differences
without reading stdin. These flags are mutually exclusive. Every existing
plaintext must be decryptable before removal and requires a usable identity.
If ciphertext changes after confirmation, removal fails and plaintext is
retained. Missing
or damaged ciphertext, no usable or matching identity, non-regular plaintext,
I/O errors, and a failed final recheck mark the item FAILED and retain it;
neither policy bypasses these checks.

--force is a separate, dangerous mode: it removes every selected plaintext
that currently exists without reading ciphertext, resolving identities,
comparing content, or prompting. It still follows the configured plaintext
path through symlinks, deletes only the same final regular file it inspected,
continues after per-file failures, and never removes directories. --force is
CLI-only, has no short form or environment variable, and is mutually exclusive
with --remove-different and --skip-different. Removed plaintext may not be
recoverable.

Plaintext paths may contain symlinks. clean removes the final regular-file
target and leaves links in place. Broken links are already absent. A changed
target or file type makes removal fail; in normal mode, changed content also
makes removal fail. Completed removals are not rolled back when another item
later fails.

Output: stdout is always empty. Prompts, warnings, REMOVED/RETAINED/FAILED
results, and the final summary go to stderr; --verbose also prints selection
details and ALREADY ABSENT results. clean does not update .gitignore or
.sops.yaml.

Exit codes: 0 when the selected policy completes, including explicitly retained
differences; 1 when any item, prompt, or output channel fails (earlier
removals remain); 2 for calling errors: invalid arguments, config, selection,
an unreadable explicit key file, or a failed key command. An empty identity
set instead makes each plaintext that needs verification fail safely with exit
1; no file is removed. --force does not require an identity.

See also: "yews diff" to inspect a difference before deciding and "yews
encrypt" to save local changes before cleaning.

Documentation: ` + docsPlaintextCleanup + `
Target selection: ` + docsTargetSelect,
		Example: `  # Clean registered plaintext under the current directory
  yews clean

  # Clean one mapping selected by its plaintext path
  yews clean config.toml

  # Select registered plaintext paths with a pattern
  yews clean './configs/*.toml'

  # Keep all differences without prompting
  yews clean --skip-different

  # Irreversibly remove differences after successful decryption
  yews clean --remove-different

  # DANGEROUS: remove every selected plaintext without recovery checks
  yews clean --force

  # Inspect one difference before cleaning it
  yews diff -- config.toml`,
		Args: func(_ *cobra.Command, args []string) error {
			enabledStrategies := 0
			for _, enabled := range []bool{opts.Force, opts.RemoveDifferent, opts.SkipDifferent} {
				if enabled {
					enabledStrategies++
				}
			}
			if enabledStrategies > 1 {
				return fmt.Errorf("--force, --remove-different, and --skip-different are mutually exclusive")
			}
			for _, arg := range args {
				if err := validateTargetArg(arg); err != nil {
					return err
				}
			}
			return nil
		},
		RunE: withConfig(load, func(cmd *cobra.Command, args []string, cfg *config.Config) error {
			return yewsapp.CleanFiles(cfg, yewsapp.CleanRequest{
				Presentation:    presentation.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), opts.Verbose),
				Input:           cmd.InOrStdin(),
				KeyFile:         opts.KeyFile,
				Targets:         args,
				Force:           opts.Force,
				RemoveDifferent: opts.RemoveDifferent,
				SkipDifferent:   opts.SkipDifferent,
			})
		}),
	}
	addCleanFlags(cmd.Flags(), &opts)
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

With no arguments, plan selects mappings with either path under the current
directory and its subdirectories. Use file paths, directories, or patterns
to select targets; directories and patterns match either path.

plan applies the same strict authorization semantics as encrypt to
every mapping resolved from the loaded config, including unselected
ones: unknown aliases, empty recipient sets, and group conflicts fail
the run. The report shows recipient aliases; --source additionally shows
the origins of each path, format, authorization set, and registry alias.
Use --json to audit canonical recipients, selection reasons, and all
provenance fields. Historical ciphertext recipients and decryption access
are not checked.

Output: stdout shows a config count and selection scope followed by a
Plaintext/Encrypted/Format/Aliases/PlaintextMode table. --source replaces
the table with one describe block per mapping and field-level origins;
--verbose also lists loaded config files. --json prints only the full
JSON report and takes precedence over --source; errors go to stderr and
never mix into the report.

Exit codes: 0 on success; calling errors (invalid patterns, a missing
or invalid .yewseal.toml, or authorization conflicts) exit 2.

See also: "yews verify" to check ciphertext and decryption access,
"yews encrypt" and "yews decrypt" to process the configured files.

Documentation: ` + docsConfiguration + `
Target selection: ` + docsTargetSelect,
		Example: `  # Inspect registered mappings under the current directory
  yews plan

  # Either side of a mapping selects it
  yews plan config.toml
  yews plan config.enc.toml

  # Filter registered mappings with a pattern
  yews plan './configs/*.toml'

  # Trace where each path, format, and authorization set came from
  yews plan --source

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
				Source:  opts.Source,
			})
		}),
	}
	addPlanFlags(cmd.Flags(), &opts)
	resolver := newOptionResolver(cmd, &opts)
	cmd.Args = resolver.before(cmd.Args)
	return cmd
}
