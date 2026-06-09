# check - 检查依赖

`check` 会显示 YewSeal 的依赖状态。

## 语法

```bash
yews check
yews doctor
```

别名是 `doctor`。

## 当前依赖模型

Age 和 SOPS 已经作为 Go 库内嵌进 YewSeal，不需要额外安装命令行工具。TOML 支持仍然需要外部的 `remarshal`，因为 YewSeal 会通过 `toml2yaml` 和 `yaml2toml` 做 TOML 与 YAML 转换。

## 示例

```bash
yews check
yews doctor
```

输出会列出内嵌库状态，并提示 `remarshal` 是否可用。

## 安装 remarshal

```bash
uv tool install remarshal
```

也可以使用 `pipx` 安装。

```bash
pipx install remarshal
```

如果只处理 YAML、JSON、ENV、INI 或二进制文件，不安装 `remarshal` 也可以使用核心加解密能力。

## 相关命令

[encrypt](/commands/encrypt) 和 [decrypt](/commands/decrypt) 在处理 TOML 文件时会需要 `remarshal`。
