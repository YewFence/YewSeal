---
title: yews
---

YewSeal - Encrypt/decrypt configuration files using SOPS and Age (supports TOML, YAML, JSON, ENV, INI, and binary files)

## Synopsis

YewSeal manages encrypted configuration files with SOPS and Age,
natively supporting TOML, YAML, JSON, ENV, INI, and binary formats.

File mappings, formats, and recipient authorization are declared once in
.yewseal.toml ("yews init" scaffolds it); command arguments only select
among the registered files.

CLI option precedence: flags > command environment variables > defaults.
Project mappings and authorization come only from .yewseal.toml.

Identity resolution order: --key-file (env YEWSEAL_KEY_FILE), then
SOPS_AGE_KEY, SOPS_AGE_KEY_FILE, SOPS_AGE_KEY_CMD, then .age/keys.txt in
the current working directory.

Documentation: https://yewfence.github.io/YewSeal/

## Options

```
  -h, --help              help for yews
  -k, --key-file string   Path to the Age private key file (fallback: SOPS_AGE_KEY*, then .age/keys.txt) (env YEWSEAL_KEY_FILE)
```

## SEE ALSO

* [yews completion](/references/yews_completion/)	 - Generate the autocompletion script for the specified shell
* [yews decrypt](/references/yews_decrypt/)	 - Decrypt encrypted file to its configured plaintext path
* [yews diff](/references/yews_diff/)	 - Compare plaintext file with decrypted encrypted file
* [yews edit](/references/yews_edit/)	 - Edit encrypted configuration file using SOPS
* [yews encrypt](/references/yews_encrypt/)	 - Encrypt configuration file (supports .toml, .yaml, .yml, .json, .env, .ini, and binary output)
* [yews init](/references/yews_init/)	 - Initialize project with Age keys and YewSeal config entries
* [yews plan](/references/yews_plan/)	 - Check configured file mappings, formats, and recipient authorization without writing files
* [yews view](/references/yews_view/)	 - Print decrypted plaintext to standard output without writing files
