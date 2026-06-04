# view - 查看加密文件

将解密后的明文输出到标准输出，不写入文件。

## 语法

```bash
yews view [选项] <target>
```

## 参数

### target

要查看的加密文件路径。

```bash
yews view config.enc.toml.yaml
```

## 选项

### --format

为选定的目标指定格式覆盖（toml/yaml/json/env/ini）。

```bash
yews view config.enc.yaml --format toml
```

### --verbose, -v

启用详细输出。

```bash
yews view config.enc.toml.yaml -v
```

## 示例

### 查看加密文件

```bash
# 查看 YAML 格式的加密文件
yews view config.enc.toml.yaml

# 查看并转换为 TOML 格式
yews view config.enc.yaml --format toml

# 查看并转换为 JSON 格式
yews view config.enc.yaml --format json
```

### 管道操作

```bash
# 查看并使用 grep 过滤
yews view config.enc.toml.yaml | grep database

# 查看并使用 jq 处理（需要先转换为 JSON）
yews view config.enc.yaml --format json | jq '.database'

# 查看并保存到文件
yews view config.enc.toml.yaml > config.toml
```

### 格式转换

```bash
# YAML 转 TOML
yews view config.enc.yaml --format toml

# YAML 转 JSON
yews view config.enc.yaml --format json

# YAML 转 ENV
yews view config.enc.yaml --format env
```

## 输出格式

输出格式由以下方式决定：

1. 如果指定了 `--format`，使用指定的格式
2. 否则使用加密文件中存储的原始格式

## 与 decrypt 的区别

| 命令 | 输出位置 | 文件写入 | 覆盖保护 |
|------|---------|---------|---------|
| `view` | 标准输出 | 否 | 不适用 |
| `decrypt` | 文件 | 是 | 是（默认） |

使用场景：
- **view**：快速查看内容、管道处理、临时查看
- **decrypt**：持久化解密、批量处理、格式转换

## 注意事项

1. **只读操作**：`view` 命令不会修改任何文件
2. **格式转换**：支持在查看时转换格式，但不影响原始加密文件
3. **管道友好**：输出到标准输出，方便与其他命令组合使用
4. **密钥要求**：需要对应的 Age 私钥才能解密

## 相关命令

- [edit](/commands/edit) - 编辑加密文件
- [decrypt](/commands/decrypt) - 解密文件到磁盘
- [diff](/commands/diff) - 比较明文和加密文件
