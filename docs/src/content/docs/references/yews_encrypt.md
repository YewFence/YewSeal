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

--source DIR reads plaintext from a caller-supplied mirrored tree instead
of registered plaintext paths: every selected mapping's plaintext comes
from DIR / relative(project-root, registered-plaintext-path), the exact
inverse of decrypt --output's projection, which makes
"decrypt --output DIR, edit, encrypt --source DIR" a closed round trip.
Ciphertext still goes to registered paths, and a file missing from the
tree is reported and skipped (outcome "missing plaintext") without
falling back to the registered path. DIR resolves relative to the
calling directory and must already exist as a real directory. The caller
owns the tree's lifetime and cleanup.

When ciphertext already exists, encrypt uses an available private identity
to verify and update it in place: unchanged files remain byte-identical,
unchanged values retain their ciphertext, and recipient-only changes only
rewrap the existing data key. If no identity is available, or none matches a
specific file, encrypt warns and replaces that ciphertext from the current
plaintext. --force always performs this fresh encryption and rotates the data
key without reading the old ciphertext. Missing plaintext is reported and
skipped.

Before encryption, --update-gitignore (enabled by default) rebuilds each
loaded configuration's .gitignore managed block from its complete plaintext
mapping set, removing obsolete entries and preserving content outside the block.
Malformed or duplicate block markers fail the command without changing that file.
See the managed-file rules for marker syntax and path escaping. After encryption,
--sync-sops-config (enabled by default) rewrites .sops.yaml in each loaded
configuration's directory. Entries belong to the mapping's source configuration,
regardless of the calling directory or selected targets.
A synchronization failure leaves completed ciphertext work in place and
makes the command fail.

Exit codes: 0 on success (skips without real errors allowed, including
a fully skipped batch); 1 when any file fails, .sops.yaml synchronization
fails, or output delivery fails. Exit 1 does not imply that no ciphertext
was written; 2 for calling errors: invalid arguments, a missing or invalid
.yewseal.toml, selection or authorization failure, an unusable
--source directory, or an unusable identity source (an unreadable
explicit file or a failed key command).

Output: ciphertext goes to files and stdout stays empty; warnings,
per-file encryption results, skip and failure reasons, and the summary go to
stderr. Unchanged files are counted in the summary; --verbose also lists them
individually and adds selection info and scan counts. --json replaces stdout with the
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

  # Close the delivery round trip: edit a mirrored tree and encrypt it back
  delivery=$(mktemp -d)
  yews decrypt --strict --output "$delivery"
  # ... edit "$delivery/config.toml" ...
  yews encrypt --source "$delivery"

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
  -P, --parallel int       Number of parallel workers for batch mode (minimum 1) (env YEWSEAL_ENCRYPT_PARALLEL) (default 1)
      --source string      Read plaintext from the project-relative mirrored tree under this existing, real directory instead of registered plaintext paths; ciphertext still goes to registered paths (env YEWSEAL_ENCRYPT_SOURCE)
      --sync-sops-config   Rewrite .sops.yaml after encryption using the managed-file rules (env YEWSEAL_SYNC_SOPS_CONFIG) (default true)
      --update-gitignore   Rebuild .gitignore managed blocks from all configured plaintext paths and the default key path; false leaves files untouched (env YEWSEAL_UPDATE_GITIGNORE) (default true)
  -v, --verbose            Enable verbose output (scan counts, selection info and per-file unchanged results on stderr) (env YEWSEAL_ENCRYPT_VERBOSE)
```

## Options inherited from parent commands

```
  -k, --key-file string   Path to the Age private key file (fallback: YEWSEAL_AGE_IDENTITIES, SOPS_AGE_KEY, SOPS_AGE_KEY_FILE, YEWSEAL_AGE_KEY_CMD, SOPS_AGE_KEY_CMD, then .age/keys.txt in the current directory) (env YEWSEAL_KEY_FILE)
```

## SEE ALSO

* [yews](/references/yews/)	 - YewSeal - Encrypt/decrypt configuration files using SOPS and Age (supports TOML, YAML, JSON, ENV, INI, and binary files)
