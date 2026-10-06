---
title: yews decrypt
---

Decrypt encrypted file to its configured plaintext path

## Synopsis

Decrypt registered SOPS-encrypted files to their configured plaintext
paths. The format comes from the project config or the registered file
path; runtime format overrides and cross-format conversion are not
supported.

With no arguments, decrypt selects registered ciphertext under the current
directory and its subdirectories. Use file paths, directories, or patterns
to select targets; directories and patterns match encrypted paths.

The config still governs plaintext/ciphertext paths and formats, but the
recipients actually used for decryption come from the ciphertext's SOPS
metadata: when an alias referenced by the current config no longer
exists, decrypt warns on stderr and continues with the identity bundle.

Overwrite protection: an existing plaintext file whose content differs
from the decryption result is not overwritten unless --force is set.
Before decryption, --update-gitignore (enabled by default) adds plaintext
entries for selected mappings, or all configured mappings when no target is
given, using the managed-file rules.

Exit codes: by default, files are skipped when no Age identity is
available (outcome "no-identity") or when the available identities do
not match ("no-matching-identity"); even a fully skipped batch exits 0,
so lenient callers can treat unavailable decryption access as a
degradable condition. Real errors (a missing or corrupted ciphertext,
read or write failures, overwrite conflicts) and output delivery
failures exit 1. With --strict, any skip also exits 1, but remaining
files are still processed and successful results are kept;
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

Target selection: https://yewfence.github.io/YewSeal/guide/target-selection
Managed-file rules: https://yewfence.github.io/YewSeal/guide/configuration#managed-files
SOPS interoperability: https://yewfence.github.io/YewSeal/guide/sops
Result classification and exit codes: https://yewfence.github.io/YewSeal/guide/decryption-results

```
yews decrypt [command options] [path-or-pattern]... [flags]
```

## Examples

```
  # Decrypt registered ciphertext under the current directory
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
  yews decrypt --json > report.json
```

## Options

```
  -f, --force              Force overwrite existing plaintext file when it differs from decrypted content (env YEWSEAL_DECRYPT_FORCE)
  -h, --help               help for decrypt
      --json               Print the batch report as JSON on stdout (diagnostics stay on stderr) (env YEWSEAL_DECRYPT_JSON)
  -o, --output string      Output plaintext file for a single file target (env YEWSEAL_DECRYPT_OUTPUT)
  -P, --parallel int       Number of parallel workers for batch mode (minimum 1) (env YEWSEAL_DECRYPT_PARALLEL) (default 1)
      --strict             Require every selected file to be decrypted (env YEWSEAL_DECRYPT_STRICT)
      --update-gitignore   Add plaintext and default key entries to .gitignore; false leaves it untouched (env YEWSEAL_UPDATE_GITIGNORE) (default true)
  -v, --verbose            Enable verbose output (selection info and per-file results on stderr) (env YEWSEAL_DECRYPT_VERBOSE)
```

## Options inherited from parent commands

```
  -k, --key-file string   Path to the Age private key file (fallback: YEWSEAL_AGE_IDENTITIES, SOPS_AGE_KEY, SOPS_AGE_KEY_FILE, YEWSEAL_AGE_KEY_CMD, SOPS_AGE_KEY_CMD, then .age/keys.txt in the current directory) (env YEWSEAL_KEY_FILE)
```

## SEE ALSO

* [yews](/references/yews/)	 - YewSeal - Encrypt/decrypt configuration files using SOPS and Age (supports TOML, YAML, JSON, ENV, INI, and binary files)
