package cli

import (
	"fmt"
	"strings"

	"github.com/YewFence/YewSeal/internal/agekey"
	yewsapp "github.com/YewFence/YewSeal/internal/app"
	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/errx"
	"github.com/YewFence/YewSeal/internal/presentation"
	"github.com/YewFence/YewSeal/internal/project"

	"github.com/spf13/cobra"
)

func initCommand() *cobra.Command {
	opts := initOptions{SyncSOPSConfig: true}
	var resolver *optionResolver

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize project with Age keys and YewSeal config entries",
		Long: `Initialize a project: generate an Age key pair, create .yewseal.toml
with its first config entry, optionally sync .sops.yaml, and update
.gitignore.

Run without flags for interactive mode: YewSeal asks whether to rebuild
an existing configuration, whether to create .sops.yaml, and records
one or more plaintext/encrypted mappings. Passing --input or --output
switches to non-interactive mode for scripts. An explicit
--sync-sops-config value skips its interactive question.

Generated files:
  .yewseal.toml  main YewSeal config (recipient registry, defaults,
                 file entries)
  .age/keys.txt  Age private key; must not be committed to version
                 control
  .sops.yaml     SOPS config for direct sops usage; skipped with
                 --sync-sops-config=false

When only --input is given, the encrypted file name is inferred
(config.toml becomes config.enc.toml; other formats use the matching
.enc.* suffix).

--force rebuilds keys, recipient registry, defaults, file entries, and,
when synchronization is enabled, the managed .sops.yaml; old aliases and
mappings are not preserved, and existing ciphertext may become undecryptable
for the new owner identity. With --sync-sops-config=false, initialization
leaves any existing .sops.yaml untouched. A failure to create .sops.yaml
makes initialization fail.

Output: stdout stays empty; prompts, warnings, errors, and the
completion summary (mapping count and key file locations) go to stderr,
answers are read from stdin. --json additionally prints the
initialization report (mappings, key file, .sops.yaml outcome) on
stdout, which suits non-interactive scripting.

Exit codes: 0 on success (including keeping an existing configuration
after declining the overwrite prompt); 1 when writing project files
fails; 2 for calling errors (invalid arguments or an invalid --format).

See also: "yews encrypt" to encrypt the registered files, "yews decrypt"
to decrypt them. Private key storage and distribution are managed
outside YewSeal.

Documentation: ` + docsTutorial + `
Private key handling: ` + docsPrivateKeys,
		Example: `  # Interactive setup
  yews init

  # Non-interactive first mapping
  yews init --input config.toml --output config.enc.toml --format toml

  # Infer the encrypted path from the plaintext file
  yews init --input .dev.vars --format env

  # Rebuild keys and configuration from scratch (existing ciphertext
  # may become undecryptable)
  yews init --force

  # Report the initialization result as JSON for scripts
  yews init --input config.toml --json`,
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.NoArgs(cmd, args); err != nil {
				return err
			}
			_, err := yewsapp.ValidateCLIFormatOverride(opts.Format)
			return err
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			out := presentation.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), false)
			return project.InitProject(project.InitOptions{
				Force:             opts.Force,
				InputFile:         opts.Input,
				OutputFile:        opts.Output,
				FormatOverride:    opts.Format,
				CreateExample:     opts.CreateExample,
				CreateExampleSet:  resolver.IsSet("create-example"),
				SyncSOPSConfig:    opts.SyncSOPSConfig,
				SyncSOPSConfigSet: resolver.IsSet("sync-sops-config"),
				JSON:              opts.JSON,
			}, out, out.Prompts(cmd.InOrStdin()))
		},
	}
	cmd.Flags().BoolVarP(&opts.Force, "force", "f", false, "Rebuild keys and configuration; existing ciphertext may become undecryptable")
	cmd.Flags().StringVarP(&opts.Input, "input", "i", "", "Plaintext file for the first config entry (switches to non-interactive mode)")
	cmd.Flags().StringVarP(&opts.Output, "output", "o", "", "Encrypted file for the first config entry (non-interactive mode)")
	cmd.Flags().StringVar(&opts.Format, "format", "", "Format override for the first config entry (toml/yaml/json/env/ini/binary)")
	cmd.Flags().BoolVar(&opts.CreateExample, "create-example", false, "Create an example plaintext file (interactive: for recorded entries; non-interactive: for the first entry)")
	cmd.Flags().BoolVar(&opts.SyncSOPSConfig, "sync-sops-config", opts.SyncSOPSConfig, "Create or update .sops.yaml; explicit true or false skips the interactive prompt")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Print the initialization report as JSON on stdout (prompts and diagnostics stay on stderr)")
	resolver = newOptionResolver(cmd, &opts)
	cmd.Args = resolver.before(cmd.Args)
	return cmd
}

