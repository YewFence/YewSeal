# encrypt - 加密配置文件

`encrypt` 用来把明文配置文件加密成 SOPS 文件，支持 TOML、YAML、JSON、ENV、INI 和二进制文件。所有格式（含 TOML）都由内嵌的 SOPS 引擎原生加密，加密文件保持原格式。

## 语法

```bash
yews encrypt [command options] [path]
```

别名是 `e`。

## 目标选择

不传 `path` 时，YewSeal 会处理 `.yewseal.toml` 中当前目录范围内的全部 `[[encryption.files]]` 和 `[[encryption.groups]]`。没有配置文件、配置中没有 file/group，或当前 scope 没有可选文件时会报错。

传入文件路径时，该路径必须匹配已加载配置中 FilePair 的 plaintext 或 encrypted 任一侧；命中后始终使用完整的已配置映射和授权集合。未登记文件不会被临时转换为加密目标。

传入目录路径时，YewSeal 只扫描配置中 Group 管理的文件。没有已配置 Group 时会报错。

## 选项

### --output, -o

为单文件目标指定加密文件路径。

```bash
yews encrypt config.toml -o config.enc.toml
```

`--output` 只支持文件目标，不支持配置模式或目录扫描。

### --format

为文件目标指定格式，支持 `toml`、`yaml`、`json`、`env`、`ini` 和 `binary`。

```bash
yews encrypt .dev.vars --format env -o .dev.enc.env
```

`--format` 只支持单文件模式。批量场景需要使用 `--format-rule` 或配置里的 `format_rules`。

### --format-rule

为分组扫描指定格式规则，形式是 `<pattern>=<format>`。

```bash
yews encrypt ./configs --format-rule ".dev.vars=env"
```

可以多次传入。命令行规则会追加到配置规则之后。

### --pattern

为配置模式或目录扫描指定匹配规则。

```bash
yews encrypt ./configs --pattern "*.toml" --pattern "!*.enc.toml"
```

规则支持 `*`、`?`、`**`、以 `!` 开头的排除规则、以 `/` 开头的根目录锚定规则和以 `/` 结尾的目录规则。

### --unknown-as-binary

在分组扫描中，把无法识别格式的明文文件按二进制文件加密。

```bash
yews encrypt ./secrets --unknown-as-binary
```

### --parallel, -P

设置批量模式的并行工作线程数，默认值是 `1`。

```bash
yews encrypt ./configs --parallel 4
```

### Recipient 授权

命令没有 `--public-key` 选项。每个文件的 recipient 由 `.yewseal.toml` 中的 alias 严格解析：显式 FilePair `recipients` 优先于 Group，Group 优先于顶层 `recipients.defaults`。最终集合为空、包含未知 alias 或多个 Group 对同一路径给出不同集合时，整个批次会在写入任何密文前失败。
### --verbose, -v

输出详细的文件选择信息。

```bash
yews encrypt -v
```

## 示例

```bash
# 加密配置中的所有文件
yews encrypt

# 加密一个已配置的 FilePair，可使用 plaintext 或 encrypted 路径定位
yews encrypt config.toml
yews encrypt config.enc.toml

# 为已配置目标临时覆盖输出路径
yews encrypt config.toml -o review/config.enc.toml

# 在已配置 Group 的目录范围内筛选文件
yews encrypt ./configs --pattern "*.toml"

# 为已配置的非标准扩展名目标临时覆盖格式
yews encrypt .dev.vars --format env -o .dev.enc.env
```

## 输出路径

显式 FilePair 使用配置中的 `encrypted` 路径；Group 扫描结果按格式协议生成对应的 `.enc.*` 路径。单文件目标可以用 `--output` 临时覆盖写入路径，但不会改变配置中的文件映射或授权集合。

## 相关命令

[plan](/commands/plan) 可以先预览文件选择，[decrypt](/commands/decrypt) 可以解密文件，[diff](/commands/diff) 可以比较明文和加密文件。
