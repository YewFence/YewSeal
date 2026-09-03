# 按文件授权的 Recipient 注册表设计

## 状态

本文档定义 YewSeal 的第一版按文件授权模型和相关配置重构方案。本文档已完成设计确认，后续实现应以本文档为准。


### 实现状态

本文档定义的第一版 recipient registry 已实现，并通过 `mise run check`。核心契约如下：

1. **配置与 provenance**：CUE、JSON Schema、完整示例和 Go 模型均已同步；FilePair/Group 的字段 presence、registry 定义来源、授权声明来源和最终 effective source 会保留到 resolved selection。
2. **严格授权解析**：file > group > defaults 的完整替换优先级已实现；canonical recipient 集合稳定排序；多个 Group 的 effective 集合冲突会失败，显式 FilePair 可对同路径冲突作最终裁决。
3. **全量 encrypt/plan preflight**：所有选中 pair 会在 metadata 或密文写入前完成未知 alias、raw recipient、空集合、非法公钥及 Group 冲突检查；plan 对 encrypted target 同样采用严格授权语义。
4. **旧入口删除**：`[key].public_key` 会返回迁移错误；`GetPublicKey`、`--public-key`、`SOPS_AGE_RECIPIENTS` fallback，以及 app/task/seal 中的单 public-key API 和私钥推导加密 recipient 逻辑均已删除。
5. **Identity bundle**：显式 key file、`YEWSEAL_AGE_IDENTITIES`、既有 SOPS source、配置 key file 和默认 key file 按优先级解析一次，去重后作为完整 bundle 供整个解密批次复用。
6. **Decrypt/Edit**：decrypt 遇到已失效 alias 时向 stderr 输出非致命 warning，并继续依据密文 metadata 解密；edit 必须命中配置，并保留原密文的完整 recipient 集合。
7. **Init 与 SOPS 配置**：init 写入 owner registry、defaults、显式 FilePair 及其 alias；`--force` 重建 key/policy/files，并在跳过 SOPS 配置时删除旧托管文件；key、主配置和 `.sops.yaml` 使用临时文件替换。
8. **可审查输出**：plan 的表格和 JSON 均展示 alias、canonical recipients、effective authorization source 和 registry 来源；`.sops.yaml` 按文件生成稳定、多 recipient、完全托管的规则。

当前文档后续章节仍保留原始设计依据、边界和验收标准，作为实现行为的规范说明。
本版本的核心目标是：让每个加密文件最终拥有清晰、可审查的 Age recipient 集合，同时通过默认授权集合保持 quickstart 的低摩擦体验。文件路径必须属于 YewSeal 配置模型，但用户不必为每个文件重复书写 `recipients`。

本文档暂不设计 `rekey`、数据密钥轮换和运行时临时明文交付。已有密文的 recipient 迁移、数据密钥轮换和运行时临时明文交付属于后续独立工作。

## 设计原则

1. 文件路径归属必须显式来自配置：`[[encryption.files]]` 或 `[[encryption.groups]]`。
2. 文件授权可以通过默认集合降低配置成本，但不能从私钥文件或任意 CLI 参数隐式推导。
3. 配置层负责把 alias 解析为 canonical Age recipient；SOPS 引擎只接收真实公钥。
4. Encrypt 在写入任何密文前完成所有选中文件的授权预检。
5. Decrypt 使用密文 metadata 作为历史密文的 recipient 事实，不依赖 alias 当前是否仍存在。
6. 私密 identity 与公开授权 registry 是两个独立领域，不能自动互相推导或同步修改。
7. recipient 集合是无序集合；所有 canonical 输出和比较使用稳定顺序。
8. 本次重构允许删除旧语义，不通过兼容层继续保留已经被否定的隐式授权行为。

## 背景

同一个项目中的不同配置文件通常不应该拥有相同的访问范围。例如开发配置可能允许开发者访问，生产配置可能只允许运维人员和备份身份访问。

直接在每个文件中写完整公钥虽然语义明确，但配置可读性较差，也不利于审查人员快速理解授权对象：

```toml
[[encryption.files]]
plaintext = "config/production.yaml"
encrypted = "config/production.enc.yaml"
recipients = [
  "age1xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
  "age1yyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy",
]
```

YewSeal 提供一个只包含公开信息的内联 recipient 注册表。文件配置只引用 alias，YewSeal 在进入 SOPS 引擎前将 alias 解析成规范化的 Age 公钥集合。

```toml
[recipients]
defaults = ["owner"]

[recipients.registry]
ops = "age1xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
backup = "age1yyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy"

[[encryption.files]]
plaintext = "config/production.yaml"
encrypted = "config/production.enc.yaml"
recipients = ["ops", "backup"]
```

对于不需要细分权限的 quickstart 项目，文件可以省略 `recipients`，继承顶层默认集合：

```toml
[recipients]
defaults = ["owner"]

[recipients.registry]
owner = "age1owner..."

[[encryption.files]]
plaintext = "config/development.yaml"
encrypted = "config/development.enc.yaml"
# recipients 省略，使用 ["owner"]
```

## 目标

