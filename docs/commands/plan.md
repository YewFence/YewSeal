# plan - 预览文件选择

`plan` 会运行预检并打印解析后的文件选择，不会加密、解密或写入文件。

## 语法

```bash
yews plan [command options] [path]
```

## 目标选择

不传 `path` 时，`plan` 会显示当前目录范围内的配置选择。传入明文文件、加密文件或目录时，目标仍必须来自已加载的 FilePair 或 Group；plan 会展示最终映射、格式来源、recipient alias、canonical recipient、registry 来源和 effective authorization source。

Plan 复用 encrypt 的严格授权 preflight。即使目标是 encrypted 路径，未知 alias、raw recipient、空授权集合或 Group 冲突也会直接失败。

## 选项

### --output, -o

为单文件目标预览输出路径。

```bash
yews plan config.toml -o config.enc.toml
```

### --format

为单文件目标预览格式覆盖。

```bash
yews plan .dev.vars --format env
```

### --format-rule

为目录或配置分组预览格式规则。

```bash
yews plan ./configs --format-rule ".dev.vars=env"
```

### --pattern

为目录或配置模式筛选文件。

```bash
yews plan ./configs --pattern "*.toml"
```

### --unknown-as-binary

预览未知格式按二进制处理后的选择结果。

```bash
yews plan ./secrets --unknown-as-binary
```

### --parallel, -P

设置预览输出中的并行数，默认值是 `1`。

```bash
yews plan --parallel 4
```

### --json

以 JSON 格式输出预检结果。

```bash
yews plan --json
```

### --verbose, -v

输出已加载配置文件等详细信息。

```bash
yews plan -v
```

## 示例

```bash
# 查看配置模式会选择哪些文件
yews plan

# 查看单文件加密会写到哪里
yews plan config.toml

# 查看目录扫描结果
yews plan ./configs --pattern "*.toml"

# 输出 JSON 供脚本读取
yews plan --json
```

## 相关命令

[encrypt](/commands/encrypt) 和 [decrypt](/commands/decrypt) 使用同一套目标选择逻辑。
