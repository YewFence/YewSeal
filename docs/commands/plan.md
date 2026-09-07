# plan - 检查配置映射

`plan` 检查并展示已登记的文件映射、格式、当前配置授权及其来源，不会加密、解密或写入文件。它是无方向的配置映射检查，不是 encrypt/decrypt 的 dry run；成功不代表文件一定可加密、密文一定可解开或输出路径一定可写。

## 语法

```bash
yews plan [command options] [path]
```

## 目标选择

不传 `path` 时，选择明文或密文任一侧位于当前目录范围内的映射。传入文件时，可以匹配已登记映射的任一侧；传入目录时，选择任一侧位于该目录内的映射。路径只选择映射，不推断操作方向。

Group 始终按所属配置目录和发现规则扫描，`plan` 使用明文侧与密文侧发现结果的并集。因此只有明文的新文件、只有密文的部署文件都可展示；两侧都存在时保留已发现的真实明文路径，例如 `config.yml` 与 `config.enc.yaml`。竞争映射必须由显式 FilePair 裁决，否则报冲突。显式 FilePair 不要求对应文件当前存在。

`plan` 和 encrypt 对当前配置授权采用相同的严格语义：对本次已加载配置所解析出的全部映射检查未知 alias、空授权集合和 Group 冲突，未选中项的授权错误也会导致失败。展示内容包括 recipient alias、canonical recipient、registry 来源和 effective authorization source。

`plan` 不加载 Identity bundle，不读取密文内容或 metadata 来验证授权和完整性，也不使用 decrypt 的失效 alias 警告后继续策略。

## 选项

### --pattern

进一步筛选配置范围或目录目标内已登记的映射，不覆盖 Group 的发现规则，也不会登记额外文件。筛选规则相对于当前工作目录，允许匹配明文或密文路径。

```bash
yews plan ./configs --pattern "*.toml"
```

### --json

以 JSON 输出配置映射及来源。

```bash
yews plan --json
```

### --verbose, -v

展示已加载配置文件等详细信息。

```bash
yews plan -v
```

`plan` 不接受 `--output`、`-o`、`--parallel`、`-P`，也不读取 `SOPS_OUTPUT_FILE`。输出不包含 metadata 写入计划。

## 示例

```bash
# 检查当前目录范围内的映射
yews plan

# 两种路径选择同一个已登记映射
yews plan config.toml
yews plan config.enc.toml

# 筛选目录内已登记的映射
yews plan ./configs --pattern "*.toml"

# 输出 JSON 供脚本读取
yews plan --json
```

## 相关命令

[encrypt](/commands/encrypt) 和 [decrypt](/commands/decrypt) 共享已登记映射的选择主线，但发现方向、目录归属和历史解密的授权处理各有不同。