1. 支持每个 `[[encryption.files]]` 声明自己独立的授权 recipient 集合。
2. 支持 `[[encryption.groups]]` 为扫描得到的文件提供授权默认值。
3. 使用短而稳定的 alias 提升 `.yewseal.toml` 的可读性和审查体验。
4. 提供一个可选的顶层默认授权集合，让普通项目不需要为每个文件重复配置。
5. 将 alias 解析集中在配置层，SOPS 和加密引擎只接收真实 Age recipient 公钥。
6. 在配置加载或 encrypt preflight 阶段发现未知 alias、非法公钥、空集合和重复定义。
7. 保持 recipient 集合的顺序无关语义，并使用稳定顺序输出和比较。
8. 允许一个消费者通过自己的 identity bundle 使用多把私钥解密多个权限范围不同的文件。
9. 不在项目配置中保存任何私钥，也不要求消费者知道文件使用了哪个 alias。
10. 为后续 `verify` 提供唯一、可复用的 recipient 解析结果。
11. 保留当前目录 scope 下的批量处理能力，同时禁止未配置路径成为临时加密目标。

## 非目标

1. 第一版不实现 `rekey`。
2. 第一版不实现 `--rotate-data-key`。
3. 第一版不实现 alias 到多个 recipient 的分组映射。
4. 第一版不实现多个命名默认 profile。
5. 第一版不实现从 recipient alias 反向查找或自动选择本地私钥。
6. 第一版不将私钥路径、私钥内容或 Secret Manager 引用放入 recipient registry。
7. 第一版不扫描 Git 历史中的曾经泄露内容。
8. 第一版不新增完整的 `verify` CLI，只提供后续 verify 可复用的解析和引擎接口。
9. 第一版不通过 `.sops.yaml` 作为 YewSeal 加密时的第二个授权事实来源。

## 领域术语

### Recipient

Age 公钥，例如 `age1...`。Recipient 是写入 SOPS metadata、用于声明授权主体的公开值。

### Recipient alias

recipient 的人类可读名称，例如 `ops`、`backup` 或 `alice`。Alias 只存在于 YewSeal 配置语义中，不写入 SOPS metadata。

### Recipient registry

将 alias 映射到一个 Age recipient 公钥的公开注册表。registry 不包含任何私钥。

### Default recipient set

顶层 `[recipients].defaults` 声明的默认 alias 集合。它是一个单一的可选集合，不是命名 profile 系统。

### File authorization set

某个文件最终允许访问的 alias 集合。解析后，它对应一个 canonical recipient 集合。

### Identity bundle

消费者实际拥有的一把或多把 Age 私钥的集合。Identity bundle 不属于项目授权配置；它代表当前消费者能够尝试使用的能力。

### Effective recipient set

在考虑 file、group 和 top-level defaults 的覆盖关系后，一个 resolved file pair 实际使用的 recipient 集合。

## 配置模型

### Identity source

`[key]` 只负责私密 identity source：

```toml
[key]
file_path = ".age/keys.txt"
```

`key` 不再包含 `public_key`。公开授权不属于 key source，而属于顶层 `recipients` 授权域。

### Recipient 注册表

第一版使用顶层 `[recipients.registry]` 保存 alias 到公钥的映射：

```toml
[recipients.registry]
owner = "age1owner..."
ops = "age1ops..."
backup = "age1backup..."
```

registry 的 value 必须是单个有效 Age recipient。第一版不允许 value 是数组：

```toml
# 第一版不支持
[recipients.registry]
ops = ["age1alice...", "age1bob..."]
```

如果一个文件需要允许多把公钥访问，直接在文件或 group 上列出多个 alias：

```toml
recipients = ["ops", "backup"]
```

### 顶层默认集合

顶层 `[recipients].defaults` 是一个可选的单一默认集合：

```toml
[recipients]
defaults = ["owner"]
```

它只用于降低 quickstart 配置成本。新增 registry alias 不会自动改变已有文件的默认授权范围；只有显式修改 defaults、group 或 file 配置时，文件的有效授权才会变化。

默认集合可以省略。如果用户希望每个敏感文件或 group 都显式声明授权，可以只配置 registry，并在每个授权层写 `recipients`。如果某个待加密文件最终没有任何有效授权来源，encrypt 必须失败。

### 文件授权集合

每个受控文件在 `[[encryption.files]]` 中可以声明 alias 集合：

```toml
[[encryption.files]]
plaintext = "config/development.yaml"
encrypted = "config/development.enc.yaml"
format = "yaml"
recipients = ["owner"]

[[encryption.files]]
plaintext = "config/production.yaml"
encrypted = "config/production.enc.yaml"
format = "yaml"
recipients = ["ops", "backup"]
```

`recipients` 的语义是完整替换集合，不是从全局配置或父配置逐项追加。显式 file-level 集合可以收窄 group 或 defaults 的授权范围。

FilePair 的 `recipients` 字段必须保留 presence 信息：

- 字段省略：继承 group 或 defaults；
- `recipients = []`：显式清除继承；允许被配置层识别，但 encrypt 最终必须因空集合失败；
- 非空列表：完全替换更低优先级的授权集合。

### Group 授权默认

Group 可以声明扫描结果的默认授权集合：

```toml
[[encryption.groups]]
patterns = ["config/*.yaml"]
recipients = ["dev"]
```

Group 省略 `recipients` 时继承顶层 defaults：

```toml
[[encryption.groups]]
patterns = ["config/**/*.yaml"]
# recipients 省略，继承 recipients.defaults
```

Group 的 `recipients` 也必须保留 presence 信息：

