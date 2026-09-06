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
  -h, --help              help for plan
      --json              Print preflight result as JSON
  -o, --output string     Output file for a single file target
  -P, --parallel int      Number of parallel workers for batch mode (minimum 1) (default 1)
      --pattern strings   Group pattern for directory mode or encryption.groups override
  -v, --verbose           Enable verbose output
```

### Options inherited from parent commands

```
  -k, --key-file string   Path to Age private key file
```

### SEE ALSO

* [yews](yews.md)	 - YewSeal - Encrypt/decrypt configuration files using SOPS and Age (supports TOML, YAML, JSON, ENV, INI)
