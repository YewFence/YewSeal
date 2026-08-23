# decrypt - 解密配置文件

`decrypt` 用来把 SOPS 加密文件解密成明文文件，输出格式由配置、命令行格式覆盖或加密文件名协议决定。

## 语法

```bash
yews decrypt [command options] [path]
```

别名是 `d`。

## 目标选择

不传 `path` 时，YewSeal 会使用 `.yewseal.toml` 中当前目录范围内的配置项。传入加密文件路径时，会解密这个文件。传入目录路径时，会扫描目录下符合加密文件名协议的文件。

如果目标在 `[[encryption.files]]` 中声明过，会使用配置里的明文路径和格式。如果目标没有配置，`config.enc.toml` 会解密到 `config.toml`，`config.enc.yaml` 会解密到 `config.yaml`，其他支持格式也按同样协议推断。

## 选项

### --output, -o

为单文件目标指定明文输出路径。

```bash
yews decrypt config.enc.toml -o config.toml
```

`--output` 只支持文件目标，不支持配置模式或目录扫描。

### --format

为文件目标指定输出格式，支持 `toml`、`yaml`、`json`、`env`、`ini` 和 `binary`。

```bash
yews decrypt config.enc.yaml --format toml -o config.toml
```

`--format` 只支持单文件模式。

### --format-rule

为分组扫描指定格式规则，形式是 `<pattern>=<format>`。

```bash
yews decrypt ./configs --format-rule ".dev.vars=env"
```

### --pattern

为配置模式或目录扫描指定匹配规则。

```bash
yews decrypt ./configs --pattern "*.toml"
```

目录解密时，`--pattern` 匹配的是逻辑明文路径。比如 `--pattern "*.toml"` 可以选中 `config.enc.toml`。

### --unknown-as-binary

在分组扫描中，允许把无法识别格式的加密输入按二进制文件处理。

```bash
yews decrypt ./secrets --unknown-as-binary
```

### --parallel, -P

设置批量模式的并行工作线程数，默认值是 `1`。

```bash
yews decrypt ./configs --parallel 4
```

### --force, -f

当明文文件已存在且内容不同的时候强制覆盖。

```bash
yews decrypt config.enc.toml --force
```

### --verbose, -v

输出详细的文件选择信息。

```bash
yews decrypt -v
```

## 示例

```bash
# 解密配置中的所有文件
yews decrypt

# 解密单个加密文件，输出路径由文件名推断
yews decrypt config.enc.toml

# 解密单个加密文件，并指定输出路径
yews decrypt config.enc.toml -o config.toml

# 解密目录中匹配的加密文件
yews decrypt ./configs --pattern "*.toml"

# 输出为指定格式
yews decrypt config.enc.yaml --format json -o config.json
```

## 覆盖保护

默认情况下，如果输出文件已经存在且内容与解密结果不同，`decrypt` 会拒绝覆盖。需要覆盖时传入 `--force`。

## TOML 输出格式

TOML 由内嵌的原生 TOML store 直接解密，不经过格式转换。解密输出是规范化 TOML：字符串使用单引号字面量风格，注释会保留，内容与原文等价，但排版可能与手写格式不同。

## 相关命令

[plan](/commands/plan) 可以先预览文件选择，[view](/commands/view) 可以把明文打印到标准输出，[diff](/commands/diff) 可以比较明文和加密文件。
