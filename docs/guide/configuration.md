# 配置说明

YewSeal 使用 `.yewseal.toml` 管理文件映射、Age 密钥位置、Age 公钥和密钥同步配置，使用 `.sops.yaml` 提供给 SOPS 直接运行时的加密规则，默认使用 `.age/keys.txt` 存储 Age 私钥。

## 配置加载顺序

YewSeal 会从当前 Git 仓库根目录开始，一路加载到当前目录，每一层只选择优先级最高的一个配置文件。单个目录内的优先级是 `.yewseal/.yewseal.toml` 高于 `.config/.yewseal.toml` 高于 `.yewseal.toml`。

子目录配置会覆盖或追加上层配置。`[[encryption.files]]` 会按明文路径或加密路径去重后覆盖，`[[encryption.groups]]` 会追加，`key` 和 `sync` 字段会以后加载的非空值为准。

## .yewseal.toml

### 基本结构

```toml
[key]
file_path = ".age/keys.txt"
public_key = "age1xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"

[[encryption.files]]
plaintext = "config.toml"
encrypted = "config.enc.toml"
```

如果没有配置文件，YewSeal 会使用默认映射 `wrangler.toml` 到 `wrangler.enc.toml`，默认私钥文件是 `.age/keys.txt`。

### 文件映射

`[[encryption.files]]` 用来声明一个明文文件和一个加密文件的对应关系：

```toml
[[encryption.files]]
plaintext = "config.toml"
encrypted = "config.enc.toml"

[[encryption.files]]
plaintext = ".dev.vars"
encrypted = ".dev.enc.env"
format = "env"
```

`format` 是可选字段，支持 `toml`、`yaml`、`json`、`env`、`ini` 和 `binary`，适合 `.dev.vars` 这种无法从扩展名判断格式的文件。

### 分组扫描

`[[encryption.groups]]` 用来按模式扫描一批文件：

```toml
[[encryption.groups]]
patterns = [
  "*.toml",
  "*.yaml",
  "secrets/**/*.json",
  "!*.enc.toml",
  "!*.enc.yaml",
  "!*.enc.json",
]
format_rules = [
  ".dev.vars=env",
  "secrets/*.conf=ini",
]
unknown_as_binary = false
```

`patterns` 支持 `*`、`?`、`**`、以 `!` 开头的排除规则、以 `/` 开头的根目录锚定规则和以 `/` 结尾的目录规则。没有配置 `patterns` 时，加密会默认扫描 `.toml`、`.yaml`、`.yml`、`.json`、`.env`、`.ini`、`.bin`、`.binary`，并排除常见的 `.enc.*` 文件，解密会默认扫描 `.enc.toml`、`.enc.yaml`、`.enc.json`、`.enc.env`、`.enc.ini` 和 `.enc.bin`。

`format_rules` 使用 `<pattern>=<format>` 形式，会按匹配顺序决定格式，命令行传入的 `--format-rule` 会追加到配置规则之后。`unknown_as_binary` 为 `true` 时，分组加密中无法识别格式的文件会按二进制文件处理。

## .sops.yaml 配置

`.sops.yaml` 是 SOPS 的配置文件，YewSeal 的 `init` 和 `encrypt` 会根据当前文件映射同步它。跳过 `.sops.yaml` 不影响 YewSeal 自身通过内嵌 SOPS 引擎加解密，但直接运行 `sops` 时会更方便。

```yaml
creation_rules:
  - path_regex: ^config\.enc\.toml$
    age: age1xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

YewSeal 生成的是针对每个加密文件的精确匹配规则。直接运行 `sops` 新建加密文件时，也可以手写更宽松的正则。

### 多环境配置

可以为不同的文件模式配置不同的密钥：

```yaml
creation_rules:
  # 生产环境配置
  - path_regex: \.prod\.enc\.toml$
    age: age1prod_key_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx

  # 开发环境配置
  - path_regex: \.dev\.enc\.toml$
    age: age1dev_key_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx

  # 默认规则
  - path_regex: \.enc\.toml$
    age: age1default_key_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

