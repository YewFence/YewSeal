---
title: yews
---

## yews

YewSeal - Encrypt/decrypt configuration files using SOPS and Age (supports TOML, YAML, JSON, ENV, INI)

### Options

```
  -h, --help              help for yews
  -k, --key-file string   Path to Age private key file
```

### SEE ALSO

* [yews completion](yews_completion.md)	 - Generate the autocompletion script for the specified shell
* [yews decrypt](yews_decrypt.md)	 - Decrypt encrypted file (output format determined by extension)
* [yews diff](yews_diff.md)	 - Compare plaintext file with decrypted encrypted file
* [yews edit](yews_edit.md)	 - Edit encrypted configuration file using SOPS
* [yews encrypt](yews_encrypt.md)	 - Encrypt configuration file (supports .toml, .yaml, .yml, .json, .env, .ini, and binary output)
* [yews init](yews_init.md)	 - Initialize project with Age keys and YewSeal config entries
* [yews plan](yews_plan.md)	 - Run preflight and print the resolved file selection without writing files
* [yews view](yews_view.md)	 - Print decrypted plaintext to standard output without writing files
