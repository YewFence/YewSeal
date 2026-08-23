---
title: yews sync
---

## yews sync

Sync sensitive files to secret management service

```
yews sync [flags]
```

### Options

```
      --env string          Environment name in the provider
  -h, --help                help for sync
  -k, --key-file string     Path to the key file to sync (default ".age/keys.txt")
  -n, --name string         Secret name in the provider (default "AGE_KEY_FILE")
      --path string         Path/folder in the provider (e.g., /yewseal)
      --project-id string   Infisical project ID
  -p, --provider string     Secret management provider (infisical) (default "infisical")
```

### SEE ALSO

* [yews](yews.md)	 - YewSeal - Encrypt/decrypt configuration files using SOPS and Age (supports TOML, YAML, JSON, ENV, INI)
* [yews sync pull](yews_sync_pull.md)	 - Pull key from secret management service to local file