- 字段省略：继承 defaults；
- `recipients = []`：显式清除继承；encrypt 的最终解析必须因空集合失败；
- 非空列表：作为该 group 扫描结果的完整默认集合。

Group 不会把授权写回 `.yewseal.toml`。它只在 selection/resolution 阶段为扫描得到的具体 FilePair 提供来源明确的有效集合。

### Alias 命名

Alias 只允许 ASCII 字母、数字、下划线和连字符，并且必须以字母开头：

```text
[a-zA-Z][a-zA-Z0-9_-]*
```

以下值不作为 alias 接受：

```text
空字符串
包含空格的名称
包含点号的名称
包含逗号的名称
包含换行的名称
```

Alias 查找大小写敏感。`Ops` 和 `ops` 是两个不同的 alias；不会在查找时隐式转小写。

限制命名字符可以让错误信息、JSON 输出、shell 工具和未来的配置编辑命令保持简单一致。

### Alias 唯一性

合并后的配置中 alias 名称必须唯一。父目录和子目录配置中出现同名 alias 时直接报错，即使两个定义恰好指向同一个公钥也不自动合并或覆盖。

同一个规范化公钥被多个 alias 引用时直接报错。这样可以避免出现两个名字看似不同、实际授权主体完全相同的配置项，降低审查时的歧义。

### Alias 解析

文件 recipients 的每一项都按以下步骤解析：

```text
读取 alias
  ↓
检查命名格式
  ↓
在合并后的 recipient registry 中查找
  ↓
读取公钥并去除首尾空白
  ↓
验证 Age recipient 格式
  ↓
规范化并检查重复
  ↓
按公钥排序
  ↓
生成 canonical recipient 集合
```

未知 alias 必须失败，不允许把它当作 raw recipient 继续处理：

```text
config/production.yaml references unknown recipient alias "prod-ops"
```

第一版 file-level 和 group-level `recipients` 只接受 alias，不允许 alias 和 raw `age1...` 混写。这样拼写错误会尽早暴露，也不会让用户猜测某个字符串到底是 alias 还是公钥。

## 有效授权集合与覆盖规则

对一个具体文件，授权集合按以下优先级解析：

```text
显式 FilePair.recipients
  > 命中 Group 的有效 recipients
  > recipients.defaults
  > 无授权集合
```

最终规则：

1. 显式 FilePair 集合可以完全替换 group/defaults 集合。
2. Group 集合可以完全替换 defaults 集合。
3. 任何层级都不做并集，除非用户在该层显式列出全部需要的 alias。
4. FilePair/Group 的显式空数组可以清除继承，但 encrypt 最终必须拒绝空集合。
5. `recipients.defaults = []` 在配置校验阶段直接报错。
6. defaults 可以省略；如果最终没有有效集合，encrypt 和 plan 失败。
7. decrypt 不把当前有效授权集合当作历史密文的解密依据。

### 多 Group 命中

一个扫描文件可能同时被多个 group 命中。处理规则如下：

1. 分别解析每个 group 的有效集合，包括其对 defaults 的继承。
2. 如果所有命中的 group 的有效集合完全相同，合并为一个 FilePair。
3. 如果有效集合不同，直接报冲突错误；不按配置顺序选择，也不自动取并集。
4. 显式 FilePair 对同路径的扫描结果拥有最终裁决权，并可以完全替换 group/defaults 授权。
5. 相同路径不能生成多个实际加密任务。

示例：

```toml
[recipients]
defaults = ["owner"]

[[encryption.groups]]
patterns = ["config/*.yaml"]
# 有效集合为 ["owner"]

[[encryption.groups]]
patterns = ["*.yaml"]
recipients = ["ops"]
# 两个 group 命中同一文件时产生冲突
```

## 配置合并与来源

当前配置加载机制会按 Git 根目录到当前目录加载配置，并合并多份 `.yewseal.toml`。

### Registry 合并

所有已加载配置的 registry 合并为一个平面命名空间：

- alias 名称重复：报错；
- 即使公钥相同：仍报错；
- 不允许子目录覆盖父目录 alias；
- 每个 alias 记录其 registry 定义来源。

### Defaults 合并

`recipients.defaults` 只能在合并配置中定义一次：

- 父子配置都定义 defaults：报错；
- 不做替换；
- 不做并集；
- defaults 可以完全省略。

### FilePair 合并

同路径 FilePair 继续使用现有的路径覆盖模型：后加载的同路径 FilePair 替换先前的完整 FilePair。

替换是完整项替换，不做字段级继承。因此，子配置中的 FilePair 如果省略 `recipients`，它会重新按该 FilePair 所处的有效 group/defaults 解析，而不会自动继承父 FilePair 的 recipients。

### Resolved provenance

resolved FilePair 至少保留以下授权上下文：

1. 用户写出的 recipient alias 列表；
2. alias 对应 registry 定义的配置来源；
3. FilePair、Group 或 defaults 授权声明的配置来源；
4. 最终 effective recipient set 的来源类型和路径；
5. canonical Age recipient 列表。

下层 task、seal 和 sopsx 只消费 canonical recipients 和文件/格式信息，不依赖 alias 或配置 registry。

## 配置加载与校验阶段

`LoadConfig` 负责基础配置完整性，不必在所有命令上执行同样严格的最终授权解析。

