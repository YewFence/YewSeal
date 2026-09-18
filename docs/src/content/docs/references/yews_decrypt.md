---
title: yews decrypt
---

Decrypt encrypted file to its configured plaintext path

## Synopsis

Decrypt registered SOPS-encrypted files to their configured plaintext
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
identity are skipped and never fail the run; even a fully skipped batch
exits 0, so lenient callers can treat "no matching identity" as a
degradable condition. Real errors (a missing or corrupted ciphertext,
read or write failures, overwrite conflicts) and output delivery
failures exit 1. With --strict, any skip also exits 1, but remaining
files are still processed and successful results are kept;
--strict=false overrides YEWSEAL_DECRYPT_STRICT. Calling errors
(invalid arguments, a missing or invalid .yewseal.toml, selection
failure, or an unusable identity source) exit 2.

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

Documentation: https://yewfence.github.io/YewSeal/guide/target-selection
Result classification and exit codes: https://yewfence.github.io/YewSeal/guide/decryption-results

```
yews decrypt [command options] [path-or-pattern]... [flags]
```

## Examples

```
  # Decrypt every file registered in the config
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
  -f, --force           Force overwrite existing plaintext file when it differs from decrypted content (env YEWSEAL_DECRYPT_FORCE)
  -h, --help            help for decrypt
      --json            Print the batch report as JSON on stdout (diagnostics stay on stderr) (env YEWSEAL_DECRYPT_JSON)
  -o, --output string   Output plaintext file for a single file target (env YEWSEAL_DECRYPT_OUTPUT)
  -P, --parallel int    Number of parallel workers for batch mode (minimum 1) (env YEWSEAL_DECRYPT_PARALLEL) (default 1)
      --strict          Require every selected file to be decrypted (env YEWSEAL_DECRYPT_STRICT)
  -v, --verbose         Enable verbose output (selection info and per-file results on stderr) (env YEWSEAL_DECRYPT_VERBOSE)
```

## Options inherited from parent commands

```
  -k, --key-file string   Path to the Age private key file (fallback: YEWSEAL_AGE_IDENTITIES, SOPS_AGE_KEY, SOPS_AGE_KEY_FILE, YEWSEAL_AGE_KEY_CMD, SOPS_AGE_KEY_CMD, then .age/keys.txt in the current directory) (env YEWSEAL_KEY_FILE)
```

## SEE ALSO

* [yews](/references/yews/)	 - YewSeal - Encrypt/decrypt configuration files using SOPS and Age (supports TOML, YAML, JSON, ENV, INI, and binary files)
