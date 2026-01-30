# YewSeal

一个用于管理加密 TOML 配置文件的 CLI 工具，使用 SOPS + Age 加密，补充 SOPS 缺失的 TOML 文件加密支持

## 功能特性

- 📦 TOML ↔ YAML 格式转换后加密，沿用 SOPS 已有功能
- 🔑 支持环境变量和文件两种密钥管理方式
- ✨ 简单易用的命令行界面
- 🎯 专为 `toml` 配置文件优化

## 前置要求

安装以下工具：

- [Age](https://github.com/FiloSottile/age) - 加密工具
- [SOPS](https://github.com/getsops/sops) - 密钥管理

### Windows 安装（Scoop）

```powershell
scoop install age
scoop install sops
```

### macOS 安装（Homebrew）

```bash
brew install age
brew install sops
```

### Linux 安装

请参考各工具的官方文档。

## 安装

### 从源码编译

```bash
git clone <repository>
cd YewSeal
go build -o yews.exe ./cmd
```

### 直接下载

从 Releases 页面下载适合你系统的预编译二进制文件。

## 使用指南

### 1. 初始化项目

首次使用时，在项目根目录运行：

```bash
./yews init
```

这将会：
- 生成 Age 密钥对（存储在 `.age/keys.txt` 和 `.age.pub`）
- 创建 `.sops.yaml` 配置文件（可选但推荐）
- 在 `.gitignore` 文件中补充需要忽略的文件

> 📝 **关于 `.sops.yaml`**：虽然 `yews` 工具本身不强制要求此配置文件（通过环境变量传递密钥），但仍**强烈推荐**创建它，原因如下：
> - 🤝 便于团队协作，确保使用统一的加密配置
> - 🔧 支持直接使用原生 `sops` 命令操作加密文件
> - 📋 符合 SOPS 官方最佳实践
> - 🎯 明确定义哪些文件应该被加密

⚠️ **重要提示**：请妥善保管 `.age/keys.txt` 私钥文件，丢失后无法恢复！

### 2. 加密配置文件

将 `wrangler.toml` 加密为 `wrangler.enc.yaml`：

```bash
./yews encrypt
# 或者使用简写
./yews e
```

支持的选项：
```bash
./yews encrypt --input custom.toml --output custom.enc.yaml
```

### 3. 解密配置文件

从加密文件恢复原始配置：

```bash
./yews decrypt
# 或者使用简写
./yews d
```

支持的选项：
```bash
./yews decrypt --input custom.enc.yaml --output custom.toml
```

### 4. 直接编辑加密文件

使用默认编辑器编辑加密文件（自动处理加密/解密）：

```bash
./yews edit
```

指定编辑器：
```bash
./yews edit --editor "code -w"
./yews edit --editor vim
```

## 配置管理

### 配置持久化

YewSeal 支持通过配置文件管理常用设置，避免每次都输入参数。YewSeal 会从项目根目录起的三个路径读取配置文件，如果没有读取到则回退到默认逻辑

#### 支持的配置文件路径
- `{path-to-project}/.yewseal/.yewseal.toml`
- `{path-to-project}/.config/.yewseal.toml`
- `{path-to-project}/.yewseal.toml`

#### 1. 创建配置文件

```bash
# 创建配置文件
cp .yewseal.example.toml .yewseal.toml
```

#### 2. 编辑配置文件

编辑 `.yewseal.toml`：

```toml
[encryption]
# 需要加密的文件（TOML 格式）
input_file = "wrangler.toml"

# 加密后的输出文件（YAML 格式）
output_file = "wrangler.enc.yaml"

[key]
# Age 私钥文件路径（只存储路径，不存储密钥值）
file_path = ".age/key.txt"
```

#### 3. 配置优先级

参数获取遵循以下优先级（从高到低）：

1. **命令行参数** - 最高优先级
2. **环境变量** - `SOPS_INPUT_FILE`, `SOPS_OUTPUT_FILE`, `AGE_KEY_FILE`
3. **配置文件** - `.yewseal/.config/.yewseal.toml`
4. **默认值** - 最低优先级

#### 4. 使用示例

```bash
# 使用配置文件中的设置
./yews encrypt
./yews decrypt

# 命令行参数会覆盖配置文件
./yews encrypt -i custom.toml -o custom.enc.yaml

# 环境变量也会覆盖配置文件
export SOPS_INPUT_FILE="custom.toml"
./yews encrypt
```

⚠️ **安全提示**：配置文件只应存储密钥文件的路径，不要存储密钥值本身

### 命令简称

为了提高效率，加密和解密命令支持简写：

```bash
# 加密（encrypt 的简称）
./yews e

# 解密（decrypt 的简称）
./yews d
```

## 密钥管理

工具支持多种方式提供 Age 私钥，优先级从高到低：

### 1. 环境变量（推荐用于 CI/CD）

```bash
export SOPS_AGE_KEY="AGE-SECRET-KEY-..."
./yews decrypt
```

### 2. 命令行参数

```bash
./yews --key-file /path/to/keys.txt decrypt
```

### 3. 配置文件（推荐用于本地开发）

在 `.yewseal.toml` 中配置密钥文件路径：

```toml
[key]
file_path = ".age/key.txt"
```

### 4. 默认位置

默认从 `.age/keys.txt` 读取。

## Git 工作流

### 提交到版本控制

```bash
git add .gitignore wrangler.enc.yaml
# .sops.yaml 是可选的，但推荐提交以便团队协作
git add .sops.yaml
git commit -m "Add encrypted configuration"
```

### ⚠️ 永远不要提交

- `wrangler.toml` - 明文配置文件
- `.age/keys.txt` - 私钥文件
- `*.tmp.*` - 临时文件

## CI/CD 集成

### GitHub Actions 示例

```yaml
name: Deploy
on: push

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Install tools
        run: |
          # 安装 age 和 sops
          curl -Lo age.tar.gz https://github.com/FiloSottile/age/releases/download/v1.1.1/age-v1.1.1-linux-amd64.tar.gz
          tar xf age.tar.gz
          sudo mv age/age /usr/local/bin/
          
          curl -Lo sops https://github.com/getsops/sops/releases/download/v3.8.1/sops-v3.8.1.linux.amd64
          chmod +x sops
          sudo mv sops /usr/local/bin/
      
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

确保 `age` 和 `sops` 在 PATH 中：

```bash
age --version
sops --version
```

### 解密失败

检查密钥是否正确设置：

```bash
# 环境变量
echo $SOPS_AGE_KEY

# 或文件
cat .age/keys.txt
```

### TOML 格式问题

工具使用 Go 库进行 TOML/YAML 转换，应该支持所有标准 TOML 特性。如果遇到问题，请提交 Issue。

## 项目结构

```
.
├── .age/
│   └── keys.txt              # Age 私钥（不提交）
├── .age.pub                  # Age 公钥
├── .sops.yaml                # SOPS 配置（可选但推荐）
├── .gitignore                # Git 忽略规则
├── wrangler.toml             # 明文配置（不提交）
├── wrangler.enc.yaml         # 加密配置（提交）
└── yews.exe                  # 工具二进制文件
```

## 命令参考

### 全局选项

```
--key-file, -k   指定 Age 私钥文件路径
--verbose        显示详细输出
--help, -h       显示帮助信息
--version, -v    显示版本信息
```

### 子命令

#### init

初始化项目，生成密钥和配置文件。

```bash
yews init [--force]
```

选项：
- `--force`: 强制覆盖已有配置

#### encrypt

加密 TOML 文件为 YAML 格式。

```bash
yews encrypt [--input FILE] [--output FILE]
```

选项：
- `--input`: 输入 TOML 文件（默认：wrangler.toml）
- `--output`: 输出加密 YAML 文件（默认：wrangler.enc.yaml）

#### decrypt

解密 YAML 文件为 TOML 格式。

```bash
yews decrypt [--input FILE] [--output FILE]
```

选项：
- `--input`: 输入加密 YAML 文件（默认：wrangler.enc.yaml）
- `--output`: 输出 TOML 文件（默认：wrangler.toml）

#### edit

使用编辑器编辑加密文件。

```bash
yews edit [--file FILE] [--editor COMMAND]
```

选项：
- `--file`: 要编辑的加密文件（默认：wrangler.enc.yaml）
- `--editor`: 编辑器命令（默认使用 $EDITOR 环境变量或 SOPS 默认编辑器）

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request！

## 相关链接

- [Age](https://github.com/FiloSottile/age)
- [SOPS](https://github.com/getsops/sops)
- [Cloudflare Wrangler](https://developers.cloudflare.com/workers/wrangler/)
