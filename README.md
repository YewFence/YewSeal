# YewSeal

YewSeal 是一个围绕 **SOPS + age** 构建的 CLI 工具，用清晰的配置文件定义批量加密/解密工作流。

它不重复造加密轮子，而是把项目初始化、文件映射登记、批量加密/解密和 age 私钥管理整合进一个友好的 CLI。

## 功能特性

- 基于 SOPS + age，兼容原生 `.sops.yaml` 和 age 密钥工作流
- 使用直观的 TOML 配置文件，统一管理项目中的 `plaintext` / `encrypted` 文件映射
- `init` 快速初始化，自动生成密钥、配置和推荐的项目文件
- 提供简洁的 `yews e` / `yews d` 别名，可按配置处理所有文件，并支持目录扫描与并行处理
- 支持 TOML、YAML、JSON、ENV、INI；TOML 由内置的原生 TOML store 直接加密（基于 [YewFence/sops](https://github.com/YewFence/sops) fork）
- 支持从环境变量、文件或配置文件读取密钥
- 可选将私钥同步到密钥管理服务（Infisical）
- 加密/解密后保持原始数据结构

## 快速开始

```bash
# 使用 mise 引入工具
mise use github:YewFence/YewSeal

# 交互式初始化项目
# 生成 age 密钥、.yewseal.toml，并登记需要处理的文件
yews init

# 按配置批量加密所有登记过的文件
yews e

# 按同一份配置批量解密
yews d
```

初始化完成后，建议将加密后的文件提交到版本控制，并妥善备份 `.age/keys.txt`。它包含 age 密钥对，一旦丢失就无法解密文件。

**不想增加额外的配置文件？** 完全可以只用命令行参数运行：

```bash
# 所有参数都可以通过命令行指定
yews encrypt config.toml -o config.enc.toml -k .age/keys.txt -p "age1..."
yews decrypt config.enc.toml -o config.toml -k .age/keys.txt
```

## 安装

### go install

```bash
go install github.com/YewFence/YewSeal/cmd/yews@latest
```

### mise

通过 [mise](https://mise.jdx.dev/) 的 [github backend](https://mise.jdx.dev/dev-tools/backends/github.html) 直接下载 release 中的预构建可执行文件并安装：

```bash
mise use --global github:YewFence/YewSeal
yews --version
```

发布包里的可执行文件名固定为 `yews`（Windows 下为 `yews.exe`），所以安装后直接使用 `yews` 即可。

### 从源码安装

```bash
git clone https://github.com/YewFence/YewSeal
cd YewSeal

# 使用 mise task（推荐）
mise run install    # 安装到 $GOPATH/bin，可全局使用

# 或手动安装
go install -ldflags "-s -w -X main.version=$(git describe --tags --always --dirty 2>/dev/null || echo dev)" ./cmd/yews
```

### Github Release

从 Releases 页面下载适合你系统的预编译二进制文件。

### Docker

如果你的环境里已经有 Docker，但不想额外安装 `yews`，可以直接跑容器镜像 `ghcr.io/yewfence/yew-seal:latest`

```bash
docker run --rm -it \
  --user "$(id -u):$(id -g)" \
  -v "$PWD:/work" \
  ghcr.io/yewfence/yew-seal:latest --help
```

#### 补充说明

手动使用 infisical cli 导出私钥至本地的参考命令如下
```bash
umask 077
mkdir -p .age
infisical secrets get AGE_KEY_FILE --plain > ./.age/keys.txt
```

> `edit` 和 `sync` 不建议通过 Docker 运行：前者依赖宿主编辑器，后者依赖宿主机上的 `infisical` CLI 和登录状态

## 使用指南

### 1. 初始化项目

首次使用时，在项目根目录运行：

```bash
yews init
```

#### 交互式模式（默认）

直接运行 `yews init` 将进入交互式模式，程序会逐步询问：

1. **原始配置文件名**（默认：`wrangler.toml`）
   - 输入您需要加密的配置文件名

2. **加密文件名**
   - TOML 默认：`wrangler.enc.toml`
   - 其他格式默认：在原扩展名前插入 `.enc`
   - 直接回车即可使用按文件类型推断出的默认值

3. **是否为当前文件创建示例文件**（默认：否）
   - 每录入一个文件都会单独询问一次
   - 建议只给需要保留结构说明或注释的配置文件创建 example

4. **是否继续添加下一个文件**
   - 可以连续录入多组 `plaintext` / `encrypted` 文件映射

5. **是否创建 .sops.yaml**（默认：是）
   - 非必需，但便于直接使用 `sops` 命令

初始化结束后，生成的 `.yewseal.toml` 会统一使用 `[[encryption.files]]` 数组配置。
每一项都用 `plaintext` 表示明文文件、`encrypted` 表示加密文件。

#### 初始化完成后会创建

- `.age/keys.txt` - Age 密钥对（私钥 + 公钥）
- `.yewseal.toml` - YewSeal 配置文件（在 `[recipients.registry]` 中保存公开 recipients，并包含 `[[encryption.files]]` 数组）
- `.sops.yaml` - SOPS 配置文件（如果选择创建）
- `.gitignore` - 自动添加需要忽略的文件
- `*.example.toml` - 示例配置文件（如果选择创建）

> 如果选择创建了示例配置文件，记得在将其提交到版本控制系统前删除其中的敏感内容

> 📝 **关于 `.sops.yaml`**：虽然 `yews` 工具本身不强制要求此配置文件（通过环境变量传递密钥），但仍**强烈推荐**创建它，原因如下：
> - 🤝 便于团队协作，确保使用统一的加密配置
> - 🔧 支持直接使用原生 `sops` 命令操作加密文件
> - 📋 符合 SOPS 官方最佳实践
> - 🎯 明确定义哪些文件应该被加密

**重要提示**：请妥善保管 `.age/keys.txt` 私钥文件，丢失后无法恢复！

### 2. 加密配置文件

```bash
yews encrypt
yews e  # 简写
```

### 3. 解密配置文件

```bash
yews decrypt
yews d  # 简写
```

### 4. 直接编辑加密文件

使用编辑器直接编辑加密文件（自动处理加密/解密）：

```bash
yews edit -f ./path/to/file
```

## 密钥读取

解密支持多种 Age identity source，优先级从高到低：

1. 显式全局选项 `--key-file` / `-k`，或 `AGE_KEY_FILE`
2. `YEWSEAL_AGE_IDENTITIES` 环境变量中的逗号分隔 identity bundle
3. `SOPS_AGE_KEY` 环境变量中的多行 identity bundle
4. `SOPS_AGE_KEY_FILE` 环境变量
5. `SOPS_AGE_KEY_CMD` 环境变量
6. `.yewseal.toml` 的 `[key].file_path`
7. 默认路径 `.age/keys.txt`

### 命令行参数

```bash
yews --key-file /path/to/keys.txt decrypt
```

加密 recipient 只来自 `.yewseal.toml` 的 registry 与 file/group/defaults alias 配置，不从私钥或环境变量推导。

### 环境变量（推荐用于 CI/CD）

```bash
# 传递逗号分隔的多把私钥
export YEWSEAL_AGE_IDENTITIES='AGE-SECRET-KEY-1...,AGE-SECRET-KEY-1...'
yews decrypt

# SOPS 兼容变量继续使用多行 bundle 语义
export SOPS_AGE_KEY="AGE-SECRET-KEY-..."
yews decrypt
```

### 默认位置

默认从 `.age/keys.txt` 读取私钥；加密授权从 `.yewseal.toml` 的 `[recipients.registry]` 和 alias 集合解析。完整规则见[文档站](https://yewfence.github.io/YewSeal/guide/configuration#age-密钥管理)。

## 密钥同步

### Infisical 集成

支持将 Age 密钥同步到 Infisical 密钥管理服务。
> [Infisical](https://infisical.com/) 是一个开源的，可轻松自托管的秘密管理平台，此处使用它的 [infisical CLI](https://infisical.com/docs/cli/overview)。

#### 1. 配置 Infisical

```bash
infisical login
infisical init
```

此时你的项目中会生成一个 `.infisical.json` 配置文件，本工具会检测该配置文件是否存在作为 Infisical 配置完成与否的标志。

#### 2. 同步 AGE 密钥到 Infisical
```bash
# 直接同步到项目根目录的 AGE_KEY_FILE 变量
yews sync
```

## Git 工作流

### 将加密文件和配置文件提交到版本控制

```bash
git add .gitignore .yewseal.toml wrangler.enc.toml
# git add .infisical.json  # 如果使用 Infisical
git add .sops.yaml  # 可选但推荐
git commit -m "feat: 添加加密配置"
```

### 忽略的文件

- `wrangler.toml` - 明文配置文件
- `.age/keys.txt` - 私钥文件

## CI/CD 集成

### GitHub Actions 示例

在仓库 `mise.toml` 声明 `github:YewFence/YewSeal` 依赖后使用：

```yaml
name: Deploy
on: push

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7
        with:
          persist-credentials: false

      - uses: jdx/mise-action@v4

      - name: Decrypt configuration
        env:
          SOPS_AGE_KEY: ${{ secrets.AGE_KEY }}
        run: yews decrypt

      - name: Deploy
        run: wrangler deploy
```

## 故障排查

### 解密失败

检查密钥是否正确设置：

```bash
# 检查环境变量
echo $SOPS_AGE_KEY

# 或检查文件
cat .age/keys.txt
```

### TOML 格式问题

TOML 由内置的原生 TOML store 直接加密（不再有格式转换环节）。解密输出是规范化 TOML——字符串使用单引号字面量风格，注释会保留；内容与原文等价，但排版可能与手写格式不同。如果遇到问题，请提交 Issue。

## 命令参考

完整的命令行参考请查看 [docs/references](docs/references/yews.md)（由 `mise run cli:docs` 自动生成）。

## 工作原理

所有格式（含 TOML）: 原文件 → sops encrypt/decrypt → 加密/解密文件

YewSeal 内嵌了成熟的加密库：
- **Age** ([`filippo.io/age`](https://github.com/FiloSottile/age))
- **SOPS(fork)** ([`github.com/YewFence/sops/v3`](https://github.com/YewFence/sops))

> **关于内嵌的 SOPS fork**：核心加解密引擎用的是我自己的 fork [YewFence/sops](https://github.com/YewFence/sops)（在上游 [getsops/sops](https://github.com/getsops/sops) 基础上新增原生 TOML store）。我还没有认真审核过它，所以也没有向上游提交 PR。如果发现引擎层面的问题，欢迎直接到我的 fork 开 Issue。

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request！详见 [CONTRIBUTING.md](CONTRIBUTING.md)。

## 碎碎念

我写这个的原因是因为我真的不想把 `wrangler.toml` 直接放在 GitHub 上，虽然里面没有泄露就出事的敏感信息，但是会有自己的域名/项目名称/ `KV ID` `D1 ID` / `R2 NAME` 之类的信息，放在公开仓库总觉得不太好。直接放在 Infisical 里又不太方便，毕竟每次改配置都得去网站上操作一遍。所以就想到使用 SOPS ，但 SOPS 原生不支持 TOML 格式，我又真的不想用 yaml，缩进地狱，所以就写了这个工具来帮忙做加密解密的编排工作。

想着来都来了，增加一个方便点的包装吧，于是就有了短命令和配置文件持久化功能，省得每次都敲一大堆参数。包装都包装了，干脆支持所有 SOPS 支持的格式吧，万一以后有别的配置文件需要加密呢。支持都支持了，不如增加一个多文件批量加密解密的功能，省得每次都得写脚本。既然有了批量处理，那就顺便加个并行处理，都用 Go 了，不些并行多可惜。总之就是越做越多功能，最后变成了现在这个样子。

接下来就是密钥管理的问题了，Age 的密钥对生成很方便，而且公钥可以直接放在配置文件里，私钥只需要保存在本地就行了。再结合 Infisical 这种密钥管理服务，就能比较方便地在 CI/CD 里使用密钥了，它也可以实现一个项目一个密钥，减少爆破半径，非常完美。

关于安全性嘛，Age + SOPS 还是挺靠谱的，至少比我自己写加密算法要靠谱多了。

## 声明

本工具仅作为个人学习项目，未经充分测试和审计，不建议在高安全要求环境中使用。

## 致谢

本项目核心功能依赖于以下优秀的开源工具，感谢它们的贡献：

- [Age](https://github.com/FiloSottile/age) (BSD 3-Clause) - A simple, modern and secure encryption tool (and Go library) with small explicit keys, no config options, and UNIX-style composability.
- [SOPS](https://github.com/getsops/sops) (MPL 2.0) - Simple and flexible tool for managing secrets
- [Infisical CLI](https://github.com/Infisical/cli) (MIT) - The official CLI of Infisical

## 相关链接

- [Age](https://github.com/FiloSottile/age)
- [SOPS](https://github.com/getsops/sops)
- [YewFence/sops](https://github.com/YewFence/sops)（带原生 TOML store 的 SOPS fork）
- [Infisical](https://infisical.com/)
- [Wrangler Configuration](https://developers.cloudflare.com/workers/wrangler/configuration/)
