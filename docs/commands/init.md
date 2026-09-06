# init - 初始化项目

`init` 用来生成 Age 密钥、创建 YewSeal 配置，并可选同步 `.sops.yaml`。

## 语法

```bash
yews init [command options]
```

## 选项

### --force, -f

完全重建 `.age/keys.txt`、recipient registry、defaults、FilePair 和托管的 `.sops.yaml`。旧 alias 和文件映射不会保留，新 owner identity 可能无法解密已有密文。

```bash
yews init --force
```

### --input, -i

为第一个配置条目指定明文文件。传入 `--input` 或 `--output` 后会进入非交互模式。

```bash
yews init --input config.toml
```

### --output, -o

为第一个配置条目指定加密文件。

```bash
yews init --input config.toml --output config.enc.toml
```

### --format

为第一个配置条目指定格式，支持 `toml`、`yaml`、`json`、`env`、`ini` 和 `binary`。

```bash
yews init --input .dev.vars --output .dev.enc.env --format env
```

### --create-example

创建示例明文文件。交互模式下会为录入的配置条目创建示例文件，非交互模式下会为第一个配置条目创建示例文件。

```bash
yews init --input config.toml --create-example
```

### --skip-sops-config

跳过创建或更新 `.sops.yaml`。

```bash
yews init --skip-sops-config
```

## 交互模式

直接运行 `yews init` 时，命令会询问是否覆盖已有配置，是否创建 `.sops.yaml`，并录入一个或多个明文文件和加密文件映射。

```bash
yews init
```

初始化完成后会写入 `.yewseal.toml`，生成 `.age/keys.txt`，并更新 `.gitignore`。

## 非交互模式

脚本中可以传入第一个文件映射。

```bash
yews init \
  --input config.toml \
  --output config.enc.toml \
  --format toml
```

如果只传 `--input`，加密文件名会自动推断。`config.toml` 会推断为 `config.enc.toml`，其他格式会使用对应的 `.enc.*` 后缀。

## 生成的文件

### .yewseal.toml

这是 YewSeal 的主配置文件。

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
recipients = ["owner"]
```

### .age/keys.txt

这是 Age 私钥文件，不能提交到版本控制。

### .sops.yaml

这是 SOPS 的配置文件，方便直接使用 `sops` 命令。传入 `--skip-sops-config` 时不会创建。

## 相关命令

[encrypt](/commands/encrypt) 可以加密配置文件，[decrypt](/commands/decrypt) 可以解密配置文件。私钥的保存与分发由开发者或部署环境自行管理，参考[外部私钥来源](/guide/private-keys)。