### LoadConfig 负责

1. TOML 结构解析。
2. 顶层 `[recipients]` 和 `[recipients.registry]` 解析。
3. Alias 命名校验。
4. Registry alias 重复校验。
5. 公钥格式校验。
6. 重复规范化公钥校验。
7. 父子配置 defaults 重复校验。
8. 旧字段迁移错误检测。
9. 文件和 group 路径、格式等基础校验。
10. 保留 recipients 的 presence 信息和来源信息。

### Encrypt/Plan selection 负责

1. 根据当前 scope 和选择条件生成最终 FilePair 集合。
2. 解析每个 FilePair 的 effective recipient set。
3. 检查未知 alias、空授权集合和 group 冲突。
4. 生成 canonical recipient 列表。
5. 在 encrypt 中确保全部 pair 都通过后才允许写密文。
6. 在 plan 中展示最终授权结果。

### Decrypt 负责

Decrypt 仍必须通过配置选择 FilePair 的路径和格式，但当前 alias 无法解析时不阻止解密：

- 输出 stderr warning；
- warning 包含文件、alias 和配置来源；
- warning 不包含 identity 内容；
- 不比较当前配置集合与密文 metadata 集合；
- 继续使用密文 metadata 和 identity bundle 尝试解密。

## 配置优先级与兼容性

本次重构删除旧的隐式加密 recipient 语义。

### 删除的旧入口

以下内容不再参与新模型：

1. `[key].public_key`；
2. `--public-key`；
3. `SOPS_AGE_RECIPIENTS` 作为 YewSeal encrypt recipient 的 fallback；
4. file-level raw `age1...` recipient；
5. 从全局 public key 推导文件授权；
6. 从私钥文件第一条 identity 推导 encrypt recipient；
7. 从多 identity bundle 自动推导 encrypt recipient；
8. 未配置目标的临时 FilePair。

检测到旧 `[key].public_key`、旧 raw recipient 或其他已移除字段时直接报迁移错误，并提示使用 `[recipients.registry]`、`recipients.defaults` 和 alias 列表。不得静默忽略，也不得自动猜测迁移结果。

### 加密 recipient 的唯一来源

YewSeal encrypt 的 recipient 唯一来自：

```text
配置 registry
  ↓
FilePair / Group / defaults alias
  ↓
canonical Age recipients
  ↓
sopsx.Encrypt
```

私钥 source、identity bundle、SOPS 环境变量和 `.sops.yaml` 都不能决定 YewSeal encrypt 的授权范围。

## 文件选择与默认行为

### 无配置时

程序不再提供内置的 `DefaultFilePair` 运行时 fallback。

如果没有配置文件，或者配置文件存在但合并后没有任何 `files` 和 `groups`，需要返回明确错误。程序不会自动创建：

```text
wrangler.toml ↔ wrangler.enc.toml
```

### Init 生成的默认值

`init` 仍然使用 wrangler 名称作为首次配置的默认输入/输出值，但会把它们写入生成的 `.yewseal.toml`：

```toml
[[encryption.files]]
plaintext = "wrangler.toml"
encrypted = "wrangler.enc.toml"
```

这只是普通的显式 FilePair，不是程序内部默认对象。

### 无位置参数

`encrypt` 和 `decrypt` 不带位置参数时，语义是当前目录 scope 下全部已配置 pair：

- 从 Git 根目录执行：选择整个项目当前 scope 的配置项；
- 从子目录执行：只选择当前子树 scope 的配置项；
- group 扫描结果如果属于当前 scope，也可以被选中；
- 不需要增加 `--all` 参数。

### 有位置参数

显式 target 必须命中已加载配置中的 FilePair：

- 可以匹配 plaintext 路径；
- 可以匹配 encrypted 路径；
- 可以寻址当前 cwd scope 外、但仍属于已加载配置的 pair；
- 命中后使用完整 FilePair，不允许 target 改写另一侧路径或授权集合；
- 未命中配置时直接报错，不创建临时 FilePair。

命令方向决定实际读写方向：

```text
target ∈ {plaintext, encrypted}

encrypt: plaintext → encrypted
decrypt: encrypted → plaintext
```

### 批量与 pattern

现有 batch、pattern 和 parallel 能力可以保留，但它们只能处理配置中的 FilePair 或由配置 Group 生成的 pair。它们不能把任意文件变成临时授权对象。

## 消费者的多 Identity 输入

### 文件输入

`--key-file` 表示一个 identity bundle 文件，而不是只能包含一把私钥的文件：

```text
# .age/keys.txt
# alice
AGE-SECRET-KEY-1...

# ops
AGE-SECRET-KEY-1...
```

文件继续使用 Age 的逐行格式。Identity parser：

1. 收集所有可解析的 Age identity；
2. 忽略注释和空行；
3. 忽略其他无法识别的非 identity 行；
4. 对每个收集到的 identity 执行 Age parser 验证；
5. 最终至少需要一把有效 identity；
6. 按规范化 identity 文本去重；
7. 保持首次出现顺序；
8. 不输出 identity 内容。

如果文件中只有注释、空行或其他无效内容，返回明确的“没有有效 Age identity”错误。

### 环境变量输入

YewSeal 增加专用环境变量：

```text
YEWSEAL_AGE_IDENTITIES
```

该变量使用逗号分隔多把私钥：

