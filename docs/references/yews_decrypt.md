---
title: yews decrypt
---

## yews decrypt

Decrypt encrypted file (output format determined by extension)

```
yews decrypt [command options] [path] [flags]
```

### Options

```
  -f, --force                 Force overwrite existing plaintext file when it differs from decrypted content
      --format string         Format override for file targets (toml/yaml/json/env/ini/binary)
      --format-rule strings   Group format rule in <pattern>=<format> form
  -h, --help                  help for decrypt
  -o, --output string         Output plaintext file for a single file target
  -P, --parallel int          Number of parallel workers for batch mode (default 1)
      --pattern strings       Pattern filter for configured groups
      --unknown-as-binary     Allow group mode to treat unknown encrypted inputs as binary when needed
  -v, --verbose               Enable verbose output
```

### Options inherited from parent commands

```
  -k, --key-file string   Path to Age private key file
```

### SEE ALSO

* [yews](yews.md)	 - YewSeal - Encrypt/decrypt configuration files using SOPS and Age (supports TOML, YAML, JSON, ENV, INI)
