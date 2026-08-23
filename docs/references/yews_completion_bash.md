---
title: yews completion bash
---

## yews completion bash

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
  -k, --key-file string   Path to Age private key file
```

### SEE ALSO

* [yews completion](yews_completion.md)	 - Generate the autocompletion script for the specified shell
