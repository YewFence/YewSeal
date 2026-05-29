# encrypt - 加密配置文件

加密配置文件，支持 .toml、.yaml、.yml、.json、.env、.ini 格式。

## 语法

```bash
yews encrypt [选项]
```

别名：`e`

## 模式

### 单文件模式

加密单个文件：

```bash
yews encrypt -i config.toml -o config.enc.toml.yaml
```

### 批量模式

加密目录中的多个文件：

```bash
yews encrypt --dir ./configs --pattern "*.toml"
```

## 选项

### --input, -i

输入文件路径（单文件模式）。

默认值：`wrangler.toml`

```bash
yews encrypt -i config.toml
```

### --output, -o

输出加密文件路径（仅单文件模式）。

默认值：`wrangler.enc.toml.yaml`

```bash
yews encrypt -i config.toml -o config.enc.toml.yaml
```

### --format

格式覆盖（toml/yaml/json/env/ini），用于单文件模式。

```bash
yews encrypt -i config.txt --format toml
```

### --public-key, -p

用于加密的 Age 公钥。

```bash
yews encrypt -i config.toml -p age1xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

### --dir

要扫描的目录（启用批量模式）。

```bash
yews encrypt --dir ./configs
```

### --pattern

目录中匹配文件的 Glob 模式。

默认值：`*.toml`

```bash
yews encrypt --dir ./configs --pattern "*.yaml"
```

### --output-dir

加密文件的输出目录（批量模式）。

```bash
yews encrypt --dir ./configs --output-dir ./encrypted
```

### --output-suffix

输出文件的后缀（批量模式）。

默认值：`.enc.toml.yaml`

```bash
yews encrypt --dir ./configs --output-suffix ".encrypted.yaml"
```

### --parallel, -P

批量模式的并行工作线程数。

默认值：`1`

```bash
yews encrypt --dir ./configs --parallel 4
```

### --verbose, -v

启用详细输出。

```bash
yews encrypt -i config.toml -v
```

## 示例

### 单文件加密

```bash
# 使用默认文件名
yews encrypt

# 指定输入输出
yews encrypt -i app.toml -o app.enc.toml.yaml

# 指定格式
yews encrypt -i config.txt --format json -o config.enc.json.yaml
```

### 批量加密

```bash
# 加密目录中所有 .toml 文件
yews encrypt --dir ./configs

# 使用自定义模式
yews encrypt --dir ./configs --pattern "*.yaml"

# 指定输出目录
yews encrypt --dir ./configs --output-dir ./encrypted

# 并行处理
yews encrypt --dir ./configs --parallel 4 -v
```

### 使用特定公钥

```bash
# 使用指定的公钥加密
yews encrypt \
  -i config.toml \
  -o config.enc.toml.yaml \
  -p age1xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

## 加密格式

YewSeal 将所有配置文件加密为 YAML 格式（SOPS 标准格式）：

```yaml
# 原始 TOML 文件
[database]
host = "localhost"
password = "secret"

# 加密后的 YAML 文件
database:
    host: localhost
    password: ENC[AES256_GCM,data:xxx,iv:xxx,tag:xxx,type:str]
sops:
    age:
        - recipient: age1xxx
          enc: |
            -----BEGIN AGE ENCRYPTED FILE-----
            ...
            -----END AGE ENCRYPTED FILE-----
```

## 格式检测

YewSeal 通过以下方式检测文件格式：

1. 如果指定了 `--format`，使用指定的格式
2. 否则根据文件扩展名检测：
   - `.toml` → TOML
   - `.yaml`, `.yml` → YAML
   - `.json` → JSON
   - `.env` → ENV
   - `.ini` → INI

## 注意事项

1. **TOML 支持**：TOML 文件会先转换为 YAML，然后加密为 YAML 格式
2. **批量模式**：批量模式下，输出文件名为 `原文件名 + output-suffix`
3. **并行处理**：使用 `--parallel` 可以加速批量加密，建议值为 CPU 核心数
4. **公钥来源**：如果不指定 `--public-key`，会从 `.sops.yaml` 中读取

## 相关命令

- [decrypt](/commands/decrypt) - 解密配置文件
- [edit](/commands/edit) - 编辑加密文件
- [view](/commands/view) - 查看加密文件内容
- [init](/commands/init) - 初始化项目
