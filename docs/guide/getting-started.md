# 快速开始

YewSeal 是一个基于 SOPS 和 Age 的配置文件加密管理工具，支持 TOML、YAML、JSON、ENV、INI 和二进制文件。所有格式（含 TOML）都由内嵌的 SOPS 引擎原生加密，不需要格式转换。

## 安装

### mise（推荐）

通过 [mise](https://mise.jdx.dev/) 的 [github backend](https://mise.jdx.dev/dev-tools/backends/github.html) 直接安装 release 中的预构建可执行文件：

```bash
mise use --global github:YewFence/YewSeal
yews --version
```

也可以在项目的 `mise.toml` 中声明 `github:YewFence/YewSeal` 依赖，让团队成员和 CI 使用同一版本。

### go install

```bash
go install github.com/YewFence/YewSeal/cmd/yews@latest
```

需要 `$GOPATH/bin` 在 `$PATH` 中。

### GitHub Release

从 [Releases 页面](https://github.com/YewFence/YewSeal/releases)下载适合你系统的预编译二进制文件。可执行文件名固定为 `yews`（Windows 下为 `yews.exe`）。

### 从源码构建

```bash
git clone https://github.com/YewFence/YewSeal.git
cd YewSeal
mise run install    # 安装到 $GOPATH/bin
# 或只构建：mise run build → build/yews
```

### Docker

不想安装二进制时，可以直接运行容器镜像 `ghcr.io/yewfence/yew-seal`，详见 [Docker 运行](/guide/docker)。

## 初始化项目

在项目目录中运行 `init` 命令初始化 Age 密钥、`.yewseal.toml` 和可选的 `.sops.yaml`：

```bash
yews init
```

该命令会生成 `.age/keys.txt`，把 Age 公钥写进 `.yewseal.toml`，交互式录入一个或多个 `[[encryption.files]]` 映射，并把私钥目录和明文文件写进 `.gitignore`。

### 非交互模式

如果需要在脚本中使用，可以为第一个配置条目直接传入明文文件和加密文件：

```bash
yews init \
  --input config.toml \
  --output config.enc.toml \
  --format toml \
  --create-example
```

## 基本使用

### 加密配置文件

```bash
# 加密配置里的所有文件
yews encrypt

# 加密一个已配置的明文文件，使用配置中的 encrypted 路径和授权
yews encrypt config.toml

# 加密单个明文文件，并指定输出路径
yews encrypt config.toml -o config.enc.toml

# 在已配置 Group 的目录范围内筛选并加密
yews encrypt ./configs --pattern "*.toml"
```

### 解密配置文件

```bash
# 解密配置里的所有文件
yews decrypt

# 解密一个已配置的加密文件，使用配置中的 plaintext 路径
yews decrypt config.enc.toml

# 解密单个加密文件，并指定输出路径
yews decrypt config.enc.toml -o config.toml

# 在已配置 Group 的目录范围内筛选并解密
yews decrypt ./configs --pattern "*.toml"
```

默认情况下，`decrypt` 发现明文文件已存在且内容不一致时会拒绝覆盖，可以加上 `--force` 强制写入。

### 预览选择结果

```bash
yews plan
yews plan ./configs --pattern "*.toml"
yews plan --json
```

`plan` 只打印解析后的文件选择、格式来源和配置来源，不会写入文件，适合在批量加密或解密前确认结果。

### 编辑加密文件

```bash
# 使用默认编辑器
yews edit -f config.enc.toml

# 指定编辑器
yews edit -f config.enc.toml -e "code -w"
```

### 查看加密文件内容

```bash
# 输出到标准输出
yews view config.enc.toml
```

### 比较差异

```bash
# 比较明文文件和加密文件的差异
yews diff config.toml
```

## 配置文件示例

初始化后会生成类似下面的 `.yewseal.toml`：

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

也可以用分组配置让 YewSeal 按模式扫描一批文件，详见[配置说明 - 分组扫描](/guide/configuration#分组扫描)。

## 下一步

[命令参考](/commands/init) 可以查看所有命令选项，[配置说明](/guide/configuration) 可以了解 `.yewseal.toml`、`.sops.yaml` 和密钥同步的配置方式。
