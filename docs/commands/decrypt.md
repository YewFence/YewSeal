# decrypt - 解密配置文件

`decrypt` 用来把 SOPS 加密文件解密成明文文件，格式由项目配置或已登记文件的路径推断决定，不支持运行时格式覆盖或跨格式转换。

## 语法

```bash
yews decrypt [command options] [path-or-pattern]...
```

别名是 `d`。

## 目标选择

不传 `path` 时，YewSeal 会处理 `.yewseal.toml` 中密文侧位于当前目录范围内的全部已配置 FilePair 和 Group 发现结果。传入文件路径时，该路径必须匹配已加载配置中 FilePair 的 plaintext 或 encrypted 任一侧；传入目录时，选择密文侧位于该目录内的已登记映射，包括显式 FilePair。Group 始终按所属配置目录和规则发现文件，不以目标目录为新根重新扫描。

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

### 位置参数：文件、目录与模式

每个位置参数是一个选择器：已登记映射的明文或密文路径选中单个文件；已存在的目录选中其范围内（密文侧）的映射；含 `*`、`?` 等元字符的参数视为模式，与已登记映射的密文路径求交集。多个参数取并集，模式只包含不排除，排除规则由 Group 的 `patterns` 在配置中声明。任一参数没有命中都会报错。

```bash
# 选中 ./configs 下所有已登记的 .enc.toml 密文
yews decrypt './configs/*.enc.toml'
```

模式支持 `*`、`?`、`**`、以 `/` 开头的锚定规则，相对于当前工作目录。

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

向 stderr 额外输出文件选择信息和逐文件成功结果。

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

# 用模式选中 ./configs 下已登记的密文并解密
yews decrypt './configs/*.enc.toml'

```

## 输出

明文写入目标文件，stdout 为空。stderr 显示警告、逐文件跳过或失败的原因和最终汇总；使用 `--verbose` 时还会显示文件选择信息和逐文件成功结果。

```bash
# 明文仍写入目标文件，诊断消息单独保存
yews decrypt --strict 2> decrypt.log
```

## 覆盖保护

默认情况下，如果输出文件已经存在且内容与解密结果不同，`decrypt` 会拒绝覆盖。需要覆盖时传入 `--force`。

## TOML 输出格式

TOML 由内嵌的原生 TOML store 直接解密，不经过格式转换。解密输出是规范化 TOML：字符串使用单引号字面量风格，注释会保留，内容与原文等价，但排版可能与手写格式不同。

## 相关命令

[plan](/commands/plan) 可以检查当前配置映射及授权，但不是解密预演，也不验证当前身份能否解密；[view](/commands/view) 可以把明文打印到标准输出，[diff](/commands/diff) 可以比较明文和加密文件。
