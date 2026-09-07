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

传入目录路径时，YewSeal 从已登记映射中选择明文侧位于该目录内的项，包括显式 FilePair 和 Group 发现结果。Group 始终按所属配置目录和规则发现文件，不以目标目录为新根重新扫描；没有 Group 时仍可选择显式映射。

临时加密一个未登记文件且不需要项目配置时，请直接使用 SOPS CLI，用法见[与 SOPS 配合使用](/guide/sops)。

## 选项

### --output, -o

为单文件目标指定加密文件路径。

```bash
yews encrypt config.toml -o config.enc.toml
```

`--output` 只支持文件目标，不支持配置模式或目录扫描。

### --pattern

进一步筛选配置范围或目录目标内已登记的映射。规则相对于当前工作目录，可匹配明文或密文路径，不覆盖 Group 的发现规则或登记额外文件。

```bash
yews encrypt ./configs --pattern "*.toml" --pattern "!*.enc.toml"
```

规则支持 `*`、`?`、`**`、以 `!` 开头的排除规则、以 `/` 开头的根目录锚定规则和以 `/` 结尾的目录规则。

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

# .dev.vars 的 ENV 格式应在项目配置中声明
yews encrypt .dev.vars -o .dev.enc.env
```

## 输出路径

显式 FilePair 使用配置中的 `encrypted` 路径；Group 扫描结果按格式协议生成对应的 `.enc.*` 路径。单文件目标可以用 `--output` 临时覆盖写入路径，但不会改变配置中的文件映射或授权集合。

`--output` 只改变位置，不改变格式，即使输出文件的扩展名不同也是如此。业务命令没有 `--format` 选项，非标准扩展名通过配置中的 `format` 或 `format_rules` 声明。

## 相关命令

[plan](/commands/plan) 可以检查当前配置映射及授权，但不是加密预演；[decrypt](/commands/decrypt) 可以解密文件，[diff](/commands/diff) 可以比较明文和加密文件。
