# 与 SOPS 配合使用

YewSeal 面向通过项目配置统一管理多个文件的加密和解密：在 `.yewseal.toml` 中登记明文与密文映射、文件分组和公开 recipient 授权，再用命令选择本次要处理的范围。不需要项目级管理的临时单文件任务，直接使用独立的 [SOPS CLI](https://getsops.io/docs/) 即可。

YewSeal 自身使用内嵌引擎，不要求安装外部 `sops`，也不会在缺少配置时自动调用它；运行本页示例则需要自行安装。

## 如何选择

| 场景 | 推荐方式 |
| --- | --- |
| 用配置统一管理多个文件、分组和接收者授权 | 使用 YewSeal |
| 只处理项目中一个已经登记的文件 | 使用 `yews encrypt <path>` 或 `yews decrypt <path>` |
| 临时处理一个文件，不需要建立项目配置 | 直接使用 SOPS CLI |

## YewSeal 的规则

文件和加密授权统一来自 `.yewseal.toml`：

- 传入 `path` 是选择已登记的文件，不会为未登记文件自动创建映射或授权。
- `--output` 只为显式单文件目标指定本次输出位置，`--pattern` 筛选本次范围；两者都不声明新授权。
- 格式由文件的 `format`、分组的 `format_rules` 或已登记路径的推断结果确定。业务命令没有 `--format` 选项，只有创建配置时的 `init --format`。
- `--key-file` 提供的是解密身份，不是加密授权声明；加密 recipients 只由配置中的 alias 集合解析。

详细规则见[配置说明](/guide/configuration)。

## 临时处理单个文件

下面以 YAML 为例，无需创建 `.yewseal.toml` 或 `.sops.yaml`。先将 `AGE_RECIPIENT` 设置为实际的 age 公钥；这里的变量只是 shell 示例变量，不是 YewSeal 的配置项。

```bash
export AGE_RECIPIENT='age1...'

# 显式指定公钥，将新密文写入另一个文件
sops --encrypt --age "$AGE_RECIPIENT" \
  --output config.enc.yaml config.yaml

# 从密文 metadata 获取接收者信息，使用本机私钥解密
SOPS_AGE_KEY_FILE=./.age/keys.txt \
  sops --decrypt --output config.decrypted.yaml config.enc.yaml
```

加密不需要私钥。解密不需要重新传入 `--age`，但必须能获取密文所需的身份。不要将私钥或解密后的明文提交到版本控制；运行前也应确认输出路径，不能假定 SOPS 具有 YewSeal `decrypt` 的明文覆盖保护。

SOPS 使用自己的参数、环境变量和身份发现规则，不读取 `.yewseal.toml`。示例显式设置 `SOPS_AGE_KEY_FILE`，不依赖 YewSeal 的 `.age/keys.txt` 默认回退；同样不能把 `YEWSEAL_AGE_IDENTITIES` 当作 SOPS 的身份变量。外部私钥获取方式见[外部私钥来源](/guide/private-keys)。

## 格式兼容边界

YewSeal 的 YAML、JSON、ENV、INI 和 binary 密文使用相应的 SOPS store，可通过支持该格式的 SOPS CLI 处理。扩展名不足以识别格式时，需使用 SOPS 自己的 `--input-type`、`--output-type`；例如 ENV 在 SOPS 中的格式名称是 `dotenv`。

原生 TOML 是需要单独确认的例外：YewSeal 使用 [YewFence/sops fork](https://github.com/YewFence/sops) 的原生 TOML store。直接处理这种 TOML 密文，需要包含同一 store 的 SOPS CLI，例如该 fork 的构建；不能将任意上游 CLI 都视为兼容，可先查看 `sops --help` 的格式列表是否包含 `toml`。

把普通 TOML 当作 binary 交给 SOPS 整体加密是另一种可行方式，但它生成的是 binary store 的密文，不是 YewSeal 的结构化 TOML 密文。不能通过改扩展名或指定 binary 类型来直接读取已有的原生 TOML 密文。

## 已有项目中的 SOPS

YewSeal 可以生成 `.sops.yaml`，方便直接使用 SOPS；但两者的配置职责仍然独立：YewSeal 的加密授权来自 `.yewseal.toml`，SOPS 不读取其中的 registry 或 alias。

直接调用 SOPS 不会替你登记 YewSeal 文件映射或更新 `.gitignore`。对已有项目文件使用 SOPS 时，需要自行保持格式和接收者授权与项目策略一致；如果希望把临时文件纳入后续批量管理，请将其映射和授权显式登记到 `.yewseal.toml`。
