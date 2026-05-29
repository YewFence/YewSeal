# decrypt - 解密配置文件

解密加密的配置文件，输出格式由扩展名决定。

## 语法

```bash
yews decrypt [选项]
```

别名：`d`

## 模式

### 单文件模式

解密单个文件：

```bash
yews decrypt -i config.enc.toml.yaml -o config.toml
```

### 批量模式

解密目录中的多个文件：

```bash
yews decrypt --dir ./encrypted --pattern "*.enc.toml.yaml"
```

## 选项

### --input, -i

输入加密文件路径（单文件模式）。

默认值：`wrangler.enc.toml.yaml`

```bash
yews decrypt -i config.enc.toml.yaml
```

### --output, -o

输出解密文件路径（仅单文件模式）。

默认值：`wrangler.toml`

```bash
yews decrypt -i config.enc.toml.yaml -o config.toml
```

### --format

格式覆盖（toml/yaml/json/env/ini），用于单文件模式。

```bash
yews decrypt -i config.enc.yaml --format toml -o config.toml
```

### --force, -f

当明文文件存在且内容不同时，强制覆盖。

```bash
yews decrypt -i config.enc.toml.yaml -o config.toml --force
```

### --dir

要扫描加密文件的目录（启用批量模式）。

```bash
yews decrypt --dir ./encrypted
```

### --pattern

匹配加密文件的 Glob 模式。

默认值：`*.enc.toml.yaml`

```bash
yews decrypt --dir ./encrypted --pattern "*.encrypted.yaml"
```

### --output-dir

解密文件的输出目录（批量模式）。

```bash
yews decrypt --dir ./encrypted --output-dir ./configs
```

### --output-suffix

输出文件的后缀（批量模式）。

默认值：`.toml`

```bash
yews decrypt --dir ./encrypted --output-suffix ".yaml"
```

### --parallel, -P

批量模式的并行工作线程数。

默认值：`1`

```bash
yews decrypt --dir ./encrypted --parallel 4
```

### --verbose, -v

启用详细输出。

```bash
yews decrypt -i config.enc.toml.yaml -v
```

## 示例

### 单文件解密

```bash
# 使用默认文件名
yews decrypt

# 指定输入输出
yews decrypt -i app.enc.toml.yaml -o app.toml

# 指定输出格式
yews decrypt -i config.enc.yaml --format json -o config.json

# 强制覆盖
yews decrypt -i config.enc.toml.yaml -o config.toml --force
```

### 批量解密

```bash
# 解密目录中所有加密文件
yews decrypt --dir ./encrypted

# 使用自定义模式
yews decrypt --dir ./encrypted --pattern "*.encrypted.yaml"

# 指定输出目录
yews decrypt --dir ./encrypted --output-dir ./configs

# 并行处理
yews decrypt --dir ./encrypted --parallel 4 -v
```

## 输出格式

解密后的输出格式由以下方式决定：

1. 如果指定了 `--format`，使用指定的格式
2. 否则根据输出文件扩展名决定：
   - `.toml` → TOML
   - `.yaml`, `.yml` → YAML
   - `.json` → JSON
   - `.env` → ENV
   - `.ini` → INI

## 覆盖保护

默认情况下，如果输出文件已存在且内容与解密结果不同，`decrypt` 命令会报错并拒绝覆盖。

使用 `--force` 选项可以强制覆盖：

```bash
yews decrypt -i config.enc.toml.yaml -o config.toml --force
```

## 格式转换

YewSeal 支持在解密时进行格式转换：

```bash
# 将加密的 YAML 解密为 TOML
yews decrypt -i config.enc.yaml --format toml -o config.toml

# 将加密的 YAML 解密为 JSON
yews decrypt -i config.enc.yaml --format json -o config.json

# 将加密的 YAML 解密为 ENV
yews decrypt -i config.enc.yaml --format env -o .env
```

## 批量模式文件名

批量模式下，输出文件名的生成规则：

1. 移除输入文件的 `--pattern` 匹配部分
2. 添加 `--output-suffix` 后缀

示例：

```bash
# 输入：config.enc.toml.yaml
# 模式：*.enc.toml.yaml
# 后缀：.toml
# 输出：config.toml

yews decrypt --dir ./encrypted --pattern "*.enc.toml.yaml" --output-suffix ".toml"
```

## 注意事项

1. **密钥要求**：解密需要对应的 Age 私钥，通过 `--key-file` 或 `SOPS_AGE_KEY_FILE` 环境变量指定
2. **格式转换**：YewSeal 支持在解密时转换格式，但需要确保数据结构兼容
3. **并行处理**：使用 `--parallel` 可以加速批量解密，建议值为 CPU 核心数
4. **覆盖保护**：默认不会覆盖已存在的文件，使用 `--force` 强制覆盖

## 相关命令

- [encrypt](/commands/encrypt) - 加密配置文件
- [edit](/commands/edit) - 编辑加密文件
- [view](/commands/view) - 查看加密文件内容
- [diff](/commands/diff) - 比较明文和加密文件
