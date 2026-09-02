---
title: yews encrypt
---

## yews encrypt

Encrypt configuration file (supports .toml, .yaml, .yml, .json, .env, .ini, and binary output)

```
yews encrypt [command options] [path] [flags]
```

### Options

```
      --format string         Format override for file targets (toml/yaml/json/env/ini/binary)
      --format-rule strings   Group format rule in <pattern>=<format> form
  -h, --help                  help for encrypt
  -o, --output string         Output encrypted file for a single file target
  -P, --parallel int          Number of parallel workers for batch mode (default 1)
      --pattern strings       Group pattern for temporary directory mode or encryption.groups override
      --unknown-as-binary     Allow group mode to encrypt unknown plaintext formats as binary
  -v, --verbose               Enable verbose output
```

### Options inherited from parent commands

```
  -k, --key-file string   Path to Age private key file
```

### SEE ALSO

* [yews](yews.md)	 - YewSeal - Encrypt/decrypt configuration files using SOPS and Age (supports TOML, YAML, JSON, ENV, INI)
