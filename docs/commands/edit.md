# edit - 编辑加密文件

`edit` 会把加密文件解密到临时文件，打开编辑器，保存后再重新加密写回原文件。

## 语法

```bash
yews edit [command options]
```

## 选项

### --file, -f

指定要编辑的加密文件，默认值是 `wrangler.enc.toml.yaml`。

```bash
yews edit -f config.enc.toml.yaml
```

这个目标也可以是 `.yewseal.toml` 中声明过的明文路径，YewSeal 会找到对应的加密文件。

### --editor, -e

指定编辑器命令。

```bash
yews edit -f config.enc.toml.yaml -e "code -w"
```

如果没有传入 `--editor`，YewSeal 会依次使用 `EDITOR`、`VISUAL`，在 Windows 上会尝试 `code -w` 或 `notepad`，其他系统默认使用 `vi`。

## 示例

```bash
# 使用默认目标
yews edit

# 编辑指定加密文件
yews edit -f config.enc.toml.yaml

# 使用 VS Code 并等待窗口关闭
yews edit -f config.enc.toml.yaml -e "code -w"

# 使用 Vim
yews edit -f config.enc.toml.yaml -e vim
```

## 注意事项

编辑器命令需要等待文件保存并关闭后再退出。比如 VS Code 需要使用 `code -w`，否则 YewSeal 可能在文件还没编辑完成时就继续执行。

## 相关命令

[view](/commands/view) 可以只读查看加密文件，[decrypt](/commands/decrypt) 可以写出明文文件。
