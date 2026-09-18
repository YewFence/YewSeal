# 增量加密与已有密文更新

## 状态

本文档记录 YewSeal 更新已有 SOPS 密文时避免无关 diff 的设计决策。方案已实现；实现细节与命令帮助保持同步。

## 背景

YewSeal 此前把每次 `encrypt` 都当作首次加密：读取明文、生成新的随机 data key、为每个 Age recipient 包装 data key，再用新的 AES-GCM cipher 加密整棵 SOPS tree。即使明文没有变化，重新生成的 data key 和每个 value 的随机 IV 也会让全部 value ciphertext 改变；`lastmodified` 和 MAC 还会产生额外 metadata diff。

时间信息不是全部 Values 变化的根因。`lastmodified` 主要影响 metadata 和 MAC；全部 Values 变化来自“新 data key + 每个 value 的新随机 IV”。这种概率加密行为本身是正确的安全属性，不能通过固定 IV、复用发生过明文变化的 nonce 或确定性加密来消除，否则会泄漏明文相等关系，错误复用 AES-GCM nonce 还会破坏机密性和完整性。

实际需求不是重复执行一个纯粹的 `Encrypt(plaintext, recipients)`，而是让已有密文与当前明文和授权策略协调一致。协调已有状态天然是读改写操作：如果不解密旧密文、不引入公开的明文指纹，也不依赖不可靠的本地缓存，就无法判断当前明文是否与密文内容相同。

## 目标

1. 未变化的文件保持字节不变，不因重复执行 `yews encrypt` 产生 diff。
2. 结构化文件只修改一个 value 时，未变化 value 的 ciphertext 保持不变。
3. 仅 recipients 变化时，只更新被 Age recipients 包装的 data key，不重加密 Values。
4. 继续允许没有私钥的环境从明文和公开 recipients 创建可用密文，但明确报告无法执行增量更新。
5. 保留 `internal/sopsx` 作为唯一 SOPS engine facade，其他包不直接依赖 SOPS 类型。
6. 保持现有文件选择、授权解析、Group 发现和项目 metadata 同步职责。

## 非目标

1. 不引入确定性加密、固定 IV 或公开的明文 hash。
2. 不增加本地状态数据库、mtime 判断或 sidecar cache 作为正确性依据。
3. 不新增 `sync` 命令；该名称容易与已经明确排除的私钥同步和 Provider 集成混淆。
4. 不在本次设计中改变密文文件的写入原子性。
5. 不改变 `yews diff` 的用户语义。

## 核心决策

### 区分首次加密与已有密文更新

**决策**：保留纯粹的 `sopsx.Encrypt`，新增面向已有密文的 `sopsx.Update`。CLI 仍使用 `yews encrypt` 表达“根据已登记明文创建或更新密文”，app 层根据目标状态选择 `Encrypt` 或 `Update`。

**理由**：`Encrypt` 只依赖明文和公开 recipients，不读取旧密文，也不需要私钥；`Update` 明确是读改写操作，可以合法地解密旧状态、比较内容、复用 data key 并最小化输出变化。把两个概念拆开后，低层接口保持纯粹，CLI 也不要求用户区分首次创建和日常更新。

### 普通更新复用 data key

**决策**：已有密文的普通内容更新复用原 data key；只有首次创建、缺少可用 identity 时的降级路径，以及显式 `--force` 才生成新 data key。

**理由**：SOPS 的 AES cipher 会在解密时记录 `(plaintext, path) -> IV`。使用同一个 cipher 实例和同一个 data key 重新加密新 tree 时，值和路径都未变化的叶子会复用原 IV，得到完全相同的 ciphertext；变更值或路径变化的值才生成新 IV。该机制只对相同 plaintext 和相同 AAD 复用 nonce，不会让同一个 key/nonce 组合加密不同明文。

**边界**：binary store 的整个文件是一个值，任何内容变化都会更新这个值；结构化格式才能获得逐 value 的最小 diff。键重命名、移动或数组位置变化会改变 path AAD，因此对应 value 必须重新加密。

### 更新仍然解密一次

**决策**：`Update` 对已有密文执行一次完整解密，取得原 data key、验证 MAC、获得旧明文 tree，并填充同一个 AES cipher 的 IV stash。比较和后续重加密共享这次解密结果，不再额外调用独立的 diff 解密流程。

**理由**：正确判断 `plaintext == Decrypt(ciphertext)` 必须掌握私有解密能力或泄漏一个公开相等性 oracle。解密不是纯 `Encrypt` 的隐藏步骤，而是 `Update` 的显式职责。

### 明文内容按字节判断是否完全未变

**决策**：`Update` 优先比较旧 tree 输出的明文字节与本次读取的明文快照；如果 store 会规范化排版，则再比较解析后的 tree。只有字节或 store 语义都没有变化且 recipients 相同，整个密文文件才原样返回。

