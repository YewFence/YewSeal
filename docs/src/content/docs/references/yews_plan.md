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
the run. The report includes recipient aliases, canonical recipients,
registry origins, and the effective authorization source. plan does not
load an identity bundle and does not read ciphertext content or
metadata, so it has no historical-decrypt tolerance.

Output: a table report on stdout (config count, selection scope, file
mappings); --json prints only JSON; errors go to stderr and never mix
into the report. plan defines no output or worker flags, reads no
output-related environment variables, and its report contains no
metadata write plan.

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

  # Print JSON for scripts (errors stay on stderr)
  yews plan --json > plan.json
```

## Options

```
  -h, --help      help for plan
      --json      Print configured file mappings as JSON
  -v, --verbose   Enable verbose output
```

## Options inherited from parent commands

```
  -k, --key-file string   Path to the Age private key file (env AGE_KEY_FILE; fallback: YEWSEAL_AGE_IDENTITIES, SOPS_AGE_KEY*, then .age/keys.txt)
```

## SEE ALSO

* [yews](/references/yews/)	 - YewSeal - Encrypt/decrypt configuration files using SOPS and Age (supports TOML, YAML, JSON, ENV, INI, and binary files)
