# init - 初始化项目

初始化项目，生成 Age 密钥对和 YewSeal 配置。

## 语法

```bash
yews init [选项]
```

## 选项

### --create-example

创建示例文件（非交互模式）。

```bash
yews init --create-example
```

### --force, -f

强制覆盖已存在的配置。

```bash
yews init --force
```

### --format

为第一个配置条目指定格式覆盖（toml/yaml/json/env/ini）。

```bash
yews init --format toml
```

### --input, -i

为第一个配置条目指定明文文件（非交互模式）。

```bash
yews init --input config.toml
```

### --output, -o

为第一个配置条目指定加密文件（非交互模式）。

```bash
yews init --output config.enc.toml.yaml
```

### --skip-sops-config

跳过创建 `.sops.yaml` 文件（非交互模式）。

```bash
yews init --skip-sops-config
```

## 交互模式

默认情况下，`init` 命令以交互模式运行，会提示用户输入配置信息：

```bash
yews init
```

交互流程：
1. 检查是否已存在 Age 密钥，如果不存在则生成新密钥
2. 询问是否创建 `.sops.yaml` 配置文件
3. 询问是否添加第一个配置文件条目
4. 如果添加配置条目，询问明文文件路径、加密文件路径和格式

## 非交互模式

通过提供所有必需的选项，可以在非交互模式下运行：

```bash
yews init \
  --input config.toml \
  --output config.enc.toml.yaml \
  --format toml
```

## 示例

### 基本初始化

```bash
# 交互式初始化
yews init
```

### 完全非交互初始化

```bash
# 创建示例文件
yews init --create-example

# 指定第一个配置文件
yews init \
  --input wrangler.toml \
  --output wrangler.enc.toml.yaml \
  --format toml
```

### 强制重新初始化

```bash
# 覆盖已存在的配置
yews init --force
```

### 只生成密钥

```bash
# 跳过 .sops.yaml 创建
yews init --skip-sops-config
```

## 生成的文件

### .age/keys.txt

Age 私钥文件，格式如下：

```text
# created: 2024-01-01T00:00:00Z
# public key: age1xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
AGE-SECRET-KEY-1XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
```

**重要**：该文件包含私钥，不应提交到版本控制系统。

### .sops.yaml

SOPS 配置文件，定义加密规则：

```yaml
creation_rules:
  - path_regex: \.enc\.toml\.yaml$
    age: age1xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

## 注意事项

1. **密钥安全**：生成的 `.age/keys.txt` 包含私钥，务必添加到 `.gitignore`
2. **覆盖保护**：默认情况下不会覆盖已存在的密钥和配置，使用 `--force` 强制覆盖
3. **公钥分发**：可以从 `.age/keys.txt` 中提取公钥分发给团队成员用于加密

## 相关命令

- [encrypt](/commands/encrypt) - 加密配置文件
- [decrypt](/commands/decrypt) - 解密配置文件
- [sync](/commands/sync) - 同步密钥到密钥管理服务