**理由**：字节比较保留了 `yews diff` 对空白、换行和注释的敏感性；语义 fallback 处理 YAML、JSON、INI 等 store 的规范化输出，避免同一份结构化数据因 emitter 的固定排版而每次更新。每个明文文件只读取一次，比较和实际加密使用同一份 bytes 快照。

**边界**：SOPS store 本身不承诺保留的纯排版细节无法稳定映射到密文；各格式必须通过 round-trip 测试固定实际能力。

### recipients 变化优先复用原 data key

**决策**：当当前配置的 recipients 与密文 metadata 不同时，如果已有 identity 能解开旧 data key，则复用该 data key，并只重新包装到当前 recipients；如果明文同时变化，再使用同一个 data key 和 cipher 执行增量内容更新。

**理由**：recipient 授权变化不等于 data key 轮换。SOPS `updatekeys` 的语义就是保留 Values，只更新 master key 包装。这样纯授权调整不会制造 Values diff。

**边界**：若没有 identity 或现有 identity 均不匹配，无法取得原 data key。此时在明文存在且当前 recipients 已通过配置校验的前提下，警告后执行全新加密；该文件会产生全量密文 diff。

### `--force` 表示全新加密

**决策**：`yews encrypt --force` 跳过已有密文的读取、比较和更新，直接从当前明文生成新的 data key 和完整密文。

**理由**：`--force` 是修复损坏密文、验证完整重建和显式轮换 data key 的恢复入口。它只绕过旧密文比较，不绕过明文存在性、recipient 配置校验或加密错误。

## 处理矩阵

| 明文 | 密文 | identity | `--force` | 行为 |
| --- | --- | --- | --- | --- |
| 缺失 | 任意 | 任意 | 任意 | `missing plaintext`，不写密文 |
| 存在 | 缺失 | 任意 | 否 | `Encrypt`，创建新密文 |
| 存在 | 任意 | 任意 | 是 | `Encrypt`，生成新 data key 并完整重写 |
| 存在 | 存在 | 匹配 | 否 | `Update`，解密一次并最小化更新 |
| 存在 | 存在 | 无任何来源 | 否 | 批次 warning 后 `Encrypt` |
| 存在 | 存在 | bundle 不匹配 | 否 | 逐文件 warning 后 `Encrypt` |
| 存在 | 损坏 | 本应匹配 | 否 | 该文件失败并保留原密文 |

明文缺失本身不使批次失败；所有选中项都因明文缺失而跳过时仍退出 0。Group 保持按明文侧发现，因此只有密文、没有明文的动态 Group 文件不会新增为候选；显式 file mapping 可以报告 `missing plaintext`。

## `sopsx.Update` 契约

建议的 facade 形状如下，具体 Go 类型可在实现时按现有错误与结果模型调整：

```go
type UpdateOptions struct {
	Plaintext          []byte
	ExistingCiphertext []byte
	Format             string
	AgeIdentity        string
	Recipients         []string
}

type UpdateResult struct {
	Ciphertext        []byte
	ContentChanged    bool
	RecipientsChanged bool
	Unchanged         bool
}

func Update(opts UpdateOptions) (UpdateResult, error)
```

`Update` 的内部顺序：

1. 按格式载入 encrypted tree，并读取历史 recipients。
2. 创建一个 AES cipher 实例。
3. 使用 identity bundle 解开原 data key，并用该 cipher 解密 tree 和验证 MAC；解密过程填充 IV stash。
4. 由旧 tree 输出明文字节，与调用方提供的单次明文快照比较。
5. 用 store 解析新明文；需要更新内容时，以新 branches 替换旧 branches。
6. recipients 变化时，用原 data key 构造当前 Age master keys；不轮换 data key。
7. 内容变化时，用同一个 cipher 和 data key 加密新 branches；未变叶子自动复用 IV。
8. 内容和 recipients 均未变化时直接返回原 ciphertext，不重新 emit。
9. 仅 recipients 变化时只 emit 更新后的 metadata，不修改 Values、MAC 或 `lastmodified`。

`Update` 不负责缺文件判断、fallback 决策、warning、磁盘写入和批量汇总；这些仍由 seal/task/app/presentation 各自承担。

## Identity 加载与降级

**决策**：所有“明文和密文都存在、未指定 `--force`”的候选都可能从增量更新中获益，因此 app 在任何项目 metadata 或密文写入前至多解析一次 identity bundle，并把结果交给整个批次。

最终找不到任何 identity 来源属于允许的边界：批次 warning 一次，相关文件执行全新加密。成功加载 bundle 但某个旧密文没有匹配 identity 时逐文件 warning，并对该文件执行全新加密。