```bash
YEWSEAL_AGE_IDENTITIES='AGE-SECRET-KEY-1...,AGE-SECRET-KEY-1...' yews decrypt
```

解析规则：

1. 按逗号切分；
2. 对每一项执行首尾空白清理；
3. 空项直接报错；
4. 每项验证为可用 Age identity；
5. 重复 identity 按规范化文本去重；
6. 内部转换为统一 IdentityBundle；
7. 不在任何输出中显示 identity 内容。

以下输入必须失败：

```text
,AGE-SECRET-KEY-1...
AGE-SECRET-KEY-1...,
AGE-SECRET-KEY-1...,,AGE-SECRET-KEY-1...
```

### SOPS 环境变量兼容

`SOPS_AGE_KEY` 是外部 SOPS 约定的变量。YewSeal 不改变其原有语义：

- `YEWSEAL_AGE_IDENTITIES` 才使用 YewSeal 定义的逗号集合语义；
- `SOPS_AGE_KEY` 继续按完整多行 bundle 语义处理；
- `SOPS_AGE_KEY_FILE` 和 `SOPS_AGE_KEY_CMD` 保持现有兼容语义；
- 不把 `SOPS_AGE_KEY` 重新解释为逗号列表。

### Identity source 优先级
建议优先级为：

```text
显式 --key-file
  > YEWSEAL_AGE_IDENTITIES
  > 既有 SOPS source 的原有优先级
  > 默认 .age/keys.txt
```

具体来说，除显式 `--key-file` 和 `YEWSEAL_AGE_IDENTITIES` 外，继续保持当前 SOPS source 的顺序：

```text
SOPS_AGE_KEY
  > SOPS_AGE_KEY_FILE
  > SOPS_AGE_KEY_CMD
  > 默认 .age/keys.txt
```

这里的列表表达 source 层级，而不是要求 YewSeal 重新解释 SOPS 变量的内容。`SOPS_AGE_KEY` 继续按完整多行 bundle 语义处理；只有 `YEWSEAL_AGE_IDENTITIES` 使用 YewSeal 定义的逗号集合语义。
如果用户显式传入 `--key-file`：

- 文件不存在、不可读或无法解析时直接失败；
- 不回退到环境变量或默认文件；
- 用户必须能明确知道实际使用的是哪一个 source。

对于新定义的 `YEWSEAL_AGE_IDENTITIES`，变量非空时也视为明确 source，解析失败直接失败。既有 SOPS source 的历史回退行为继续保持，不擅自改变外部 SOPS 兼容性。

### Identity 与文件授权的关系

消费者不需要传入 alias：

```bash
yews decrypt --key-file .age/keys.txt
```

YewSeal 对每个密文使用同一个 IdentityBundle 尝试解密。某个文件只要有一把 identity 能够匹配 metadata 中的 recipient，就可以通过解密检查。

例如：

```text
消费者 identity bundle：alice、ops、backup

文件 A recipients：alice
文件 B recipients：ops、backup
```

消费者可以解密 A 和 B，但文件 A 不会因为消费者额外持有 ops 私钥而允许 ops 访问。真正的授权边界由文件密文 metadata 中的 recipient 集合决定。

## 实现边界与分层

当前代码的 `task / seal` 分层仍然有职责，即使 fork 已支持原生 TOML。原生 TOML 支持消除了格式转换的必要，但不消除批处理和文件安全写入需求。

### 配置包

配置包负责：

1. 解析 recipient registry、defaults、files 和 groups；
2. 验证 alias、公钥和重复定义；
3. 保留字段 presence 和配置来源；
4. 解析每个文件的 effective authorization set；
5. 生成 canonical recipient 集合；
6. 暴露带有 alias、canonical recipients 和 provenance 的 resolved selection。

### App 层

App 层负责：

1. 命令级 preflight；
2. 当前 scope 和 target 选择；
3. encrypt 全量授权预检；
4. plan 输出；
5. 项目 metadata 更新。

### Task 层

Task 层负责：

1. 批量任务编排；
2. 并发处理；
3. 进度输出；
4. 单文件结果汇总；
5. 错误收集和最终退出错误。

Task 不解析 alias，也不读取 `.yewseal.toml`。

### Seal 层

Seal 层负责：

1. 单文件 I/O；
2. 格式推断和格式参数传递；
3. 加密输出写入；
4. 解密明文的覆盖保护和权限处理；
5. 把 canonical recipients 和 IdentityBundle 传递给 sopsx。

Seal 删除旧的 key file → public key 推导逻辑。Seal 不再从私钥文件中寻找加密公钥。

### SOPS facade

`internal/sopsx` 仍然是 engine facade，对外暴露稳定的 bytes API。它不读取 `.yewseal.toml`，也不知道 alias。

加密接口升级为多 recipient：

```go
func Encrypt(plainData []byte, format string, recipients []string) ([]byte, error)
```

一次加密使用一个 data key，并为同一个文件的全部 canonical recipient 包装该 data key，形成一个包含多个 Age master key 的 SOPS key group。

解密接口接收统一 IdentityBundle 的稳定表示，或接收由 IdentityBundle 生成的完整多行 identity 文本；具体表示不能让下层重新承担 source 选择和逗号解析。

## Encrypt 流程

