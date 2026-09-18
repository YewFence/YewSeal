---
title: yews completion fish
---

Generate the autocompletion script for fish

## Synopsis

Generate the autocompletion script for the fish shell.

To load completions in your current shell session:

	yews completion fish | source

To load completions for every new session, execute once:

	yews completion fish > ~/.config/fish/completions/yews.fish

You will need to start a new shell for this setup to take effect.


```
yews completion fish [flags]
```

## Options

```
  -h, --help              help for fish
      --no-descriptions   disable completion descriptions
```

## Options inherited from parent commands

```
  -k, --key-file string   Path to the Age private key file (fallback: YEWSEAL_AGE_IDENTITIES, SOPS_AGE_KEY, SOPS_AGE_KEY_FILE, SOPS_AGE_KEY_CMD, then .age/keys.txt) (env YEWSEAL_KEY_FILE)
```

## SEE ALSO

* [yews completion](/references/yews_completion/)	 - Generate the autocompletion script for the specified shell
