# Shell 补全

YewSeal 使用 Cobra 原生补全能力，不需要额外引入生成工具。

## 生成补全脚本

构建二进制后，可以通过 `completion` 子命令生成 Bash、Zsh、Fish 和 PowerShell 的补全脚本。

```bash
mise run build
./build/yews completion zsh > _yews
./build/yews completion bash > yews.bash
./build/yews completion fish > yews.fish
./build/yews completion powershell > yews.ps1
```

## 安装示例

Zsh 可以把生成的 `_yews` 放到 `$fpath` 中已有的目录，或者放到自定义目录后在 `~/.zshrc` 里加入该目录。

```bash
mkdir -p ~/.zsh/completions
./build/yews completion zsh > ~/.zsh/completions/_yews
```

```zsh
fpath=(~/.zsh/completions $fpath)
autoload -Uz compinit
compinit
```

Bash 可以把补全脚本放到本地目录后手动 `source`，也可以交给系统的 bash-completion 目录管理。

```bash
mkdir -p ~/.bash_completion.d
./build/yews completion bash > ~/.bash_completion.d/yews.bash
source ~/.bash_completion.d/yews.bash
```

Fish 可以直接写入用户补全目录。

```bash
mkdir -p ~/.config/fish/completions
./build/yews completion fish > ~/.config/fish/completions/yews.fish
```
