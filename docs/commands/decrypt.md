# decrypt - 解密配置文件

`decrypt` 用来把 SOPS 加密文件解密成明文文件，格式由项目配置或已登记文件的路径推断决定，不支持运行时格式覆盖或跨格式转换。

## 语法

```bash
yews decrypt [command options] [path]
```

别名是 `d`。

## 目标选择

不传 `path` 时，YewSeal 会处理 `.yewseal.toml` 中当前目录范围内的全部已配置 FilePair 和 Group 扫描结果。传入文件路径时，该路径必须匹配已加载配置中 FilePair 的 plaintext 或 encrypted 任一侧；传入目录时，只扫描已配置 Group 管理的文件。

配置仍负责治理明文、密文路径和格式，但历史密文的实际 recipient 事实来自其 SOPS metadata。当前配置引用的 alias 已删除或重命名时，decrypt 会向 stderr 输出非致命 warning，并继续使用 identity bundle 尝试解密。

临时解密一个未登记文件且不需要项目配置时，请直接使用 SOPS CLI；原生 TOML 密文需要带对应 store 的 fork CLI。用法和格式兼容边界见[与 SOPS 配合使用](/guide/sops)。

## 选项

### --output, -o

为单文件目标指定明文输出路径。

```bash
yews decrypt config.enc.toml -o config.toml
```

`--output` 只支持文件目标，不支持配置模式或目录扫描。

它表示一个输出文件，不是输出目录。批量模式即使只选中一个文件也不支持 `--output`。输出路径的扩展名不会改变解密格式，覆盖保护仍然生效。

### --pattern

为配置模式或目录扫描指定匹配规则。

```bash
yews decrypt ./configs --pattern "*.toml"
```

目录解密时，`--pattern` 匹配的是逻辑明文路径。比如 `--pattern "*.toml"` 可以选中 `config.enc.toml`。

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

### --strict

要求所有选中文件成功处理。默认情况下，没有匹配身份的文件会跳过；部分成功且没有真正错误时退出 `0`，全部跳过或发生真正错误时退出 `1`。严格模式下只要有跳过也退出 `1`，但仍继续处理其他文件，不回滚成功结果。

```bash
yews decrypt --strict
YEWSEAL_STRICT=true yews decrypt
```

显式 `--strict=false` 可以覆盖环境变量。完整结果分类、退出码和 `.gitignore` 副作用见[解密结果与严格模式](/guide/decryption-results)。

### --verbose, -v

输出详细的文件选择信息。

```bash
yews decrypt -v
```

## 示例

```bash
# 解密配置中的所有文件
yews decrypt

# 解密一个已配置的加密文件，使用配置中的 plaintext 路径
yews decrypt config.enc.toml

# 解密单个加密文件，并指定输出路径
yews decrypt config.enc.toml -o config.toml

# 在已配置 Group 的目录范围内筛选并解密
yews decrypt ./configs --pattern "*.toml"

```

## 覆盖保护

默认情况下，如果输出文件已经存在且内容与解密结果不同，`decrypt` 会拒绝覆盖。需要覆盖时传入 `--force`。

## TOML 输出格式

TOML 由内嵌的原生 TOML store 直接解密，不经过格式转换。解密输出是规范化 TOML：字符串使用单引号字面量风格，注释会保留，内容与原文等价，但排版可能与手写格式不同。

## 相关命令

[plan](/commands/plan) 可以先预览文件选择，[view](/commands/view) 可以把明文打印到标准输出，[diff](/commands/diff) 可以比较明文和加密文件。
