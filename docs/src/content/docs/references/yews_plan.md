---
title: yews plan
---

Check configured file mappings, formats, and recipient authorization without writing files

## Synopsis

Inspect registered file mappings, formats, and current-config
authorization without encrypting, decrypting, or writing any file. This
is a directionless mapping check, not an encrypt/decrypt dry run:
success does not guarantee that files can be encrypted, that ciphertext
can be opened, or that output paths are writable.

Target selection (no argument: mappings with either side within the
current directory scope):
  - a registered plaintext or encrypted path selects that single mapping;
  - an existing directory selects mappings with either side inside it;
  - arguments containing *, ?, and similar metacharacters are patterns
    matched against either side of registered mappings;
  - multiple arguments take the union; patterns only include; any
    argument matching nothing is an error.
Paths only select mappings; they never imply an operation direction.

Groups always scan by their own config directory, and plan uses the
union of plaintext-side and ciphertext-side discovery: files present on
only one side still show up (for example config.yml next to
config.enc.yaml keeps the discovered real plaintext path). Competing
mappings must be resolved by an explicit file entry or plan reports a
conflict. Explicit entries do not require the files to exist.

plan applies the same strict authorization semantics as encrypt to
every mapping resolved from the loaded config, including unselected
ones: unknown aliases, empty recipient sets, and group conflicts fail
the run. The report shows recipient aliases; --source additionally shows
the origins of each path, format, authorization set, and registry alias.
Use --json to audit canonical recipients, selection reasons, and all
provenance fields. plan does not load an identity bundle and does not
read ciphertext content or metadata, so it has no historical-decrypt
tolerance.

Output: stdout shows a config count and selection scope followed by a
four-column Plaintext/Encrypted/Format/Aliases table. --source replaces
the table with one describe block per mapping and field-level origins;
--verbose also lists loaded config files. --json prints only the full
JSON report and takes precedence over --source; errors go to stderr and
never mix into the report. plan defines no output-path or worker flags,
reads no output-path environment variables, and its report contains no
metadata write plan.

Exit codes: 0 on success; calling errors (invalid patterns, a missing
or invalid .yewseal.toml, or authorization conflicts) exit 2.

See also: "yews encrypt" and "yews decrypt" share the registered-mapping
selection, with different discovery sides and historical-decrypt
authorization handling.

Documentation: https://yewfence.github.io/YewSeal/guide/configuration

```
yews plan [command options] [path-or-pattern]... [flags]
```

## Examples

```
  # Inspect the mappings within the current directory scope
  yews plan

  # Either side of a mapping selects it
  yews plan config.toml
  yews plan config.enc.toml

  # Filter registered mappings with a pattern
  yews plan './configs/*.toml'

  # Trace where each path, format, and authorization set came from
  yews plan --source

  # Print JSON for scripts (errors stay on stderr)
  yews plan --json > plan.json
```

## Options

```
  -h, --help      help for plan
      --json      Print configured file mappings as JSON (env YEWSEAL_PLAN_JSON)
      --source    Show field-level origins in a describe layout (plain output only) (env YEWSEAL_PLAN_SOURCE)
  -v, --verbose   Enable verbose output (env YEWSEAL_PLAN_VERBOSE)
```

## Options inherited from parent commands

```
  -k, --key-file string   Path to the Age private key file (fallback: YEWSEAL_AGE_IDENTITIES, SOPS_AGE_KEY, SOPS_AGE_KEY_FILE, YEWSEAL_AGE_KEY_CMD, SOPS_AGE_KEY_CMD, then .age/keys.txt in the current directory) (env YEWSEAL_KEY_FILE)
```

## SEE ALSO

* [yews](/references/yews/)	 - YewSeal - Encrypt/decrypt configuration files using SOPS and Age (supports TOML, YAML, JSON, ENV, INI, and binary files)