### 多密钥配置

可以为同一个文件配置多个密钥（多人协作场景）：

```yaml
creation_rules:
  - path_regex: \.enc\.toml$
    age: >-
      age1key1_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx,
      age1key2_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx,
      age1key3_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

## Age 密钥管理

### 私钥读取

解密时，Age 私钥按以下优先级解析（从高到低）：

1. 全局选项 `--key-file` / `-k`（默认值来自 `AGE_KEY_FILE` 环境变量）
2. `SOPS_AGE_KEY` 环境变量（直接传私钥值，适合 CI/CD）
3. `SOPS_AGE_KEY_FILE` 环境变量（私钥文件路径）
4. `SOPS_AGE_KEY_CMD` 环境变量（执行命令获取私钥）
5. `.yewseal.toml` 的 `[key].file_path`
6. 默认路径 `.age/keys.txt`

```bash
yews --key-file ~/.age/my-key.txt decrypt config.enc.toml
```

```toml
[key]
file_path = ".age/keys.txt"
public_key = "age1xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
```

`public_key` 是公钥，可以提交到仓库，`file_path` 指向的私钥文件不能提交。

### 公钥读取

加密时，Age 公钥按以下优先级解析（从高到低）：

1. `encrypt` 的 `--public-key` / `-p` 选项
2. `SOPS_AGE_RECIPIENTS` 环境变量
3. `.yewseal.toml` 的 `[key].public_key`
4. 从私钥文件的注释中提取

## 密钥同步

YewSeal 支持将密钥同步到密钥管理服务，目前支持 Infisical。

可以把同步参数写进 `.yewseal.toml`：

```toml
[sync]
provider = "infisical"
project_id = "your-project-id"
environment = "dev"
path = "/yewseal"
secret_name = "AGE_KEY_FILE"
```

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

## 环境变量

YewSeal 支持通过环境变量配置部分选项：

| 环境变量 | 用途 |
| --- | --- |
| `AGE_KEY_FILE` | 全局 `--key-file` 的默认值 |
| `SOPS_AGE_KEY` | 直接提供 Age 私钥值 |
| `SOPS_AGE_KEY_FILE` | Age 私钥文件路径 |
| `SOPS_AGE_KEY_CMD` | 执行命令获取 Age 私钥 |
| `SOPS_AGE_RECIPIENTS` | `encrypt` 的 `--public-key` 值 |
| `SOPS_OUTPUT_FILE` | `encrypt`、`decrypt`、`plan` 的 `--output` 值 |
| `YEWSEAL_FORMAT` | `encrypt`、`decrypt`、`plan` 的 `--format` 值 |
| `SOPS_FORMAT` | `encrypt`、`decrypt`、`plan` 的 `--format` 值 |
| `EDITOR` | `edit` 命令未传 `--editor` 时使用的编辑器 |
| `VISUAL` | `edit` 命令在 `EDITOR` 未设置时使用的编辑器 |

## 最佳实践

### 密钥安全

不要提交 `.age/keys.txt`，团队协作时可以用 `yews sync` 和 `yews sync pull` 通过 Infisical 分发私钥，轮换密钥后重新运行 `encrypt` 同步 `.sops.yaml` 和加密文件。

### 文件命名约定

建议使用以下命名约定：

| 明文格式 | 加密文件命名 |
| --- | --- |
| `config.toml` | `config.enc.toml` |
| `config.yaml` | `config.enc.yaml` |
| `config.json` | `config.enc.json` |
| `.env` | `.env.enc.env` |
| `config.ini` | `config.enc.ini` |
| `secret.bin` | `secret.enc.bin` |

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
!.sops.yaml
!.yewseal.toml
!*.enc.toml
!*.enc.yaml
!*.enc.json
!*.enc.env
!*.enc.ini
!*.enc.bin
```
