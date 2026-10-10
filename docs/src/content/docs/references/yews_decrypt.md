---
title: yews decrypt
---

Decrypt registered files in place or deliver a mirrored plaintext tree

## Synopsis

Decrypt registered SOPS-encrypted files to their configured plaintext
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

Mappings with plaintext_mode=delivery are skipped by default (outcome
"delivery-restricted") without touching the registered paths; other
selected mappings continue. --inplace explicitly allows these mappings at
their configured plaintext paths; --output DIR delivers them normally.
--inplace and --output are mutually exclusive. The default plaintext_mode
is inplace. --force is a separate overwrite control; it does not supply
inplace consent. A delivery skip follows the usual lenient/strict exit
rules. Delivery classification is an accident-prevention default; use separate
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
not match ("no-matching-identity"), or when delivery mappings have no
--inplace or --output consent ("delivery-restricted"); even a fully skipped
batch exits 0.
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
(--verbose adds selection info, scan counts, and per-file success). --json replaces
stdout with the batch report (summary and per-file outcomes); stderr
diagnostics stay unchanged, and when the run never starts (a calling
error, exit 2) stdout stays empty.

For unregistered files, see the SOPS interoperability guide.

See also: "yews plan" to inspect mappings and authorization, "yews view"
to print plaintext, "yews diff" to compare it with stored ciphertext.

Target selection: https://yewfence.github.io/YewSeal/guide/target-selection
Managed-file rules: https://yewfence.github.io/YewSeal/guide/configuration#managed-files
SOPS interoperability: https://yewfence.github.io/YewSeal/guide/sops
Result classification and exit codes: https://yewfence.github.io/YewSeal/guide/decryption-results
Plaintext delivery and caller-owned lifecycle: https://yewfence.github.io/YewSeal/guide/plaintext-delivery
Scan exclusions: https://yewfence.github.io/YewSeal/guide/configuration#scan-exclusions

```
yews decrypt [command options] [path-or-pattern]... [flags]
```

## Examples

```
  # Decrypt registered ciphertext under the current directory
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
  yews decrypt --json > report.json
```

## Options

```
  -f, --force              Force overwrite existing plaintext file when it differs from decrypted content (env YEWSEAL_DECRYPT_FORCE)
  -h, --help               help for decrypt
      --inplace            Allow delivery plaintext at configured paths; conflicts with --output (env YEWSEAL_DECRYPT_INPLACE)
      --json               Print the batch report as JSON on stdout (diagnostics stay on stderr) (env YEWSEAL_DECRYPT_JSON)
      --output string      Deliver the project-relative plaintext tree into an existing, real, empty directory; conflicts with --force=true and --inplace (env YEWSEAL_DECRYPT_OUTPUT)
  -P, --parallel int       Number of parallel workers for batch mode (minimum 1) (env YEWSEAL_DECRYPT_PARALLEL) (default 1)
      --strict             Require every selected file to be decrypted (successful writes are not rolled back) (env YEWSEAL_DECRYPT_STRICT)
      --update-gitignore   Add plaintext and default key entries to .gitignore for configured-path writes; ignored with --output; false leaves it untouched (env YEWSEAL_UPDATE_GITIGNORE) (default true)
  -v, --verbose            Enable verbose output (scan counts, selection info and per-file results on stderr) (env YEWSEAL_DECRYPT_VERBOSE)
```

## Options inherited from parent commands

```
  -k, --key-file string   Path to the Age private key file (fallback: YEWSEAL_AGE_IDENTITIES, SOPS_AGE_KEY, SOPS_AGE_KEY_FILE, YEWSEAL_AGE_KEY_CMD, SOPS_AGE_KEY_CMD, then .age/keys.txt in the current directory) (env YEWSEAL_KEY_FILE)
```

## SEE ALSO

* [yews](/references/yews/)	 - YewSeal - Encrypt/decrypt configuration files using SOPS and Age (supports TOML, YAML, JSON, ENV, INI, and binary files)
