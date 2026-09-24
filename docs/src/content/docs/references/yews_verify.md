---
title: yews verify
---

Check ciphertext, recipients, plaintext consistency, version-control exposure, and .sops.yaml drift

## Synopsis

Check the health of registered encrypted files without modifying any
project file. verify never prints plaintext, private keys, or data keys.

Target selection is the same as plan: directionless, with either side of
a mapping selecting it (no argument: mappings with either side within the
current directory scope; a registered path selects one mapping; a
directory filters registered mappings; patterns match either side; any
argument matching nothing is an error). Explicit entries do not require
their files to exist; a missing ciphertext is a finding.

Checks:
  configuration   the same strict authorization as plan and encrypt; an
                  invalid config, unknown alias, empty recipient set, or
                  group conflict is a calling error, not a finding.
  ciphertext      no private key needed: the encrypted file exists, is a
                  regular file, parses as SOPS, contains exactly one key
                  group with no non-Age keys, and its Age recipients equal
                  the configured public keys
                  (order ignored; renaming an alias without changing its
                  key is not drift).
  decryption      runs when an Age identity is available: decrypts, checks
                  the MAC, and compares an existing local plaintext with
                  decrypted content by parsed structure. Without an identity it is
                  skipped. --decrypt requires an identity and treats a
                  file no identity can open as an error; --no-decrypt reads
                  no identity source. A missing plaintext is skipped.
  version control from the nearest git or jj repository (.jj wins when both
                  exist): selected plaintext and file-backed Age keys that
                  are already in history (git index; jj @-) are errors,
                  files one commit away (git untracked and not ignored;
                  jj @ only) are warnings. Ignoring a file after it was
                  committed does not clear the error. Symlink paths and their
                  effective targets are both checked. Outside a repository
                  this check is skipped; a failed git or jj query is an
                  error. jj snapshots its working copy while listing @.
  .sops.yaml      compared with the complete resolved verify policy;
                  a difference is an error, an absent file is skipped.
                  --sync-sops-config=false (shared
                  with init and encrypt) skips the comparison.

Finding codes are stable: ciphertext_missing, ciphertext_not_regular,
ciphertext_stat_error, ciphertext_read_error, ciphertext_parse_error,
ciphertext_no_recipients, key_groups_unsupported, recipient_unsupported, recipient_missing, recipient_extra,
recipient_duplicate, decrypt_failed, mac_mismatch,
decrypt_no_matching_identity (warning, error with --decrypt),
plaintext_read_error, plaintext_parse_error, plaintext_drift, plaintext_tracked,
plaintext_not_ignored (warning), key_tracked, key_not_ignored (warning),
vcs_query_failed, sops_config_read_error, sops_config_generate_error,
sops_config_drift. All are errors unless marked.

Output: stdout carries a summary line, one line per skipped check kind,
then each finding with a hint. The summary counts individual checks
(each file contributes one check per layer): pass and skipped count
checks, warning and error count findings. --json prints only
{"ok", "summary", "skipped", "findings"} on stdout, where "skipped" lists
the skip reasons. Errors go to stderr.

Exit codes: 0 when no finding is an error (warnings and skips allowed);
1 when at least one finding is an error ("fix the repository"); 2 when
verify cannot run or report ("fix the environment"): invalid arguments
(including --decrypt with --no-decrypt), a missing or invalid
.yewseal.toml, selection or authorization failure, an unreadable explicit
key file, a failed key command, --decrypt without any identity, or a
report that cannot be written.

See also: "yews plan" for mappings and authorization only, "yews encrypt"
to repair recipient drift, "yews diff" to inspect a plaintext difference.

Documentation: https://yewfence.github.io/YewSeal/guide/verifying

```
yews verify [command options] [path-or-pattern]... [flags]
```

## Examples

```
  # Check everything in the current directory scope
  yews verify

  # Check one registered mapping
  yews verify config/production.yaml

  # CI without private keys: static checks only
  yews verify --no-decrypt

  # CI with a key: fail unless every file can be decrypted
  yews verify --decrypt --key-file /run/secrets/yewseal-identities

  # Machine-readable report
  yews verify --json > verify.json
```

## Options

```
      --decrypt            Require an Age identity and treat undecryptable files as errors (env YEWSEAL_VERIFY_DECRYPT)
  -h, --help               help for verify
      --json               Print the verify report as JSON on stdout (errors stay on stderr) (env YEWSEAL_VERIFY_JSON)
      --no-decrypt         Skip decryption checks without reading any identity source (env YEWSEAL_VERIFY_NO_DECRYPT)
      --sync-sops-config   Check .sops.yaml against the resolved policy; false skips the comparison (env YEWSEAL_SYNC_SOPS_CONFIG) (default true)
```

## Options inherited from parent commands

```
  -k, --key-file string   Path to the Age private key file (fallback: YEWSEAL_AGE_IDENTITIES, SOPS_AGE_KEY, SOPS_AGE_KEY_FILE, YEWSEAL_AGE_KEY_CMD, SOPS_AGE_KEY_CMD, then .age/keys.txt in the current directory) (env YEWSEAL_KEY_FILE)
```

## SEE ALSO

* [yews](/references/yews/)	 - YewSeal - Encrypt/decrypt configuration files using SOPS and Age (supports TOML, YAML, JSON, ENV, INI, and binary files)