func identitiesCommand(load configLoader) *cobra.Command {
	opts := identitiesOptions{}
	var resolver *optionResolver

	cmd := &cobra.Command{
		Use:   "identities",
		Short: "List the Age identities YewSeal would decrypt with, their winning source, and registry aliases",
		Long: `List the Age identities YewSeal would decrypt with: which source won the
resolution chain, which present sources it shadowed, and every identity in
the winning source with its derived public key and registry alias.

Identity resolution is first-win, never merged: --key-file (env
YEWSEAL_KEY_FILE), then YEWSEAL_AGE_IDENTITIES, SOPS_AGE_KEY,
SOPS_AGE_KEY_FILE, SOPS_AGE_KEY_CMD, then .age/keys.txt in the current
directory. Only the first source that yields an identity applies, and
everything below it is not even read; shadowed lists the sources that were
present but skipped, as file:<path> or env:<NAME>.

The command requires a .yewseal.toml like every other command beyond
version, help, and completion: each derived public key is looked up in
[recipients.registry], and an unregistered public key warns on stderr
and carries a warning field in --json output.

By default no secret key material is printed. --reveal includes it:
--json adds a secret field per identity, plain output adds a Secret
column. The values then flow to stdout — mind terminal scrollback and CI
logs (GitHub Actions only redacts exact repository-secret matches); pipe
into a file or a consuming process instead of logging.

Output: plain mode prints a Source/Shadowed header plus an Alias/Public
key table (plus Secret with --reveal) on stdout; --json prints the report
on stdout; warnings go to stderr either way.

Exit codes: 0 on success; 2 when no identity source yields an identity,
the winning source is unreadable or invalid, or .yewseal.toml is missing
or invalid.

See also: "yews plan" to preview file mappings and authorization on the
other side of the pipeline.

Documentation: ` + docsReadingPrivateKeys,
		Example: `  # Which identities does this machine decrypt with, and from where
  yews identities

  # Audit which configured sources a --key-file shadows
  yews identities --key-file .age/keys.txt

  # Machine-readable report including secret keys (CI: pipe, do not log)
  yews identities --json --reveal > bundle.json

  # Inspect a specific candidate key file before adopting it
  yews identities --key-file /path/to/new-key.txt`,
		Args: cobra.NoArgs,
		RunE: withConfig(load, func(cmd *cobra.Command, args []string, cfg *config.Config) error {
			sources, err := agekey.ResolveIdentitySources(opts.KeyFile)
			if err != nil {
				return errx.Usage(err)
			}
			out := presentation.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), false)
			return out.Identities(buildIdentitiesReport(sources, cfg), presentation.IdentitiesPrintOptions{
				JSON:   opts.JSON,
				Reveal: opts.Reveal,
			})
		}),
	}
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Print the identity report as JSON on stdout (warnings stay on stderr)")
	cmd.Flags().BoolVar(&opts.Reveal, "reveal", false, "Include each identity's secret key (JSON adds a secret field; plain output adds a Secret column)")
	resolver = newOptionResolver(cmd, &opts)
	cmd.Args = resolver.before(cmd.Args)
	return cmd
}

func buildIdentitiesReport(sources agekey.IdentitySources, cfg *config.Config) presentation.IdentitiesReport {
	report := presentation.IdentitiesReport{Source: sources.Source, Shadowed: sources.Shadowed, Warnings: sources.Warnings}
	byPublicKey := make(map[string]string, len(cfg.Recipients.Registry))
	for alias, publicKey := range cfg.Recipients.Registry {
		byPublicKey[strings.TrimSpace(publicKey)] = alias
	}
	for _, identity := range sources.Identities {
		entry := presentation.IdentityEntry{PublicKey: identity.PublicKey, Secret: identity.Secret}
		if alias, ok := byPublicKey[identity.PublicKey]; ok {
			entry.Alias = alias
		} else {
			entry.Warning = "public key not registered in [recipients.registry]"
		}
		report.Identities = append(report.Identities, entry)
	}
	return report
}

