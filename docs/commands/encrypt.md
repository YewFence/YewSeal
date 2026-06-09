# encrypt - 加密配置文件

`encrypt` 用来把明文配置文件加密成 SOPS 文件，支持 TOML、YAML、JSON、ENV、INI 和二进制文件。TOML 会先转换成 YAML，再以 YAML 形式写入加密文件。

## 语法

```bash
yews encrypt [command options] [path]
```

别名是 `e`。

## 目标选择

不传 `path` 时，YewSeal 会使用 `.yewseal.toml` 中当前目录范围内的 `[[encryption.files]]` 和 `[[encryption.groups]]`。如果没有配置文件，则使用默认映射 `wrangler.toml` 到 `wrangler.enc.toml.yaml`。

传入文件路径时，YewSeal 会加密这个明文文件。如果该文件已在配置中声明，会使用配置中的加密路径和格式；如果没有配置，会根据文件名推断输出路径。

传入目录路径时，YewSeal 会扫描目录。没有传 `--pattern` 时会匹配 `.toml`、`.yaml`、`.yml`、`.json`、`.env`、`.ini`、`.bin` 和 `.binary`，并排除常见的 `.enc.*` 文件。

## 选项

### --output, -o

为单文件目标指定加密文件路径。

```bash
yews encrypt config.toml -o config.enc.toml.yaml
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
yews encrypt ./configs --pattern "*.toml" --pattern "!*.enc.toml.yaml"
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

### --public-key, -p

指定 Age 公钥。没有传入时，会优先使用 `.yewseal.toml` 的 `[key].public_key`，再从私钥文件中读取公钥。

```bash
yews encrypt config.toml -p age1xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

### --verbose, -v

输出详细的文件选择信息。

```bash
yews encrypt -v
```

## 示例

```bash
# 加密配置中的所有文件
yews encrypt

# 加密单个文件，输出路径由文件名推断
yews encrypt config.toml

# 加密单个文件，并指定输出路径
yews encrypt config.toml -o config.enc.toml.yaml

# 扫描目录并加密匹配文件
yews encrypt ./configs --pattern "*.toml"

# 为非标准扩展名指定格式
yews encrypt .dev.vars --format env -o .dev.enc.env
```

## 输出文件协议

YewSeal 会按格式推断加密文件名。`config.toml` 会变成 `config.enc.toml.yaml`，`config.yaml` 会变成 `config.enc.yaml`，`config.json` 会变成 `config.enc.json`，`.env` 会变成 `.env.enc.env`，`config.ini` 会变成 `config.enc.ini`，二进制文件会变成 `.enc.bin`。

## 相关命令

[plan](/commands/plan) 可以先预览文件选择，[decrypt](/commands/decrypt) 可以解密文件，[diff](/commands/diff) 可以比较明文和加密文件。
