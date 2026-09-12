---
title: yews completion bash
---

Generate the autocompletion script for bash

### Synopsis

Generate the autocompletion script for the bash shell.

This script depends on the 'bash-completion' package.
If it is not installed already, you can install it via your OS's package manager.

To load completions in your current shell session:

	source <(yews completion bash)

To load completions for every new session, execute once:

#### Linux:

	yews completion bash > /etc/bash_completion.d/yews

#### macOS:

	yews completion bash > $(brew --prefix)/etc/bash_completion.d/yews

You will need to start a new shell for this setup to take effect.


```
yews completion bash
```

### Options

```
  -h, --help              help for bash
      --no-descriptions   disable completion descriptions
```

### Options inherited from parent commands

```
  -k, --key-file string   Path to the Age private key file (env AGE_KEY_FILE; fallback: YEWSEAL_AGE_IDENTITIES, SOPS_AGE_KEY*, then .age/keys.txt)
```

### SEE ALSO

* [yews completion](yews_completion/)	 - Generate the autocompletion script for the specified shell
