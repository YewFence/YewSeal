---
title: yews diff
---

## yews diff

Compare plaintext file with decrypted encrypted file

### Synopsis

Compare local plaintext files with the decrypted content of their
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

Documentation: https://yewfence.github.io/YewSeal/guide/decryption-results

```
yews diff [path-or-pattern]... [flags]
```

### Examples

```
  # Compare every registered file in scope
  yews diff

  # Compare one mapping via either side
  yews diff config.toml
  yews diff config.enc.toml

  # Disable color in scripts; diagnostics stay on stderr
  yews diff --color never > changes.diff
```

### Options

```
      --color string   Colorize diff output (auto/always/never) (default "auto")
  -h, --help           help for diff
      --strict         Require comparison of mappings with both inputs present (env YEWSEAL_STRICT; --strict=false overrides it)
  -v, --verbose        Enable verbose output (selection info and per-file completion notes on stderr)
```

### Options inherited from parent commands

```
  -k, --key-file string   Path to the Age private key file (env AGE_KEY_FILE; fallback: YEWSEAL_AGE_IDENTITIES, SOPS_AGE_KEY*, then .age/keys.txt)
```

### SEE ALSO

* [yews](yews.md)	 - YewSeal - Encrypt/decrypt configuration files using SOPS and Age (supports TOML, YAML, JSON, ENV, INI)
