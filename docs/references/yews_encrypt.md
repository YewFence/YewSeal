---
title: yews encrypt
---

## yews encrypt

Encrypt configuration file (supports .toml, .yaml, .yml, .json, .env, .ini, and binary output)

### Synopsis

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
    matched against registered plaintext paths (a leading / anchors,
    ** is supported), relative to the current working directory;
  - multiple arguments take the union; patterns only include, excludes
    come from group "patterns" in the config; any argument matching
    nothing is an error.

Recipients come strictly from alias resolution in .yewseal.toml (file
"recipients" > group > top-level recipients.defaults); there is no
--public-key flag. An empty final set, an unknown alias, or groups
disagreeing on the same path fails the whole batch before any ciphertext
is written.

--output only changes the location, never the format. There is no
--format flag: non-standard extensions are declared via "format" or
"format_rules" in the config. Group results get the format's standard
.enc.* path; --output applies to single file targets only, never to
config-wide or directory-driven batches.

Exit codes: 0 on success, 1 when any file fails or the selection is
empty. Output: ciphertext goes to files and stdout stays empty; warnings,
per-file failure reasons, and the summary go to stderr (--verbose adds
selection info and per-file success).

To encrypt an unregistered file ad hoc without a project config, use
the SOPS CLI directly.

See also: "yews plan" to audit mappings and authorization (not an
encrypt dry run), "yews diff" to compare plaintext with the stored
ciphertext.

Documentation: https://yewfence.github.io/YewSeal/guide/target-selection

```
yews encrypt [command options] [path-or-pattern]... [flags]
```

### Examples

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
```

### Options

```
  -h, --help            help for encrypt
  -o, --output string   Output encrypted file for a single file target (env SOPS_OUTPUT_FILE)
  -P, --parallel int    Number of parallel workers for batch mode (minimum 1) (default 1)
  -v, --verbose         Enable verbose output
```

### Options inherited from parent commands

```
  -k, --key-file string   Path to the Age private key file (env AGE_KEY_FILE; fallback: YEWSEAL_AGE_IDENTITIES, SOPS_AGE_KEY*, then .age/keys.txt)
```

### SEE ALSO

* [yews](yews.md)	 - YewSeal - Encrypt/decrypt configuration files using SOPS and Age (supports TOML, YAML, JSON, ENV, INI)
