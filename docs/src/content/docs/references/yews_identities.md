---
title: yews identities
---

List the Age identities YewSeal would decrypt with, their winning source, and registry aliases

## Synopsis

List the Age identities YewSeal would decrypt with: which source won the
resolution chain, which present sources it shadowed, and every identity in
the winning source with its derived public key and registry alias.

Identity resolution is first-present, never merged: the first configured
source applies even when it contains no valid identity, and everything
below it is not read — the exact order is the --key-file fallback chain
shown among the flags below. shadowed lists the sources that were present
but skipped, as file:<path> or env:<NAME>. Malformed items are warned and
ignored; a source with no valid items produces an empty report while
retaining its source label. With no source at all, source is empty.

The command requires a .yewseal.toml like every other command beyond
version, help, and completion: each derived public key is looked up in
[recipients.registry], and an unregistered public key warns on stderr
and carries a warning field in --json output.

By default no secret key material is printed. --reveal includes it:
--json adds a secret field per identity, plain output adds a Secret
column. The values then flow to stdout — mind terminal scrollback and CI
logs (GitHub Actions only redacts exact repository-secret matches); pipe
into a file or a consuming process instead of logging.

Output: plain mode prints a Source/Shadowed header plus an Alias/Public
key table (plus Secret with --reveal) on stdout; --json prints the report
on stdout; warnings go to stderr either way. An empty JSON report contains
"source": "" when no source is configured and always has "identities": [].

Exit codes: 0 on success; 1 when the report or a warning cannot be
delivered (closed or full stdout/stderr); 2 when an explicit key file is
unreadable, a key command fails, or .yewseal.toml is missing or invalid.
No source, an empty source, and a source containing only malformed items
all produce an empty report with exit 0.

See also: "yews plan" to preview file mappings and authorization on the
other side of the pipeline.

Documentation: https://yewfence.github.io/YewSeal/guide/configuration#reading-private-keys

```
yews identities [flags]
```

## Examples

```
  # Which identities does this machine decrypt with, and from where
  yews identities

  # Audit which configured sources a --key-file shadows
  yews identities --key-file .age/keys.txt

  # Machine-readable report including secret keys (CI: pipe, do not log)
  yews identities --json --reveal > bundle.json

  # Inspect a specific candidate key file before adopting it
  yews identities --key-file /path/to/new-key.txt
```

## Options

```
  -h, --help     help for identities
      --json     Print the identity report as JSON on stdout (warnings stay on stderr) (env YEWSEAL_IDENTITIES_JSON)
      --reveal   Include each identity's secret key (JSON adds a secret field; plain output adds a Secret column) (env YEWSEAL_IDENTITIES_REVEAL)
```

## Options inherited from parent commands

```
  -k, --key-file string   Path to the Age private key file (fallback: YEWSEAL_AGE_IDENTITIES, SOPS_AGE_KEY, SOPS_AGE_KEY_FILE, YEWSEAL_AGE_KEY_CMD, SOPS_AGE_KEY_CMD, then .age/keys.txt in the current directory) (env YEWSEAL_KEY_FILE)
```

## SEE ALSO

* [yews](/references/yews/)	 - YewSeal - Encrypt/decrypt configuration files using SOPS and Age (supports TOML, YAML, JSON, ENV, INI, and binary files)
