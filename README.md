# YewSeal

一个用于管理加密配置文件的 CLI 工具，使用 SOPS + Age 加密，支持多种配置格式（TOML/YAML/JSON/ENV/INI）。

## 功能特性

- 📦 多格式支持：TOML、YAML、JSON、ENV、INI
- 🔄 TOML 格式转换后加密，补充 SOPS 缺失的 TOML 支持
- 🚀 批量加密解密，支持目录扫描和并行处理
- 🔑 灵活的密钥管理：环境变量、文件、配置文件
- ✨ 简单易用的命令行界面，支持命令别名
- 🛠️ 配置文件持久化常用设置
- 🔐 密钥同步到密钥管理服务（Infisical）
- 📝 加密解密后文件结构完全一致

## 前置要求

Age 和 SOPS 已内嵌为 Go 库，**无需单独安装**。

仅在使用 TOML 格式时需要安装格式转换工具：

- [Remarshal](https://github.com/remarshal-project/remarshal) - TOML/YAML 格式转换工具（仅 TOML 格式需要）

### Windows 安装（Scoop）

```powershell
# 仅 TOML 格式需要
pipx install remarshal
```

### macOS 安装（Homebrew）

```bash
# 仅 TOML 格式需要
pipx install remarshal
```

### Linux 安装

```bash
# 仅 TOML 格式需要
pipx install remarshal
```

## 快速开始

```bash
# 1. 交互式初始化项目（生成密钥和配置）
yews init
# 输入你需要加密的文件名和加密后的输出文件名
# 如果要继续加文件，直接在 init 里一路追加，或者手动编辑 [[encryption.files]]

# 2. 根据配置文件加密敏感文件
yews e
# 或者使用全称
yews encrypt 

# 3. 解密敏感文件
yews d
# 或者使用全称
yews decrypt 
```

**不想增加额外的配置文件？** 完全可以只用命令行参数运行：

```bash
# 所有参数都可以通过命令行指定
yews encrypt -i config.toml -o config.enc.yaml -k .age/keys.txt -p "age1..."
yews decrypt -i config.enc.yaml -o config.toml -k .age/keys.txt
```

> 配置文件 `.yewseal.toml` 是可选的，只是为了避免每次都输入参数。

## 安装

### 从源码安装

```bash
git clone https://github.com/YewFence/YewSeal
cd YewSeal

# 使用 just（推荐）
just install    # 安装到 $GOPATH/bin，可全局使用

# 或手动安装
go install -ldflags "-s -w -X main.Version=$(git describe --tags --always --dirty 2>/dev/null || echo dev)" ./cmd/main.go
```

安装完成后可直接在任意目录使用 `yews` 命令。

如果只需要本地构建而不安装到全局：

```bash
just build      # 生产构建 → build/yews.exe
just dev        # 开发构建 → test/yews.exe
```

### 直接下载

从 Releases 页面下载适合你系统的预编译二进制文件。


#### Scoop（Windows）

```powershell
scoop bucket add YewNursery https://github.com/YewFence/YewNursery
scoop install YewSeal
```

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

2. **加密文件名**（默认：`wrangler.enc.toml.yaml`）
   - 输入加密后生成的文件名

3. **是否创建示例文件**（默认：否）
   - 建议创建，因为加密/解密过程会丢失 TOML 注释
   - 示例文件用于保留配置结构和注释说明

4. **是否创建 .sops.yaml**（默认：是）
   - 非必需，但便于直接使用 `sops` 命令
   - 推荐创建以保持团队协作的一致性

初始化结束后，生成的 `.yewseal.toml` 会统一使用 `[[encryption.files]]` 数组配置。
每一项都用 `plaintext` 表示明文文件、`encrypted` 表示加密文件，不再区分“单文件配置字段”和“批量配置字段”。

如果配置文件中有多项 `[[encryption.files]]`，后续直接运行 `yews encrypt` / `yews decrypt` 就会按整组配置处理。

#### 非交互式模式

使用命令行参数跳过交互提示：

```bash
# 完全自定义配置
yews init --input app.toml --output app.secret.toml.yaml --create-example --skip-sops-config

# 仅指定文件名
yews init -i myapp.toml -o myapp.enc.toml.yaml

# 强制覆盖已有配置
yews init --force
```

#### 初始化完成后会创建

- `.age/keys.txt` - Age 密钥对（私钥 + 公钥）
- `.yewseal.toml` - YewSeal 配置文件（包含公钥和 `[[encryption.files]]` 数组）
- `.sops.yaml` - SOPS 配置文件（如果选择创建）
- `.gitignore` - 自动添加需要忽略的文件
- `*.example.toml` - 示例配置文件（如果选择创建）

> 如果选择创建了示例配置文件，记得在将其提交到版本控制系统前删除其中的敏感内容

> 📝 **关于 `.sops.yaml`**：虽然 `yews` 工具本身不强制要求此配置文件（通过环境变量传递密钥），但仍**强烈推荐**创建它，原因如下：
> - 🤝 便于团队协作，确保使用统一的加密配置
> - 🔧 支持直接使用原生 `sops` 命令操作加密文件
> - 📋 符合 SOPS 官方最佳实践
> - 🎯 明确定义哪些文件应该被加密
>
> 现在如果你是通过 `.yewseal.toml` 里的 `[[encryption.files]]` 运行 `yews encrypt`，工具会在加密前自动把 `.sops.yaml` 同步成当前配置对应的规则。

⚠️ **重要提示**：请妥善保管 `.age/keys.txt` 私钥文件，丢失后无法恢复！

### 2. 加密配置文件

支持单文件和批量模式：

```bash
# 单文件模式
yews encrypt
yews e  # 简写

# 批量模式（扫描目录）
yews encrypt --dir ./configs --pattern "*.toml"
yews e --dir ./configs --pattern "*.yaml" --parallel 4  # 并行处理
```

支持的选项：
```bash
yews encrypt --input custom.toml --output custom.enc.toml.yaml
yews encrypt -i custom.toml -o custom.enc.toml.yaml --public-key "age1..."
yews encrypt --verbose  # 显示详细输出
```

### 3. 解密配置文件

从加密文件恢复原始配置：

```bash
# 单文件模式
yews decrypt
yews d  # 简写

# 批量模式
yews decrypt --dir ./configs --pattern "*.enc.toml.yaml"
yews d --dir ./configs --parallel 4
```

支持的选项：
```bash
yews decrypt --input custom.enc.toml.yaml --output custom.toml
yews decrypt -i custom.enc.toml.yaml -o custom.toml --verbose
```

### 4. 直接编辑加密文件

使用编辑器直接编辑加密文件（自动处理加密/解密）：

```bash
yews edit
```

指定编辑器：
```bash
yews edit --editor "code -w"
yews edit -e vim
yews edit --file custom.enc.toml.yaml
```

## 配置管理

### 配置持久化

YewSeal 支持通过配置文件管理常用设置，避免每次都输入参数。

#### 支持的配置文件路径

按优先级从高到低：
1. `{project}/.yewseal/.yewseal.toml`
2. `{project}/.config/.yewseal.toml`
3. `{project}/.yewseal.toml`

#### 创建配置文件

```bash
cp .yewseal.example.toml .yewseal.toml
```

#### 配置文件格式

```toml
[encryption]

[[encryption.files]]
# plaintext = 明文文件
# encrypted = 加密文件
plaintext = "wrangler.toml"
encrypted = "wrangler.enc.toml.yaml"

[[encryption.files]]
plaintext = ".dev.vars"
encrypted = ".dev.vars.enc.yaml"
format = "env"

[key]
# Age 公钥（用于加密）
public_key = "age1..."
# Age 私钥文件路径（只存储路径，不存储密钥值）
file_path = ".age/keys.txt"
```

#### 配置优先级

参数获取遵循以下优先级（从高到低）：

1. **命令行参数** - 最高优先级
2. **环境变量** - `SOPS_INPUT_FILE`, `SOPS_OUTPUT_FILE`, `AGE_KEY_FILE`, `SOPS_AGE_RECIPIENTS`
3. **配置文件** - `.yewseal.toml`
4. **默认值** - 最低优先级

#### 使用示例

```bash
# 使用配置文件中的 [[encryption.files]]
# 配了 1 项就处理 1 项，配了多项就按整组处理
yews encrypt
yews decrypt

# 命令行参数会覆盖配置文件
yews encrypt -i custom.toml -o custom.enc.toml.yaml

# 环境变量也会覆盖配置文件
export SOPS_INPUT_FILE="custom.toml"
yews encrypt
```

通过配置文件驱动时，`yews encrypt` / `yews decrypt` 还会自动确认所有 `plaintext` 明文文件都在 `.gitignore` 里；其中 `yews encrypt` 还会顺便重建 `.sops.yaml`。

#### 批量配置示例

```toml
[encryption]

[[encryption.files]]
plaintext = "app.toml"
encrypted = "app.enc.toml.yaml"

[[encryption.files]]
plaintext = ".dev.vars"
encrypted = ".dev.vars.enc.yaml"
format = "env"

[key]
public_key = "age1..."
file_path = ".age/keys.txt"
```

配置好以后，直接运行下面两条命令即可批量处理，不需要额外传 `--dir`：

```bash
yews encrypt
yews decrypt
```

⚠️ **安全提示**：配置文件只应存储密钥文件的路径，不要存储私钥值本身

## 密钥管理

工具支持多种方式提供 Age 密钥，优先级从高到低：

### 1. 环境变量（推荐用于 CI/CD）

```bash
# 直接传递私钥值
export SOPS_AGE_KEY="AGE-SECRET-KEY-..."
yews decrypt

# 或传递公钥用于加密
export SOPS_AGE_RECIPIENTS="age1..."
yews encrypt
```

### 2. 命令行参数

```bash
yews --key-file /path/to/keys.txt decrypt
yews encrypt --public-key "age1..."
```

### 3. 配置文件（推荐用于本地开发）

在 `.yewseal.toml` 中配置：

```toml
[key]
file_path = ".age/keys.txt"
public_key = "age1..."
```

### 4. 默认位置

默认从 `.age/keys.txt` 读取私钥，从 `.yewseal.toml` 读取公钥。

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
# 自行指定变量名和路径（需要提前创建）
yews sync --name AGE_KEY_FILE --path /yewseal
```

## 检查环境

检查所需外部工具是否已安装：

```bash
yews check
# 或
yews doctor
```

## Git 工作流

### ✅ 应该提交到版本控制

```bash
git add .gitignore .yewseal.toml wrangler.enc.toml.yaml
# git add .infisical.json  # 如果使用 Infisical
git add .sops.yaml  # 可选但推荐
git commit -m "feat: 添加加密配置"
```

### ❌ 永远不要提交

- `wrangler.toml` - 明文配置文件
- `.age/keys.txt` - 私钥文件

## CI/CD 集成

### GitHub Actions 示例

```yaml
name: Deploy
on: push

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install tools
        run: |
          # 仅 TOML 格式需要 remarshal
          pipx install remarshal

      - name: Decrypt configuration
        env:
          SOPS_AGE_KEY: ${{ secrets.AGE_KEY }}
        run: |
          ./yews decrypt

      - name: Deploy
        run: |
          npx wrangler deploy
```

## 故障排查

### 找不到工具

Age 和 SOPS 已内嵌，无需单独安装。仅 TOML 格式需要 remarshal：

```bash
# 检查 remarshal（仅 TOML 格式需要）
toml2yaml --version
yaml2toml --version
```

### 解密失败

检查密钥是否正确设置：

```bash
# 检查环境变量
echo $SOPS_AGE_KEY

# 或检查文件
cat .age/keys.txt
```

### TOML 格式问题

工具使用 remarshal 进行 TOML/YAML 转换，应该支持所有标准 TOML 特性，并且声明转换前后的文件会完全一致。如果遇到问题，请提交 Issue。

## 命令参考

完整的命令行参考请查看 [CLI.md](CLI.md)。

### 全局选项

```
--key-file, -k   指定 Age 私钥文件路径
--help, -h       显示帮助信息
--version, -v    显示版本信息
```

### 命令列表

| 命令 | 别名 | 说明 |
|------|------|------|
| `init` | - | 初始化项目，生成密钥和配置 |
| `encrypt` | `e` | 加密配置文件 |
| `decrypt` | `d` | 解密配置文件 |
| `edit` | - | 使用编辑器编辑加密文件 |
| `check` | `doctor` | 检查外部工具是否安装 |
| `sync` | - | 同步密钥到密钥管理服务 |

## 工作原理

```
TOML 加密: TOML → toml2yaml → YAML → sops encrypt → 加密 YAML
TOML 解密: 加密 YAML → sops decrypt → YAML → yaml2toml → TOML

其他格式: 原文件 → sops encrypt/decrypt → 加密/解密文件
```

YewSeal 内嵌了成熟的加密库，仅 TOML 格式转换依赖外部工具：
- **Age** (`filippo.io/age`) - 现代化的加密库（内嵌）
- **SOPS** (`github.com/getsops/sops/v3`) - 密钥管理库（内嵌）
- **Remarshal** - 配置格式转换工具（仅 TOML 需要，外部工具）

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request！

## 碎碎念

我写这个的原因是因为我真的不想把 `wrangler.toml` 直接放在 GitHub 上，虽然里面没有泄露就出事的敏感信息，但是会有自己的域名/项目名称/ `KV ID``D1 ID` / `R2 NAME` 之类的信息，放在公开仓库总觉得不太好。直接放在 Infisical 里又不太方便，毕竟每次改配置都得去网站上操作一遍。所以就想到使用 SOPS ，但 SOPS 原生不支持 TOML 格式，我又真的不想用 yaml，缩进地狱，所以就写了这个工具来帮忙做格式转换和加密解密的编排工作。

想着来都来了，增加一个方便点的包装吧，于是就有了短命令和配置文件持久化功能，省得每次都敲一大堆参数。包装都包装了，干脆支持所有 SOPS 支持的格式吧，万一以后有别的配置文件需要加密呢。支持都支持了，不如增加一个多文件批量加密解密的功能，省得每次都得写脚本。既然有了批量处理，那就顺便加个并行处理，都用 Go 了，不些并行多可惜。总之就是越做越多功能，最后变成了现在这个样子。

接下来就是密钥管理的问题了，Age 的密钥对生成很方便，而且公钥可以直接放在配置文件里，私钥只需要保存在本地就行了。再结合 Infisical 这种密钥管理服务，就能比较方便地在 CI/CD 里使用密钥了，它也可以实现一个项目一个密钥，减少爆破半径，非常完美。

关于安全性嘛，Age + SOPS 还是挺靠谱的，至少比我自己写加密算法要靠谱多了。

## 声明

本工具仅作为个人学习项目，未经充分测试和审计，不建议在高安全要求环境中使用。

## 致谢

本项目核心功能依赖于以下优秀的开源工具，感谢它们的贡献：

- [Age](https://github.com/FiloSottile/age) (BSD 3-Clause) - A simple, modern and secure encryption tool (and Go library) with small explicit keys, no config options, and UNIX-style composability.
- [SOPS](https://github.com/getsops/sops) (MPL 2.0) - Simple and flexible tool for managing secrets
- [Remarshal](https://github.com/remarshal-project/remarshal) (MIT) - Convert between CBOR, JSON, MessagePack, TOML, and YAML 1.1 & 1.2
- [Infisical CLI](https://github.com/Infisical/cli) (MIT) - The official CLI of Infisical

## 相关链接

- [Age](https://github.com/FiloSottile/age)
- [SOPS](https://github.com/getsops/sops)
- [Remarshal](https://github.com/remarshal-project/remarshal)
- [Infisical](https://infisical.com/)
- [Wrangler Configuration](https://developers.cloudflare.com/workers/wrangler/configuration/)