```text
加载配置
  ↓
选择当前 scope/target 的配置 pair
  ↓
生成所有扫描结果并去重
  ↓
解析每个 pair 的 effective recipients
  ↓
验证 alias、公钥、空集合和 group 冲突
  ↓
所有 pair 通过全量 preflight
  ↓
逐文件把 canonical []string 传入 task/seal/sopsx
  ↓
写入密文
```

### 全量预检

批量 encrypt 必须先解析并校验全部选中的 pair：

- 任一 alias 未知：整个批次在写入前失败；
- 任一公钥非法：整个批次在写入前失败；
- 任一最终集合为空：整个批次在写入前失败；
- 任一 group 授权冲突：整个批次在写入前失败；
- 不允许前面的文件已经写入、后面的文件才暴露配置错误。

全量 preflight 的边界是“授权配置错误不会产生半套密文”。底层文件系统错误或单个文件内容错误仍由 task 按既有批处理结果模型报告。

### 每文件 recipient

每个 resolved FilePair 独立传递 recipient 集合：

```text
ResolvedFilePair.Recipients
  ↓
task.FilePair.Recipients
  ↓
seal.Encrypt(..., recipients)
  ↓
sopsx.Encrypt(..., recipients)
```

不能先把所有文件的 recipient 做全局并集后再用同一集合加密所有文件。

## Decrypt 流程

```text
加载配置
  ↓
按配置选择 FilePair 的路径和格式
  ↓
当前 alias 无法解析时输出 stderr warning
  ↓
解析 IdentityBundle
  ↓
读取密文 metadata
  ↓
使用密文中的 recipient 集合尝试解密
  ↓
校验 MAC 并写入明文
```

Decrypt 不比较当前配置授权集合和密文 metadata，也不因为 alias 已删除、重命名或当前 registry 不完整而阻止历史密文解密。当前配置仍然负责文件路径和格式治理，因此 decrypt 不能绕过配置读取任意未登记的密文路径。

## Edit

如果本期继续保留现有 edit：

1. edit 必须命中已加载配置中的 FilePair；
2. 删除 nil Config 时的 `DefaultConfig` 和固定 wrangler fallback；
3. 解密后编辑并重新加密时，保留原密文的完整 recipient 集合；
4. 不能只取 metadata 中的第一把 recipient；
5. 不能使用当前配置集合自动替换原密文集合；
6. 不借 edit 偷渡 rekey 语义。

这样 edit 仍然是“编辑现有密文”，而不是“借编辑机会改变授权”。如果本期不实现 edit，则不扩大本期范围，但不能保留绕过配置治理边界的旧旁路。

## Init

`init` 只创建一个 owner bootstrap，不管理组织中的额外身份。

初始化结果至少包含：

```toml
[key]
file_path = ".age/keys.txt"

[recipients]
defaults = ["owner"]

[recipients.registry]
owner = "age1generated..."

[[encryption.files]]
plaintext = "wrangler.toml"
encrypted = "wrangler.enc.toml"
```

初始化流程：

```text
生成 owner identity
  ↓
写入 .age/keys.txt
  ↓
写入 registry.owner = generated public key
  ↓
写入 recipients.defaults = ["owner"]
  ↓
写入显式 wrangler FilePair
  ↓
按配置 recipient 生成 .sops.yaml
```

Init 不负责：

- 生成 ops、backup 等额外身份；
- 录入额外 recipient 公钥；
- 推导组织授权关系；
- 管理身份生命周期。

额外 alias 由用户在 registry 中配置，额外私钥由外部身份管理或 sync 流程提供。

### Init force

`init --force` 是完全重建操作：

- 重建 `.age/keys.txt`；
- 重建 recipient registry；
- 重建 defaults；
- 重建 files 配置；
- 重建 `.sops.yaml`；
- 旧的额外 alias 和旧文件映射不保留；
- 旧密文可能无法再由新 owner identity 解密。

本期接受 `--force` 即重置的破坏性语义，不增加额外的二次确认或 `--reset` 开关。CLI 和文档必须明确提示该操作可能使现有密文失去解密能力。

## `.sops.yaml` 集成

`.sops.yaml` 是 YewSeal 生成的完全托管文件，而不是 YewSeal encrypt 的授权输入。

文件头应包含类似警告：

```yaml
# Generated by YewSeal. DO NOT EDIT.
# This file is fully managed and will be overwritten by init/sync/encrypt.
creation_rules:
  - path_regex: '^config/production\.enc\.yaml$'
    age: age1backup...,age1ops...
```

### 生成规则

1. 每个 encrypted path 一条 `creation_rule`；
2. 每条规则使用该文件最终解析后的 canonical recipient 集合；
3. 多 recipient 使用一个 `age` 字段表达；
4. recipient 按 canonical 公钥顺序序列化；
5. 同一 encrypted path 不生成重复规则；
6. 每次同步完全重写全部 `creation_rules`；
7. 不保留用户手工添加的额外 rules。

用户手工修改 `.sops.yaml` 的内容会在下一次 init、sync 或自动同步时丢失。CLI、文档和生成文件头必须明确说明这一点。

### YewSeal 与 `.sops.yaml` 的边界

YewSeal encrypt：

```text
.yewseal.toml
  ↓
resolved canonical recipients
  ↓
sopsx.Encrypt
```