显式 identity 文件不可读、identity 内容无效、环境来源解析失败或 `SOPS_AGE_KEY_CMD` 执行失败仍是配置错误，整批在任何写入前失败。缺失的 `SOPS_AGE_KEY_FILE` 保持现有 resolver 的后续来源回退语义；只有最终 `AgeKeyNotFoundError` 被视为允许降级。

## 错误与批量语义

密文能被识别为目标但解析失败、匹配 identity 解密失败、MAC 不一致或 store 处理失败时，默认保护旧密文并将该文件记为失败；其他文件继续处理，批次最终退出 1。用户可通过 `--force` 明确跳过旧密文读取并完整覆盖。

recipient 配置解析仍保持现有严格行为，不因明文缺失或增量更新新增授权豁免。`.gitignore` 继续在加密前同步；启用时，`.sops.yaml` 在任务结束后根据完整项目策略尽力同步，失败只产生 warning。增量更新不改变配置作为授权唯一声明来源的原则。

## 结果与输出

encrypt summary 稳定区分以下结果：

```text
Summary (encrypted): 2 encrypted, 3 unchanged, 1 missing plaintext, 0 failed (6 selected)
```

`encrypted` 表示目标密文实际写入，包括首次创建、增量更新、recipient 更新和无 identity 降级重建；`unchanged` 表示内容与 recipients 都已验证一致；`missing plaintext` 表示未处理，不能解释为已同步。

默认 stderr 打印 warning、missing、failed 和 summary；`encrypted` 与 `unchanged` 的逐文件明细只在 verbose 模式打印。stdout 保持为空。输出通道故障继续遵循[命令输出的所有权与失败语义](command-output.md)。

## SOPS MAC 兼容性修正

旧实现中的 `sopsx.Encrypt` 和 `sopsx.Rekey` 直接把 `tree.Encrypt` 返回的 SHA-512 值写入 `Metadata.MessageAuthenticationCode`，解密时也直接比较该值。标准 SOPS 会使用 data key 加密 MAC，并把 `lastmodified` 作为 AAD；直接使用 fork 的 SOPS CLI 读取 YewSeal 密文时必须遵循同一格式。

本次实现已经让 facade 的 MAC 生成与验证对齐 SOPS 原生流程，并增加由 YewSeal 加密、再通过同版本 fork 的公开 data-key、tree 和 MAC API 解密验证的互操作测试。为避免现有项目首次升级时只能通过 `--force` 制造一次全量 diff，读取路径仍接受并验证旧版 YewSeal 的明文 MAC；新建和内容更新只写标准加密 MAC，不保留旧格式的写入能力。IV stash、原 data key 复用和 `updatekeys` 语义因此建立在统一的 SOPS 文件模型上。

## 对原设计树的调整

此前已经确认的文件选择、缺明文、无 identity 降级、错误保护、`--force`、输出与 summary 决策继续有效。以下节点需要替换：

1. “先调用 diff 判断，再全新加密”改为“调用一次 `sopsx.Update` 完成解密、比较与最小化更新”。
2. identity 的潜在使用范围从“recipients 相同、需要内容比较的文件”扩大为“所有已有密文且非 force 的文件”。
3. recipients 变化不再默认直接生成新 data key；有可用 identity 时复用原 data key 并更新包装，没有可用 identity 时才降级为全新加密。
4. 新增 data key 生命周期决策：普通更新复用，`--force` 才轮换。
5. 密文内容未变且仅 recipients 变化时，Values、MAC 和 `lastmodified` 都保持不变，只有 Age master key metadata 更新。

## 验证要求

实现必须覆盖 TOML、YAML、JSON、ENV、INI 和 binary，并至少验证：

1. 重复执行 encrypt 时密文逐字节不变。
2. 修改一个结构化 value 时，未修改 value 的 `ENC[...]` 字符串保持不变。
3. 新增、删除、移动和改变类型时可正确解密，并只更新受 path/AAD 影响的值。
4. 仅 recipients 变化时 Values、MAC 和 `lastmodified` 不变，新 recipients 可以解密，移除的 recipients 不能再解开 data key。
5. `--force` 生成新 data key，并使全部加密 Values 更新。
6. 没有 identity 和 identity 不匹配时按约定 warning 后完整重建。
7. 密文损坏或 MAC 不一致时默认不覆盖，`--force` 可以恢复。
8. 明文只读取一次，比较和写入基于同一快照。
9. 顺序不同但集合相同的 recipients 不触发更新；重复 recipient metadata 触发一次规范化。
10. YewSeal 与同版本 fork SOPS CLI 可以双向解密、编辑和更新 keys。
11. 顺序和并行批次拥有相同结果分类、退出码和磁盘副作用。
12. `mise run check` 和受影响模块的 race 测试通过。
