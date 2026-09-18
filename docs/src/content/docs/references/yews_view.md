---
title: yews view
---

Print decrypted plaintext to standard output without writing files

## Synopsis

Print the decrypted plaintext of one registered encrypted file to
standard output without writing any plaintext file.

The target must match the plaintext or encrypted path of a registered
file. The format comes from the project config or the registered path;
view outputs only the file's own format and performs no cross-format
conversion (pipe the output into a converter when needed).

view shares decrypt's full selection and historical-decrypt semantics:
it warns and continues when the configured alias no longer exists, and
decrypts according to the ciphertext metadata and the current identity
bundle. It stays single-target: it fails without emitting empty
plaintext and never touches .gitignore or .sops.yaml.

Output: stdout carries only the plaintext; warnings, errors, and
--verbose detail go to stderr, so the plaintext stays pipeable.

See also: "yews decrypt" to write plaintext files with overwrite
protection, "yews edit" to edit the encrypted file directly.

Documentation: https://yewfence.github.io/YewSeal/guide/target-selection

```
yews view [command options] <target> [flags]
```

## Examples

```
  # Print the decrypted plaintext of a registered file
  yews view config.enc.toml

  # Pipe a registered JSON file into jq
  yews view config.enc.json | jq '.database'

  # Save only the plaintext; detail stays on stderr
  yews view config.enc.toml --verbose > inspected.toml
```

## Options

```
  -h, --help      help for view
  -v, --verbose   Enable verbose output (detail goes to stderr; stdout stays plaintext only) (env YEWSEAL_VIEW_VERBOSE)
```

## Options inherited from parent commands

```
  -k, --key-file string   Path to the Age private key file (fallback: SOPS_AGE_KEY*, then .age/keys.txt) (env YEWSEAL_KEY_FILE)
```

## SEE ALSO

* [yews](/references/yews/)	 - YewSeal - Encrypt/decrypt configuration files using SOPS and Age (supports TOML, YAML, JSON, ENV, INI, and binary files)
