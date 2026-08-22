---
title: yews plan
---

## yews plan

Run preflight and print the resolved file selection without writing files

```
yews plan [command options] [path] [flags]
```

### Options

```
      --format string         Format override for file targets (toml/yaml/json/env/ini/binary)
      --format-rule strings   Group format rule in <pattern>=<format> form
  -h, --help                  help for plan
      --json                  Print preflight result as JSON
  -o, --output string         Output file for a single file target
  -P, --parallel int          Number of parallel workers for batch mode (default 1)
      --pattern strings       Group pattern for directory mode or encryption.groups override
      --unknown-as-binary     Allow group mode to treat unknown formats as binary
  -v, --verbose               Enable verbose output
```

### Options inherited from parent commands

```
  -k, --key-file string   Path to Age private key file
```

### SEE ALSO

* [yews](yews.md)	 - YewSeal - Encrypt/decrypt configuration files using SOPS and Age (supports TOML, YAML, JSON, ENV, INI)
