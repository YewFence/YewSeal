# edit - 编辑加密文件

`edit` 会把加密文件解密到临时文件，打开编辑器，保存后再重新加密写回原文件。

## 语法

```bash
yews edit [command options]
```

## 选项

### --file, -f

指定 `.yewseal.toml` 中已登记的加密文件或对应明文路径。该选项没有默认目标，未配置路径会报错。

```bash
yews edit -f config.enc.toml
```

这个目标也可以是 `.yewseal.toml` 中声明过的明文路径，YewSeal 会找到对应的加密文件。

## 编辑器

YewSeal 依次读取 `VISUAL`、`EDITOR`；均未设置或为空时，Windows 默认使用 `notepad`，其他系统默认使用 `vi`。不再提供 `--editor` / `-e` 选项。

环境变量支持可执行文件及参数，采用以下受限语法，不是完整的 shell 命令：

- 空格、制表符和换行分隔参数；单引号、双引号可以包住含空格的路径或参数，相邻片段会合并。
- 保留空参数（`""` 或 `''`），但可执行文件不能为空。
- 单引号内所有字符均为字面值；双引号内只对 `\"` 和 `\\` 去掉转义反斜杠，其他反斜杠保留。Windows 路径请用引号包住。
- 引号外反斜杠转义下一个字符；未闭合引号、末尾孤立反斜杠、NUL 和无效 UTF-8 会报错。
- 引号外未转义的 `| & ; < > ( ) $` 和反引号会报错；需要字面值时请引用或转义。不会展开环境变量、命令替换、通配符或 `~`，也不会把 `#` 解释为注释。

YewSeal 直接启动编辑器进程并等待退出，将临时文件路径作为独立的最后一个参数传入，不经过 shell。

## 示例

```bash
# 编辑指定加密文件
yews edit -f config.enc.toml

# 使用 VS Code 并等待窗口关闭
VISUAL="code --wait" yews edit -f config.enc.toml

# 使用 Vim
VISUAL=vim yews edit -f config.enc.toml
```

## 注意事项

编辑器命令需要等待文件保存并关闭后再退出。比如 VS Code 需要使用 `code -w`，否则 YewSeal 可能在文件还没编辑完成时就继续执行。

## 相关命令

[view](/commands/view) 可以只读查看加密文件，[decrypt](/commands/decrypt) 可以写出明文文件。
