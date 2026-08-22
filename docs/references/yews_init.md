---
title: yews init
---

## yews init

Initialize project with Age keys and YewSeal config entries

```
yews init [flags]
```

### Options

```
      --create-example     Create example file (non-interactive mode)
  -f, --force              Force overwrite existing configuration
      --format string      Format override for the first config entry (toml/yaml/json/env/ini/binary)
  -h, --help               help for init
  -i, --input string       Plaintext file for the first config entry (non-interactive mode)
  -o, --output string      Encrypted file for the first config entry (non-interactive mode)
      --skip-sops-config   Skip creating .sops.yaml file (non-interactive mode)
```

### Options inherited from parent commands

```
  -k, --key-file string   Path to Age private key file
```

### SEE ALSO

* [yews](yews.md)	 - YewSeal - Encrypt/decrypt configuration files using SOPS and Age (supports TOML, YAML, JSON, ENV, INI)
