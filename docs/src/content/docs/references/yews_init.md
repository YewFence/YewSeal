---
title: yews init
---

Initialize project with Age keys and YewSeal config entries

### Synopsis

Initialize a project: generate an Age key pair, create .yewseal.toml
with its first config entry, optionally sync .sops.yaml, and update
.gitignore.

Run without flags for interactive mode: YewSeal asks whether to rebuild
an existing configuration, whether to create .sops.yaml, and records
one or more plaintext/encrypted mappings. Passing --input or --output
switches to non-interactive mode for scripts.

Generated files:
  .yewseal.toml  main YewSeal config (recipient registry, defaults,
                 file entries)
  .age/keys.txt  Age private key; must not be committed to version
                 control
  .sops.yaml     SOPS config for direct sops usage; skipped with
                 --skip-sops-config

When only --input is given, the encrypted file name is inferred
(config.toml becomes config.enc.toml; other formats use the matching
.enc.* suffix).

--force rebuilds keys, recipient registry, defaults, file entries, and
the managed .sops.yaml; old aliases and mappings are not preserved, and
existing ciphertext may become undecryptable for the new owner
identity.

Output: stdout stays empty; prompts, warnings, errors, and the
completion summary (mapping count and key file locations) go to stderr,
answers are read from stdin.

See also: "yews encrypt" to encrypt the registered files, "yews decrypt"
to decrypt them. Private key storage and distribution are managed
outside YewSeal.

Documentation: https://yewfence.github.io/YewSeal/guide/getting-started
Private key handling: https://yewfence.github.io/YewSeal/guide/private-keys

```
yews init [flags]
```

### Examples

```
  # Interactive setup
  yews init

  # Non-interactive first mapping
  yews init --input config.toml --output config.enc.toml --format toml

  # Infer the encrypted path from the plaintext file
  yews init --input .dev.vars --format env

  # Rebuild keys and configuration from scratch (existing ciphertext
  # may become undecryptable)
  yews init --force
```

### Options

```
      --create-example     Create an example plaintext file (interactive: for recorded entries; non-interactive: for the first entry)
  -f, --force              Rebuild keys and configuration; existing ciphertext may become undecryptable
      --format string      Format override for the first config entry (toml/yaml/json/env/ini/binary)
  -h, --help               help for init
  -i, --input string       Plaintext file for the first config entry (switches to non-interactive mode)
  -o, --output string      Encrypted file for the first config entry (non-interactive mode)
      --skip-sops-config   Skip creating or updating .sops.yaml (non-interactive mode)
```

### Options inherited from parent commands

```
  -k, --key-file string   Path to the Age private key file (env AGE_KEY_FILE; fallback: YEWSEAL_AGE_IDENTITIES, SOPS_AGE_KEY*, then .age/keys.txt)
```

### SEE ALSO

* [yews](yews/)	 - YewSeal - Encrypt/decrypt configuration files using SOPS and Age (supports TOML, YAML, JSON, ENV, INI)
