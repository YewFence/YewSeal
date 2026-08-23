---
title: yews sync pull
---

## yews sync pull

Pull key from secret management service to local file

```
yews sync pull [flags]
```

### Options

```
      --env string          Environment name in the provider
  -h, --help                help for pull
  -k, --key-file string     Local path to save the key file (default ".age/keys.txt")
  -n, --name string         Secret name in the provider (default "AGE_KEY_FILE")
      --path string         Path/folder in the provider (e.g., /yewseal)
      --project-id string   Infisical project ID
  -p, --provider string     Secret management provider (infisical) (default "infisical")
```

### SEE ALSO

* [yews sync](yews_sync.md)	 - Sync sensitive files to secret management service
