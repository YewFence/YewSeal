# check - 检查依赖

`check` 会显示 YewSeal 的依赖状态。

## 语法

```bash
yews check
yews doctor
```

别名是 `doctor`。

## 当前依赖模型

Age 和 SOPS 已经作为 Go 库内嵌进 YewSeal，不需要额外安装任何命令行工具。TOML 支持也是原生的（由内置的 [YewFence/sops](https://github.com/YewFence/sops) fork 提供），不再需要 `remarshal` 做格式转换。

## 示例

```bash
yews check
yews doctor
```

输出会列出内嵌库状态，并确认没有外部工具依赖。

## 相关命令

[encrypt](/commands/encrypt) 和 [decrypt](/commands/decrypt) 直接内嵌处理所有格式，包括 TOML。