func editCommand(load configLoader) *cobra.Command {
	opts := editOptions{}

	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Edit encrypted configuration file using SOPS",
		Long: `Edit an encrypted file in place: decrypt it to a temporary file,
open an editor, then re-encrypt and write the result back on save.

The target must be registered in .yewseal.toml; either the encrypted
path or its configured plaintext path locates the mapping. There is no
default target and no --editor flag.

The editor is taken from VISUAL, then EDITOR; when both are unset or
empty, notepad is used on Windows and vi elsewhere. The variable may
carry the executable plus arguments using a restricted syntax, not a
full shell command:
  - spaces, tabs, and newlines separate arguments; single or double
    quotes keep a path or argument containing spaces together and
    adjacent pieces merge; empty arguments ("") are kept;
  - inside single quotes everything is literal; inside double quotes
    only \" and \\ lose their backslash; quote Windows paths;
  - outside quotes a backslash escapes the next character; unclosed
    quotes, a trailing backslash, NUL, invalid UTF-8, and unescaped
    | & ; < > ( ) $ or backticks are errors;
  - no environment variable, command substitution, glob, or ~ expansion
    happens and # is not a comment.
YewSeal spawns the editor process directly (no shell) and passes the
temporary file path as the final, separate argument.

The editor must exit only after the file is saved and closed (for
example "code --wait"), otherwise YewSeal re-encrypts before editing
finishes.

Exit codes: 0 on success (changed or unchanged); 1 when editing or
re-encryption fails; 2 for calling errors (no target, an unregistered
file, a missing or invalid .yewseal.toml, or an unusable identity
source).

Output: stdout stays empty; the update result (or "unchanged"),
warnings, and errors go to stderr.

See also: "yews view" to inspect a file read-only, "yews decrypt" to
write the plaintext to disk.

Documentation: ` + docsTutorial,
		Example: `  # Edit a registered encrypted file
  yews edit -f config.enc.toml

  # The configured plaintext path locates the same mapping
  yews edit -f config.toml

  # Pick the editor per invocation
  VISUAL="code --wait" yews edit -f config.enc.toml
  VISUAL=vim yews edit -f config.enc.toml`,
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.NoArgs(cmd, args); err != nil {
				return err
			}
			if strings.TrimSpace(opts.File) == "" {
				return fmt.Errorf("edit requires exactly one configured target")
			}
			return nil
		},
		RunE: withConfig(load, func(cmd *cobra.Command, args []string, cfg *config.Config) error {
			return yewsapp.EditEncryptedFile(yewsapp.EditRequest{
				Presentation: presentation.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), false),
				Config:       cfg,
				File:         opts.File,
				KeyFile:      opts.KeyFile,
			})
		}),
	}
	cmd.Flags().StringVarP(&opts.File, "file", "f", "", "Encrypted file to edit (must be registered in .yewseal.toml; its configured plaintext path also works)")
	resolver := newOptionResolver(cmd, &opts)
	cmd.Args = resolver.before(cmd.Args)
	return cmd
}

func viewCommand(load configLoader) *cobra.Command {
	opts := viewOptions{}

	cmd := &cobra.Command{
		Use:   "view [command options] <target>",
		Short: "Print decrypted plaintext to standard output without writing files",
		Long: `Print the decrypted plaintext of one registered encrypted file to
standard output without writing any plaintext file.

The target must match the plaintext or encrypted path of a registered
file. The format comes from the project config or the registered path;
view outputs only the file's own format and performs no cross-format
conversion (pipe the output into a converter when needed).

view shares decrypt's full selection and historical-decrypt semantics:
it warns and continues when the configured alias no longer exists, and
decrypts according to the ciphertext metadata and the current identity
bundle. It stays single-target: it fails without emitting empty
plaintext and never touches .gitignore or .sops.yaml.

Output: stdout carries only the plaintext; warnings, errors, and
--verbose detail go to stderr, so the plaintext stays pipeable. --json
wraps the plaintext in a JSON envelope (path, format, encoding,
content); binary formats encode the content as base64.

Exit codes: 0 on success; 1 when decryption or output delivery fails;
2 for calling errors (wrong argument count, an unregistered target, a
missing or invalid .yewseal.toml, or an unusable identity source).

See also: "yews decrypt" to write plaintext files with overwrite
protection, "yews edit" to edit the encrypted file directly.

Documentation: ` + docsTargetSelect,
		Example: `  # Print the decrypted plaintext of a registered file
  yews view config.enc.toml

  # Pipe a registered JSON file into jq
  yews view config.enc.json | jq '.database'

  # Save only the plaintext; detail stays on stderr
  yews view config.enc.toml --verbose > inspected.toml

  # Wrap the plaintext in a JSON envelope for scripts
  yews view config.enc.toml --json`,
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.ExactArgs(1)(cmd, args); err != nil {
				return err
			}
			if strings.TrimSpace(args[0]) == "" {
				return fmt.Errorf("view requires exactly one target")
			}
			return nil
		},
		RunE: withConfig(load, func(cmd *cobra.Command, args []string, cfg *config.Config) error {
			return yewsapp.ViewTarget(cmd.OutOrStdout(), cmd.ErrOrStderr(), cfg, yewsapp.ViewRequest{
				Target:  args[0],
				KeyFile: opts.KeyFile,
				Verbose: opts.Verbose,
				JSON:    opts.JSON,
			})
		}),
	}
	cmd.Flags().BoolVarP(&opts.Verbose, "verbose", "v", false, "Enable verbose output (detail goes to stderr; stdout stays plaintext only)")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Print the decrypted content as a JSON envelope on stdout (base64 for binary formats)")
	resolver := newOptionResolver(cmd, &opts)
	cmd.Args = resolver.before(cmd.Args)
	return cmd
}

