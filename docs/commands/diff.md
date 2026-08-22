# diff - 比较文件差异

`diff` 会比较明文文件与解密后的加密文件内容。

## 语法

```bash
yews diff [command options] [target]
```

## 目标选择

不传 `target` 时，`diff` 会比较配置中当前目录范围内的文件。传入 `target` 时，可以使用明文文件路径或加密文件路径。

```bash
yews diff config.toml
```

## 选项

### --format

为选定目标指定格式覆盖，支持 `toml`、`yaml`、`json`、`env`、`ini` 和 `binary`。

```bash
yews diff config.txt --format toml
```

### --color

控制差异输出的颜色，支持 `auto`、`always` 和 `never`，默认值是 `auto`。

```bash
yews diff config.toml --color always
```

### --verbose, -v

输出详细的文件选择信息。

```bash
yews diff config.toml -v
```

## 示例

```bash
# 比较配置中的所有文件
yews diff

# 比较单个明文文件
yews diff config.toml

# 比较单个加密文件对应的明文
yews diff config.enc.toml

# 在脚本中禁用颜色
yews diff --color never
```

## 退出码

内容一致时退出码是 `0`，内容不一致时退出码是 `1`。发生参数或解密错误时也会以非零退出码结束。

## 相关命令

[encrypt](/commands/encrypt) 可以重新加密明文文件，[view](/commands/view) 可以查看加密文件内容。
