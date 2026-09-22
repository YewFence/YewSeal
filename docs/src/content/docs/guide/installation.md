---
title: Installation
---

Install the `yews` CLI, then follow the [Tutorial](/guide/tutorial) to encrypt your first configuration file.

## Install the CLI

### mise

Install the prebuilt release binary via the [github backend](https://mise.jdx.dev/dev-tools/backends/github.html) of [mise](https://mise.jdx.dev/):

```bash
mise use --global github:YewFence/YewSeal
yews --version
```

You can also declare `github:YewFence/YewSeal` in a project's `mise.toml` so teammates and CI share one pinned version.

### go install

```bash
go install github.com/YewFence/YewSeal/cmd/yews@latest
```

Requires `$GOPATH/bin` on your `$PATH`.

### GitHub release

Download the prebuilt binary for your system from the [releases page](https://github.com/YewFence/YewSeal/releases). The executable is always named `yews` (`yews.exe` on Windows).

### Build from source

```bash
git clone https://github.com/YewFence/YewSeal.git
cd YewSeal
mise run install    # installs into $GOPATH/bin
# or build only: mise run build → bin/yews
```

### Docker

Without installing a binary, run the container image `ghcr.io/yewfence/yew-seal` directly; see [Running with Docker](/guide/docker).

## Shell completion (Optional)

`yews` ships Cobra's native completion for Bash, Zsh, Fish, and PowerShell:

```bash
yews completion bash    # bash
yews completion zsh     # zsh
yews completion fish    # fish
yews completion powershell
```

Typical installation spots:

```bash
# zsh: a directory on $fpath
mkdir -p ~/.zsh/completions
yews completion zsh > ~/.zsh/completions/_yews

# bash: sourced from ~/.bashrc
yews completion bash > ~/.bash_completion.d/yews.bash

# fish: the user completions directory
yews completion fish > ~/.config/fish/completions/yews.fish
```

## Next steps

- [Tutorial](/guide/tutorial) — take a project from `yews init` to a decrypted clone on another machine
- [Agent skills](/guide/agent-skills) — optional: teach your coding agent the YewSeal workflow and its boundaries
- [Working with a team](/guide/working-with-a-team) — shared repositories, multiple identities, strict mode