`.sops.yaml` 只作为直接使用 SOPS 的便利配置。YewSeal 不读取 `.sops.yaml`，不以它覆盖或补充 `.yewseal.toml` 的 recipient。

现有 encrypt 的自动 metadata 同步行为保留：配置变更后的 encrypt preflight 可以重写 `.sops.yaml`。Infisical sync 不修改 `.sops.yaml`，因为它只负责私密 identity bundle。

## Sync

Infisical sync 只同步完整的私密 identity bundle：

```text
provider secret
  ↔
AGE-SECRET-KEY-1...
AGE-SECRET-KEY-1...
```

Sync 不负责：

- 从 identity 推导 registry 公钥；
- 自动新增或删除 alias；
- 修改 `recipients.defaults`；
- 修改 FilePair 或 Group 授权；
- 修改 `.sops.yaml` 授权规则。

这样私密能力同步和公开授权策略保持独立。

## Plan 与未来 Verify

本期不新增 verify CLI。

配置层提供可复用的 resolved authorization API，至少能够返回：

- FilePair 路径；
- format；
- recipient alias；
- canonical Age recipients；
- registry 定义来源；
- file/group/default 授权来源；
- 最终 effective recipient set。

`plan` 复用 encrypt preflight，并展示每个 pair 的最终授权信息。Plan 对 recipient 解析与 encrypt 使用相同的严格语义；未知 alias、空集合和 group 冲突直接失败，不使用 decrypt 的宽松 warning 语义。

后续 verify 可以复用：

- config resolved authorization；
- `sopsx.Inspect`；
- `sopsx.Decrypt`；
- IdentityBundle 解析器。

本期不冻结 verify 的报告格式、strict 模式和退出码契约。

## 安全要求

1. recipient registry 中只能出现公开 Age recipient，绝不能出现私钥。
2. identity bundle 不得进入普通输出、verbose 输出、JSON、错误链或测试快照。
3. `YEWSEAL_AGE_IDENTITIES` 适合 CI 的临时注入，但安全性低于挂载的私钥文件；容器和长期运行环境优先使用 `--key-file`。
4. 加密时不能从 identity bundle 推导 recipient。
5. 未知 alias、重复 alias、非法公钥和空授权集合必须在写密文前失败。
6. alias 解析错误可以显示 alias 和配置路径，但不能显示任何 identity 内容。
7. 新增 identity 不会自动扩大任何文件的授权范围。
8. 修改 registry 不会自动改变 defaults 或已显式声明的 file/group 集合。
9. `init --force` 必须明确提示旧密文可能无法由新 identity 解密。
10. `.sops.yaml` 必须明确标记为完全托管文件，避免用户误以为手工规则会永久保留。

## 测试要求

### 配置测试

1. 顶层 `[recipients]` 和 `[recipients.registry]` 解析。
2. registry 中单个和多个 alias 解析。
3. alias 命名校验。
4. alias 大小写敏感。
5. 未知 alias 报错。
6. 非法 Age recipient 报错。
7. 空 defaults 报错。
8. FilePair 显式空 recipients 能被识别为清除继承，encrypt 最终报空集合错误。
9. Group 显式空 recipients 能被识别为清除继承，encrypt 最终报空集合错误。
10. defaults 省略时，显式 FilePair/Group 可以独立提供 recipients。
11. 最终没有任何授权来源时报错。
12. 重复 alias 报错。
13. 重复规范化公钥报错。
14. 父子配置同名 alias 报错。
15. 父子配置重复 defaults 报错。
16. 同路径 FilePair 使用完整项替换，不发生字段级 recipients 继承。
17. 显式 FilePair 覆盖 group/defaults。
18. 多个 group 命中同一路径且有效集合相同时合并。
19. 多个 group 命中同一路径且有效集合不同时报错。
20. file-level recipient 集合不与其他文件共享可变底层数组。
21. alias 解析结果稳定排序和去重。
22. resolved 结果保留 registry、声明和最终生效来源。

### 选择与命令测试

1. 无配置文件时报错。
2. 空 `.yewseal.toml` 时报错。
3. 无位置参数选择当前 directory scope 的全部配置项。
4. Git 根目录和子目录 scope 行为正确。
5. 显式 target 必须命中已配置 pair。
6. target 可以匹配 plaintext 或 encrypted 任一侧。
7. target 可以访问已加载配置中的 scope 外 pair。
8. 未配置 target 不再生成临时 FilePair。
9. Group 扫描结果可以进入配置选择，但不能绕过授权解析。
10. 无 `--all` 参数仍然支持无目标 all 语义。

### Identity 测试

1. key file 中多条私钥全部保留。
2. 注释和空行被忽略。
3. 无法识别的非 identity 行被忽略。
4. 收集到的 identity 会经过 Age parser 验证。
5. 没有有效 identity 时返回明确错误。
6. `YEWSEAL_AGE_IDENTITIES` 的逗号解析。
7. 首尾空白清理。
8. 空项拒绝。
9. 重复 identity 文本去重。
10. 去重后保持首次出现顺序。
11. 显式 `--key-file` 失败时不回退。
12. 新变量非空但解析失败时不回退。
13. `SOPS_AGE_KEY` 保持多行语义。
14. SOPS 旧 source 保持既有兼容语义。
15. 错误和日志不包含私钥内容。

### 加密和解密测试

