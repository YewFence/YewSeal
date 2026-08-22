---
title: yews completion fish
---

## yews completion fish

Generate the autocompletion script for fish

### Synopsis

Generate the autocompletion script for the fish shell.

To load completions in your current shell session:

	yews completion fish | source

To load completions for every new session, execute once:

	yews completion fish > ~/.config/fish/completions/yews.fish

You will need to start a new shell for this setup to take effect.


```
yews completion fish [flags]
```

### Options

```
  -h, --help              help for fish
      --no-descriptions   disable completion descriptions
```

### Options inherited from parent commands

```
  -k, --key-file string   Path to Age private key file
```

### SEE ALSO

* [yews completion](yews_completion.md)	 - Generate the autocompletion script for the specified shell
