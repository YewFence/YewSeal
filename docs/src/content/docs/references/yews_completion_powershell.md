---
title: yews completion powershell
---

Generate the autocompletion script for powershell

## Synopsis

Generate the autocompletion script for powershell.

To load completions in your current shell session:

	yews completion powershell | Out-String | Invoke-Expression

To load completions for every new session, add the output of the above command
to your powershell profile.


```
yews completion powershell [flags]
```

## Options

```
  -h, --help              help for powershell
      --no-descriptions   disable completion descriptions
```

## Options inherited from parent commands

```
  -k, --key-file string   Path to the Age private key file (fallback: YEWSEAL_AGE_IDENTITIES, SOPS_AGE_KEY, SOPS_AGE_KEY_FILE, SOPS_AGE_KEY_CMD, then .age/keys.txt) (env YEWSEAL_KEY_FILE)
```

## SEE ALSO

* [yews completion](/references/yews_completion/)	 - Generate the autocompletion script for the specified shell