func diffCommand(load configLoader) *cobra.Command {
	opts := diffOptions{Color: "auto"}

	cmd := &cobra.Command{
		Use:   "diff [path-or-pattern]...",
		Short: "Compare plaintext file with decrypted encrypted file",
		Long: `Compare local plaintext files with the decrypted content of their
registered ciphertext, to preview pending changes during development.
Showing differences is never itself a failure.

Target selection (no argument: mappings by plaintext path within the
current directory scope):
  - a discovered mapping's plaintext or encrypted path selects it;
  - an existing directory filters registered mappings by their plaintext
    side (no rescanning by the target directory);
  - arguments containing *, ?, and similar metacharacters are patterns
    matched against registered plaintext paths;
  - multiple arguments take the union; any argument matching nothing is
    an error.
Groups are discovered from the plaintext side; group entries with only
a ciphertext never enter the candidate set. Selecting nothing at all is
an error, which differs from selecting files that are all skipped.

A mapping is skipped with the reason reported on stderr when either
side is missing (no new/deleted patch is emitted and it does not count
as equal) or when no identity matches. Detected ciphertext corruption,
permission errors, and other read/write failures are real errors, but a
single file's error does not stop the comparison of other files. Once a
selection succeeds, the identity source is resolved exactly once: an
invalid --key-file fails even when every mapping ends up skipped.

The format and mapping come from the config; a stale recipient alias
warns and continues, and decryption follows the historical ciphertext
metadata rather than the current-config authorization.

Exit codes: 0 whenever no real error occurs, whether or not anything
was actually compared (0 means neither "equal" nor "compared"); 1 when
any comparison fails or output delivery fails. There is no strict mode
and no "differs means failure" switch, so diff is not a CI gate;
obtained diffs are never rolled back. Calling errors (invalid patterns,
a missing or invalid .yewseal.toml, selection failure, or an unusable
identity source) exit 2.

Output: stdout carries only the diff body (empty when nothing differs);
warnings, per-file skip and failure reasons, and the summary go to
stderr (--verbose adds selection info and per-file completion notes).
--json replaces the streamed diff body with the comparison report
(per-file status with embedded diff bodies).

See also: "yews encrypt" to re-encrypt changed plaintext, "yews view"
to inspect ciphertext content.

Documentation: ` + docsDecryptResults,
		Example: `  # Compare every registered file in scope
  yews diff

  # Compare one mapping via either side
  yews diff config.toml
  yews diff config.enc.toml

  # Disable color in scripts; diagnostics stay on stderr
  yews diff --color never > changes.diff

  # Print the comparison report for scripts
  yews diff --json > report.json`,
		Args: func(cmd *cobra.Command, args []string) error {
			for _, arg := range args {
				if err := validateTargetArg(arg); err != nil {
					return err
				}
			}
			_, err := presentation.ResolveDiffColor(opts.Color, cmd.OutOrStdout())
			if err != nil {
				return err
			}
			return nil
		},
		RunE: withConfig(load, func(cmd *cobra.Command, args []string, cfg *config.Config) error {
			_, err := yewsapp.DiffTargets(cmd.OutOrStdout(), cmd.ErrOrStderr(), cfg, yewsapp.DiffRequest{
				Targets:   args,
				KeyFile:   opts.KeyFile,
				Verbose:   opts.Verbose,
				ColorMode: opts.Color,
				JSON:      opts.JSON,
			})
			return err
		}),
	}
	cmd.Flags().StringVar(&opts.Color, "color", opts.Color, "Colorize diff output (auto/always/never)")
	cmd.Flags().BoolVarP(&opts.Verbose, "verbose", "v", false, "Enable verbose output (selection info and per-file completion notes on stderr)")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Print the comparison report as JSON on stdout (per-file status with embedded diff bodies)")
	resolver := newOptionResolver(cmd, &opts)
	cmd.Args = resolver.before(cmd.Args)
	return cmd
}
