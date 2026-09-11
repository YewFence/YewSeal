package cli

import (
	"fmt"
	"strings"

	yewsapp "github.com/YewFence/YewSeal/internal/app"
	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/presentation"
	"github.com/YewFence/YewSeal/internal/project"

	"github.com/spf13/cobra"
)

func initCommand() *cobra.Command {
	var force bool
	var input string
	var output string
	var format string
	var createExample bool
	var skipSOPSConfig bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize project with Age keys and YewSeal config entries",
		Long: `Initialize a project: generate an Age key pair, create .yewseal.toml
with its first config entry, optionally sync .sops.yaml, and update
.gitignore.

Run without flags for interactive mode: YewSeal asks whether to rebuild
an existing configuration, whether to create .sops.yaml, and records
one or more plaintext/encrypted mappings. Passing --input or --output
switches to non-interactive mode for scripts.

Generated files:
  .yewseal.toml  main YewSeal config (recipient registry, defaults,
                 file entries)
  .age/keys.txt  Age private key; must not be committed to version
                 control
  .sops.yaml     SOPS config for direct sops usage; skipped with
                 --skip-sops-config

When only --input is given, the encrypted file name is inferred
(config.toml becomes config.enc.toml; other formats use the matching
.enc.* suffix).

--force rebuilds keys, recipient registry, defaults, file entries, and
the managed .sops.yaml; old aliases and mappings are not preserved, and
existing ciphertext may become undecryptable for the new owner
identity.

Output: stdout stays empty; prompts, warnings, errors, and the
completion summary (mapping count and key file locations) go to stderr,
answers are read from stdin.

See also: "yews encrypt" to encrypt the registered files, "yews decrypt"
to decrypt them. Private key storage and distribution are managed
outside YewSeal.

Documentation: ` + docsGettingStarted + `
Private key handling: ` + docsPrivateKeys,
		Example: `  # Interactive setup
  yews init

  # Non-interactive first mapping
  yews init --input config.toml --output config.enc.toml --format toml

  # Infer the encrypted path from the plaintext file
  yews init --input .dev.vars --format env

  # Rebuild keys and configuration from scratch (existing ciphertext
  # may become undecryptable)
  yews init --force`,
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.NoArgs(cmd, args); err != nil {
				return err
			}
			_, err := yewsapp.ValidateCLIFormatOverride(format)
			return err
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			out := presentation.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), false)
			return project.InitProject(force, input, output, format, createExample, skipSOPSConfig,
				out, out.Prompts(cmd.InOrStdin()))
		},
	}
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Rebuild keys and configuration; existing ciphertext may become undecryptable")
	cmd.Flags().StringVarP(&input, "input", "i", "", "Plaintext file for the first config entry (switches to non-interactive mode)")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Encrypted file for the first config entry (non-interactive mode)")
	cmd.Flags().StringVar(&format, "format", "", "Format override for the first config entry (toml/yaml/json/env/ini/binary)")
	cmd.Flags().BoolVar(&createExample, "create-example", false, "Create an example plaintext file (interactive: for recorded entries; non-interactive: for the first entry)")
	cmd.Flags().BoolVar(&skipSOPSConfig, "skip-sops-config", false, "Skip creating or updating .sops.yaml (non-interactive mode)")
	return cmd
}

func editCommand(load configLoader, keyFile *string) *cobra.Command {
	var file string

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

Output: stdout stays empty; the update result (or "unchanged"),
warnings, and errors go to stderr.

See also: "yews view" to inspect a file read-only, "yews decrypt" to
write the plaintext to disk.

Documentation: ` + docsWorkflows,
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
			if strings.TrimSpace(file) == "" {
				return fmt.Errorf("edit requires exactly one configured target")
			}
			return nil
		},
		RunE: withConfig(load, func(cmd *cobra.Command, args []string, cfg *config.Config) error {
			return yewsapp.EditEncryptedFile(yewsapp.EditRequest{
				Presentation: presentation.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), false),
				Config:       cfg,
				File:         file,
				KeyFile:      *keyFile,
			})
		}),
	}
	cmd.Flags().StringVarP(&file, "file", "f", "", "Encrypted file to edit (must be registered in .yewseal.toml; its configured plaintext path also works)")
	return cmd
}

func viewCommand(load configLoader, keyFile *string) *cobra.Command {
	var verbose bool

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
--verbose detail go to stderr, so the plaintext stays pipeable.

See also: "yews decrypt" to write plaintext files with overwrite
protection, "yews edit" to edit the encrypted file directly.

Documentation: ` + docsTargetSelect,
		Example: `  # Print the decrypted plaintext of a registered file
  yews view config.enc.toml

  # Pipe a registered JSON file into jq
  yews view config.enc.json | jq '.database'

  # Save only the plaintext; detail stays on stderr
  yews view config.enc.toml --verbose > inspected.toml`,
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
			return yewsapp.WriteViewedTarget(cmd.OutOrStdout(), cmd.ErrOrStderr(), cfg, args[0], *keyFile, verbose)
		}),
	}
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output (detail goes to stderr; stdout stays plaintext only)")
	return cmd
}

func diffCommand(load configLoader, keyFile *string) *cobra.Command {
	var color string
	var verbose bool
	var strict bool

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
    matched against either side of registered mappings;
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

Exit codes: in lenient mode, 0 whenever no real error occurs, whether
or not anything was actually compared (0 means neither "equal" nor
"compared"). With --strict, missing inputs do not affect success, but a
comparable mapping skipped for missing identities exits 1. There is no
"differs means failure" switch, so diff is not a CI gate; obtained
diffs are never rolled back.

Output: stdout carries only the diff body (empty when nothing differs);
warnings, per-file skip and failure reasons, and the summary go to
stderr (--verbose adds selection info and per-file completion notes).

See also: "yews encrypt" to re-encrypt changed plaintext, "yews view"
to inspect ciphertext content.

Documentation: ` + docsDecryptResults,
		Example: `  # Compare every registered file in scope
  yews diff

  # Compare one mapping via either side
  yews diff config.toml
  yews diff config.enc.toml

  # Disable color in scripts; diagnostics stay on stderr
  yews diff --color never > changes.diff`,
		Args: func(cmd *cobra.Command, args []string) error {
			for _, arg := range args {
				if err := validateTargetArg(arg); err != nil {
					return err
				}
			}
			_, err := presentation.ResolveDiffColor(color, cmd.OutOrStdout())
			if err != nil {
				return err
			}
			return resolveStrict(cmd, &strict)
		},
		RunE: withConfig(load, func(cmd *cobra.Command, args []string, cfg *config.Config) error {
			_, err := yewsapp.DiffPlaintextAgainstEncryptedTargets(cmd.OutOrStdout(), cmd.ErrOrStderr(), cfg, args, *keyFile, verbose, color, strict)
			return err
		}),
	}
	cmd.Flags().StringVar(&color, "color", "auto", "Colorize diff output (auto/always/never)")
	cmd.Flags().BoolVar(&strict, "strict", false, "Require comparison of mappings with both inputs present (env YEWSEAL_STRICT; --strict=false overrides it)")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output (selection info and per-file completion notes on stderr)")
	return cmd
}
