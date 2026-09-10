# 配置说明

YewSeal 使用 `.yewseal.toml` 管理文件映射、按文件授权、Age 私钥位置和密钥同步配置，使用 `.sops.yaml` 提供给 SOPS 直接运行时的完全托管加密规则，默认使用 `.age/keys.txt` 存储 Age 私钥。

## 配置加载顺序

YewSeal 会从当前 Git 仓库根目录开始，一路加载到当前目录，每一层只选择优先级最高的一个配置文件。单个目录内的优先级是 `.yewseal/.yewseal.toml` 高于 `.config/.yewseal.toml` 高于 `.yewseal.toml`。

子目录配置会覆盖或追加上层配置。`[[encryption.files]]` 会按明文路径或加密路径去重后覆盖，`[[encryption.groups]]` 会追加，`key` 和 `sync` 字段会以后加载的非空值为准。

## .yewseal.toml

### 编辑器补全与校验

YewSeal 为 `.yewseal.toml` 维护一份 JSON Schema，由仓库中的 `schema/config.cue` 生成，CI 保证其与实现同步。使用 Taplo 或 VSCode 的 Even Better TOML 扩展时，在配置文件顶部加一行注释关联 schema，即可获得自动补全、悬停文档和即时校验：

```toml
#:schema https://raw.githubusercontent.com/YewFence/YewSeal/main/schema/yewseal.schema.json

[encryption]
```

仓库中的 [schema/example.yewseal.toml](https://github.com/YewFence/YewSeal/blob/main/schema/example.yewseal.toml) 是覆盖全部字段的完整示例，同样由 CI 保证不过期。后续计划将 schema 提交到 SchemaStore，收录后编辑器会自动识别 `.yewseal.toml`，无需手动关联。

### 基本结构

```toml
[key]
file_path = ".age/keys.txt"

[recipients]
defaults = ["owner"]

[recipients.registry]
owner = "age1xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"

[[encryption.files]]
plaintext = "config.toml"
encrypted = "config.enc.toml"
```
所有运行时处理的路径都必须来自显式 FilePair 或 Group。没有配置文件、空配置或未登记 target 时不会自动生成 wrangler FilePair；默认私钥文件仍是 `.age/keys.txt`。

### Recipient 授权

`[recipients.registry]` 把可审查的 alias 映射到单个公开 Age recipient。alias 区分大小写，只允许以 ASCII 字母开头，并由字母、数字、下划线或连字符组成；同名 alias、同一公钥对应多个 alias、非法公钥都会报错。registry 中不能保存私钥。

FilePair 和 Group 的 `recipients` 只接受 alias。有效集合按“显式 FilePair > 命中 Group > `recipients.defaults`”选择，每一层都是完整替换而不是并集。显式空数组会清除继承，但 encrypt 和 plan 最终会因空集合失败。

同一路径命中多个 Group 时，各 Group 解析后的 canonical recipient 集合必须相同；否则报冲突。存在同路径显式 FilePair 时，以显式项为最终裁决。recipient 按公钥排序后传给 SOPS，配置中的 alias 顺序不改变授权语义。

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

`format` 是可选字段，支持 `toml`、`yaml`、`json`、`env`、`ini` 和 `binary`（也接受别名 `yml`、`dotenv`、`bin`，运行时会归一化为规范名），适合 `.dev.vars` 这种无法从扩展名判断格式的文件。

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

`patterns` 支持 `*`、`?`、`**`、以 `!` 开头的排除规则、以 `/` 开头的根目录锚定规则和以 `/` 结尾的目录规则。加密 Group 始终排除 YewSeal 协议格式的 `.enc.toml`、`.enc.yaml`、`.enc.json`、`.enc.env`、`.enc.ini` 和 `.enc.bin` 文件；同时排除配置中显式 FilePair 的 `encrypted` 路径。没有配置 `patterns` 时，加密会默认扫描 `.toml`、`.yaml`、`.yml`、`.json`、`.env`、`.ini`、`.bin`、`.binary`，解密会默认扫描上述 `.enc.*` 文件。

`format_rules` 使用 `<pattern>=<format>` 形式，会按匹配顺序决定格式。格式取值与 `format` 相同。`unknown_as_binary` 为 `true` 时，分组加密中无法识别格式的文件会按二进制文件处理。

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

1. 显式全局选项 `--key-file` / `-k`，或 `AGE_KEY_FILE`
2. `YEWSEAL_AGE_IDENTITIES` 环境变量中的逗号分隔 identity bundle
3. `SOPS_AGE_KEY` 环境变量中的完整多行 bundle
4. `SOPS_AGE_KEY_FILE` 环境变量
5. `SOPS_AGE_KEY_CMD` 环境变量
6. `.yewseal.toml` 的 `[key].file_path`
7. 默认路径 `.age/keys.txt`

```bash
yews --key-file ~/.age/my-key.txt decrypt config.enc.toml
```

```toml
[key]
file_path = ".age/keys.txt"
```

加密时唯一使用 `[recipients.registry]` 与 FilePair、Group 或 `recipients.defaults` 解析出的 canonical Age recipient。私钥文件、`SOPS_AGE_RECIPIENTS` 和 `.sops.yaml` 都不会决定 YewSeal 的加密授权范围。

### Identity bundle

一个 key file 可以包含多把 Age 私钥；YewSeal 会忽略注释、空行和无关行，并按首次出现顺序去重。CI 也可以使用专用变量传入逗号分隔的多把私钥：

```bash
YEWSEAL_AGE_IDENTITIES='AGE-SECRET-KEY-1...,AGE-SECRET-KEY-1...' yews decrypt config.enc.toml
```

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
| `AGE_KEY_FILE` | 全局 `--key-file` 的默认值，按显式 key file 处理 |
| `YEWSEAL_AGE_IDENTITIES` | 逗号分隔的 Age identity bundle |
| `SOPS_AGE_KEY` | 完整多行 Age identity bundle |
| `SOPS_AGE_KEY_FILE` | Age 私钥文件路径 |
| `SOPS_AGE_KEY_CMD` | 执行命令获取 Age identity bundle |
| `SOPS_OUTPUT_FILE` | `encrypt`、`decrypt`、`plan` 的 `--output` 值 |
| `YEWSEAL_FORMAT` | `encrypt`、`decrypt`、`plan` 的 `--format` 值 |
| `SOPS_FORMAT` | `encrypt`、`decrypt`、`plan` 的 `--format` 值 |
 `EDITOR` | `edit` 命令在 `VISUAL` 未设置时使用的编辑器 |
 `VISUAL` | `edit` 命令优先使用的编辑器 |

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
