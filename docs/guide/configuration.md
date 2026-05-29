# 配置说明

YewSeal 使用 `.sops.yaml` 文件配置加密规则，使用 `.age/keys.txt` 存储 Age 密钥。

## .sops.yaml 配置

`.sops.yaml` 文件定义了加密规则和密钥配置。YewSeal 在 `init` 命令时会自动创建该文件。

### 基本结构

```yaml
creation_rules:
  - path_regex: \.enc\.toml\.yaml$
    age: age1xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

### 多环境配置

可以为不同的文件模式配置不同的密钥：

```yaml
creation_rules:
  # 生产环境配置
  - path_regex: \.prod\.enc\.toml\.yaml$
    age: age1prod_key_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
  
  # 开发环境配置
  - path_regex: \.dev\.enc\.toml\.yaml$
    age: age1dev_key_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
  
  # 默认规则
  - path_regex: \.enc\.toml\.yaml$
    age: age1default_key_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

### 多密钥配置

可以为同一个文件配置多个密钥（多人协作场景）：

```yaml
creation_rules:
  - path_regex: \.enc\.toml\.yaml$
    age: >-
      age1key1_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx,
      age1key2_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx,
      age1key3_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

## Age 密钥管理

### 密钥文件位置

默认情况下，YewSeal 将 Age 私钥存储在项目根目录的 `.age/keys.txt` 文件中。

可以通过全局选项 `--key-file` 或 `-k` 指定其他位置：

```bash
yews --key-file ~/.age/my-key.txt decrypt -i config.enc.toml.yaml
```

### 密钥文件格式

Age 密钥文件格式如下：

```
# created: 2024-01-01T00:00:00Z
# public key: age1xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
AGE-SECRET-KEY-1XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
```

### 获取公钥

从私钥文件中提取公钥：

```bash
grep "public key:" .age/keys.txt | cut -d: -f2 | tr -d ' '
```

或者使用 `age-keygen` 工具：

```bash
age-keygen -y .age/keys.txt
```

## 密钥同步

YewSeal 支持将密钥同步到密钥管理服务，目前支持 Infisical。

### 推送密钥到 Infisical

```bash
yews sync \
  --provider infisical \
  --project-id <project-id> \
  --env dev \
  --path /yewseal \
  --name AGE_KEY_FILE \
  --key-file .age/keys.txt
```

### 从 Infisical 拉取密钥

```bash
yews sync pull \
  --provider infisical \
  --project-id <project-id> \
  --env dev \
  --path /yewseal \
  --name AGE_KEY_FILE \
  --key-file .age/keys.txt
```

## 全局选项

### --key-file, -k

指定 Age 私钥文件路径：

```bash
yews -k ~/.age/prod-key.txt encrypt -i config.toml
```

### --help, -h

显示帮助信息：

```bash
yews --help
yews encrypt --help
```

### --version, -v

显示版本信息：

```bash
yews --version
```

## 环境变量

YewSeal 支持通过环境变量配置部分选项：

- `SOPS_AGE_KEY_FILE` - Age 私钥文件路径（SOPS 标准环境变量）
- `EDITOR` - 默认编辑器（用于 `edit` 命令）

## 最佳实践

### 密钥安全

1. **不要提交私钥到版本控制**：将 `.age/` 目录添加到 `.gitignore`
2. **使用密钥管理服务**：在团队协作时使用 Infisical 等服务管理密钥
3. **定期轮换密钥**：定期更新 Age 密钥并重新加密配置文件

### 文件命名约定

建议使用以下命名约定：

- 明文文件：`config.toml`
- 加密文件：`config.enc.toml.yaml`

这样可以：
- 清晰区分明文和密文
- 利用 `.sops.yaml` 的 `path_regex` 规则
- 方便批量操作

### 版本控制

建议的 `.gitignore` 配置：

```gitignore
# Age 私钥
.age/

# 明文配置文件
*.toml
*.yaml
*.json
*.env
*.ini

# 但保留加密文件
!*.enc.toml.yaml
!*.enc.yaml
!*.enc.json
```
