---
title: yews encrypt
---

Encrypt configuration file (supports .toml, .yaml, .yml, .json, .env, .ini, and binary output)

## Synopsis

Encrypt registered configuration files with SOPS and Age,
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
selection info, scan counts, and per-file success). --json replaces stdout with the
batch report (summary and per-file outcomes); stderr diagnostics stay
unchanged, and when the run never starts (a calling error, exit 2)
stdout stays empty.

To encrypt an unregistered file ad hoc without a project config, use
the SOPS CLI directly.

See also: "yews plan" to inspect mappings and authorization, "yews diff"
to compare plaintext with stored ciphertext.

Target selection: https://yewfence.github.io/YewSeal/guide/target-selection
Managed-file rules: https://yewfence.github.io/YewSeal/guide/configuration#managed-files
Scan exclusions: https://yewfence.github.io/YewSeal/guide/configuration#scan-exclusions

```
yews encrypt [command options] [path-or-pattern]... [flags]
```

## Examples

```
  # Encrypt registered plaintext under the current directory
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
  yews encrypt --json > report.json
```

## Options

```
  -f, --force              Freshly encrypt every existing plaintext and rotate its data key (env YEWSEAL_ENCRYPT_FORCE)
  -h, --help               help for encrypt
      --json               Print the batch report as JSON on stdout (diagnostics stay on stderr) (env YEWSEAL_ENCRYPT_JSON)
  -o, --output string      Output encrypted file for a single file target (env YEWSEAL_ENCRYPT_OUTPUT)
  -P, --parallel int       Number of parallel workers for batch mode (minimum 1) (env YEWSEAL_ENCRYPT_PARALLEL) (default 1)
      --sync-sops-config   Rewrite .sops.yaml after encryption using the managed-file rules (env YEWSEAL_SYNC_SOPS_CONFIG) (default true)
      --update-gitignore   Add plaintext and default key entries to .gitignore; false leaves it untouched (env YEWSEAL_UPDATE_GITIGNORE) (default true)
  -v, --verbose            Enable verbose output (scan counts, selection info and per-file success on stderr) (env YEWSEAL_ENCRYPT_VERBOSE)
```

## Options inherited from parent commands

```
  -k, --key-file string   Path to the Age private key file (fallback: YEWSEAL_AGE_IDENTITIES, SOPS_AGE_KEY, SOPS_AGE_KEY_FILE, YEWSEAL_AGE_KEY_CMD, SOPS_AGE_KEY_CMD, then .age/keys.txt in the current directory) (env YEWSEAL_KEY_FILE)
```

## SEE ALSO

* [yews](/references/yews/)	 - YewSeal - Encrypt/decrypt configuration files using SOPS and Age (supports TOML, YAML, JSON, ENV, INI, and binary files)
