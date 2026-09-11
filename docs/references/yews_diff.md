---
title: yews diff
---

## yews diff

Compare plaintext file with decrypted encrypted file

```
yews diff [path-or-pattern]... [flags]
```

### Options

```
      --color string   Colorize diff output (auto/always/never) (default "auto")
  -h, --help           help for diff
      --strict         Require comparison of files with both inputs present (default from YEWSEAL_STRICT)
  -v, --verbose        Enable verbose output
```

### Options inherited from parent commands

```
  -k, --key-file string   Path to Age private key file
```

### SEE ALSO

* [yews](yews.md)	 - YewSeal - Encrypt/decrypt configuration files using SOPS and Age (supports TOML, YAML, JSON, ENV, INI)
