---
title: yews encrypt
---

Encrypt configuration file (supports .toml, .yaml, .yml, .json, .env, .ini, and binary output)

## Synopsis

Encrypt registered configuration files with SOPS and Age. Supported
formats: .toml, .yaml, .yml, .json, .env, .ini, and binary output; every
format is encrypted natively by the embedded SOPS engine and the
ciphertext keeps the original format.

Target selection (no argument: every file and group in .yewseal.toml
within the current directory scope):
  - a registered plaintext or encrypted path selects that single mapping;
  - an existing directory selects mappings whose plaintext side is inside
    it (groups always scan by their own config directory, never by the
    target directory);
  - arguments containing *, ?, and similar metacharacters are patterns
    matched against registered plaintext paths (a leading / anchors to
    the current working directory, ** is supported);
  - multiple arguments take the union; patterns only include, excludes
    come from group "patterns" in the config; any argument matching
    nothing is an error.

Recipients come strictly from alias resolution in .yewseal.toml (file
"recipients" > group > top-level recipients.defaults); there is no
--public-key flag. An empty final set, an unknown alias, or groups
disagreeing on the same path fails the whole batch before any ciphertext
is written.

When ciphertext already exists, encrypt uses an available private identity
to verify and update it in place: unchanged files remain byte-identical,
unchanged values retain their ciphertext, and recipient-only changes only
rewrap the existing data key. If no identity is available, or none matches a
specific file, encrypt warns and replaces that ciphertext from the current
plaintext. --force always performs this fresh encryption and rotates the data
key without reading the old ciphertext. Missing plaintext is reported and
skipped without creating an output directory.

--output only changes the location, never the format. There is no
--format flag: non-standard extensions are declared via "format" or
"format_rules" in the config. Group results get the format's standard
.enc.* path; --output applies to single file targets only, never to
config-wide or directory-driven batches.

After processing, --sync-sops-config (enabled by default) rewrites
.sops.yaml from the complete resolved project policy, not only the selected
targets. Encryption always finishes before synchronization is attempted.
A synchronization failure leaves completed ciphertext work in place but
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

See also: "yews plan" to audit mappings and authorization (not an
encrypt dry run), "yews diff" to compare plaintext with the stored
ciphertext.

Documentation: https://yewfence.github.io/YewSeal/guide/target-selection

```
yews encrypt [command options] [path-or-pattern]... [flags]
```

## Examples

```
  # Encrypt every file registered in the config
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
      --sync-sops-config   Sync the complete project policy to .sops.yaml after encryption (env YEWSEAL_ENCRYPT_SYNC_SOPS_CONFIG) (default true)
  -v, --verbose            Enable verbose output (env YEWSEAL_ENCRYPT_VERBOSE)
```

## Options inherited from parent commands

```
  -k, --key-file string   Path to the Age private key file (fallback: YEWSEAL_AGE_IDENTITIES, SOPS_AGE_KEY, SOPS_AGE_KEY_FILE, YEWSEAL_AGE_KEY_CMD, SOPS_AGE_KEY_CMD, then .age/keys.txt in the current directory) (env YEWSEAL_KEY_FILE)
```

## SEE ALSO

* [yews](/references/yews/)	 - YewSeal - Encrypt/decrypt configuration files using SOPS and Age (supports TOML, YAML, JSON, ENV, INI, and binary files)
