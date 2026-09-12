---
title: yews
---

YewSeal - Encrypt/decrypt configuration files using SOPS and Age (supports TOML, YAML, JSON, ENV, INI)

### Synopsis

YewSeal manages encrypted configuration files with SOPS and Age,
natively supporting TOML, YAML, JSON, ENV, INI, and binary formats.

File mappings, formats, and recipient authorization are declared once in
.yewseal.toml ("yews init" scaffolds it); command arguments only select
among the registered files.

Configuration precedence: CLI flags > environment variables > config file
> defaults.

Identity resolution order: --key-file (env AGE_KEY_FILE), then
YEWSEAL_AGE_IDENTITIES, SOPS_AGE_KEY, SOPS_AGE_KEY_FILE, SOPS_AGE_KEY_CMD,
then .age/keys.txt in the current working directory.

Documentation: https://yewfence.github.io/YewSeal/

### Options

```
  -h, --help              help for yews
  -k, --key-file string   Path to the Age private key file (env AGE_KEY_FILE; fallback: YEWSEAL_AGE_IDENTITIES, SOPS_AGE_KEY*, then .age/keys.txt)
```

### SEE ALSO

* [yews completion](yews_completion/)	 - Generate the autocompletion script for the specified shell
* [yews decrypt](yews_decrypt/)	 - Decrypt encrypted file (output format determined by extension)
* [yews diff](yews_diff/)	 - Compare plaintext file with decrypted encrypted file
* [yews edit](yews_edit/)	 - Edit encrypted configuration file using SOPS
* [yews encrypt](yews_encrypt/)	 - Encrypt configuration file (supports .toml, .yaml, .yml, .json, .env, .ini, and binary output)
* [yews init](yews_init/)	 - Initialize project with Age keys and YewSeal config entries
* [yews plan](yews_plan/)	 - Check configured file mappings, formats, and recipient authorization without writing files
* [yews view](yews_view/)	 - Print decrypted plaintext to standard output without writing files
