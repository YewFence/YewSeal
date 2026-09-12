---
title: Shell completion
---

YewSeal uses Cobra's native completion; no extra generator tooling is required.

## Generating completion scripts

After building the binary, the `completion` subcommand emits scripts for Bash, Zsh, Fish, and PowerShell.

```bash
mise run build
./build/yews completion zsh > _yews
./build/yews completion bash > yews.bash
./build/yews completion fish > yews.fish
./build/yews completion powershell > yews.ps1
```

## Installation examples

For Zsh, put the generated `_yews` into a directory already on `$fpath`, or into a custom directory added to `$fpath` in `~/.zshrc`.

```bash
mkdir -p ~/.zsh/completions
./build/yews completion zsh > ~/.zsh/completions/_yews
```

```zsh
fpath=(~/.zsh/completions $fpath)
autoload -Uz compinit
compinit
```

For Bash, drop the script into a local directory and `source` it, or let the system's bash-completion directory manage it.

```bash
mkdir -p ~/.bash_completion.d
./build/yews completion bash > ~/.bash_completion.d/yews.bash
source ~/.bash_completion.d/yews.bash
```

For Fish, write straight into the user completions directory.

```bash
mkdir -p ~/.config/fish/completions
./build/yews completion fish > ~/.config/fish/completions/yews.fish
```