1. 不同文件使用不同 alias 集合加密。
2. 一个文件使用多个 alias 加密。
3. 默认集合可以为未显式声明 recipients 的文件提供授权。
4. Group 默认可以为扫描文件提供授权。
5. FilePair 可以收窄 Group/defaults 授权。
6. 解析后的公钥集合按每个文件传入 SOPS facade。
7. SOPS metadata 包含同一 data key 对应的多个 recipient。
8. 全量 encrypt preflight 在任何密文写入前拒绝错误配置。
9. 批量中后续文件授权错误时不保留前置授权成功文件。
10. 多 identity 消费者可以解密多个不同授权文件。
11. 不匹配的 identity bundle 无法解密对应文件。
12. recipient alias 顺序变化不会产生授权语义变化。
13. decrypt 不因当前 registry alias 失效而阻塞历史密文。
14. decrypt alias 失效时输出 stderr warning 且成功退出码不变。
15. decrypt 不比较当前配置集合与密文 metadata。
16. edit（若保留）重新加密时保留密文完整 recipient 集合。

### Init、Sync 和 SOPS 配置测试

1. init 写入 owner registry。
2. init 写入 defaults = ["owner"]。
3. init 写入显式 wrangler FilePair。
4. init 不依赖程序内置 DefaultFilePair 运行时 fallback。
5. init --force 完全重建 key、registry、defaults、files 和 `.sops.yaml`。
6. `.sops.yaml` 生成完全托管警告。
7. `.sops.yaml` 每个 encrypted path 生成一条规则。
8. `.sops.yaml` 规则反映每个文件的完整 canonical recipients。
9. `.sops.yaml` 同步完全重写旧 rules。
10. YewSeal encrypt 不读取 `.sops.yaml`。
11. Infisical sync 保留完整多 identity bundle。
12. Infisical sync 不修改 registry/defaults。

## 分阶段实现

### 第一阶段：配置模型和授权解析

1. 在 CUE schema 中增加顶层 `recipients`、registry、defaults 和 FilePair/Group recipients。
2. 更新完整 example 配置并重新导出 JSON Schema。
3. 在 `internal/config` 中实现 presence-sensitive recipients 表示。
4. 实现合并配置中的 registry/defaults 校验。
5. 实现 alias、公钥、重复定义和 group 冲突诊断。
6. 在 resolved selection 中暴露 canonical recipients、alias 和来源。
7. 删除配置层的内置默认 FilePair 运行时 fallback。

### 第二阶段：选择语义和加密 API

1. 删除未配置 target 的临时 FilePair 生成。
2. 固定无位置参数为当前 scope 的 all 语义。
3. 让 encrypt/plan 使用严格的全量授权 preflight。
4. 将 `sopsx.Encrypt`、seal、task 和 app encrypt API 改为多 recipient。
5. 让每个文件独立携带 canonical `[]string`。
6. 删除 `public_key`、`--public-key`、`GetPublicKey` 及旧推导逻辑。
7. 更新 `.sops.yaml` 逐文件多 recipient 生成和完全托管标记。

### 第三阶段：Identity bundle

1. 把 key file 读取统一为完整 IdentityBundle。
2. 增加 `YEWSEAL_AGE_IDENTITIES` 逗号解析。
3. 保持已有 SOPS 环境变量兼容。
4. 明确显式 source 失败策略。
5. 更新 decrypt、edit 和 sync 的多 identity 行为。

### 第四阶段：Init、Edit 和迁移测试

1. 让 init 写入 owner registry、defaults 和显式 wrangler FilePair。
2. 实现 init --force 完全重建语义。
3. 让 edit 命中配置并保留原密文 recipient 集合，或明确不在本期实现 edit。
4. 删除旧模型测试，新增新契约测试。
5. 同步 CUE、example、导出的 JSON Schema 和 schema tripwire。
6. 运行完整项目检查。

## 验收标准

1. 用户可以在 `.yewseal.toml` 中通过 alias 一眼看出每个文件的授权对象。
2. quickstart 项目可以通过 init 生成 owner/defaults，并且不需要手工为每个文件填写 recipients。
3. 所有运行时处理的文件都来自显式 FilePair 或 Group 配置。
4. 无配置、空配置和未配置 target 不会被程序静默转换为 wrangler 或临时 FilePair。
5. 不同文件可以拥有完全不同的 recipient 集合。
6. Group 可以提供默认集合，FilePair 可以显式收窄或替换它。
7. 加密结果中的 SOPS metadata 使用每个文件 alias 解析后的真实 Age 公钥集合。
8. 一个消费者可以提供包含多把私钥的 IdentityBundle，并解密其有权访问的多个文件。
9. 环境变量传入多把私钥时，`YEWSEAL_AGE_IDENTITIES` 使用逗号，不依赖跨环境不稳定的 `\\n` 转义。
10. 项目配置和所有正常输出中都不存在私钥。
11. 未知 alias、重复定义、非法公钥或空授权集合不会被静默当成其他 recipient 处理。
12. encrypt 在任何密文写入前完成完整授权预检。
13. decrypt 可以在 alias 已失效的情况下继续使用密文 metadata 解密，并输出非致命 warning。
14. `.sops.yaml` 与 YewSeal 配置中的逐文件授权保持一致，并明确属于完全托管生成文件。
15. 本期提供稳定的 resolved authorization 和 IdentityBundle API，后续 verify 可以直接复用。
