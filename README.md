# YewSeal

一个用于管理加密 TOML 配置文件的 CLI 工具，使用 SOPS + Age 加密，补充 SOPS 缺失的 TOML 文件加密支持。

## 功能特性

- 📦 TOML ↔ YAML 格式转换后加密，沿用 SOPS 已有功能
- 🔑 支持环境变量和文件两种密钥管理方式
- ✨ 简单易用的命令行界面
- 💤 简短的命令别名，提高效率
- 🛠️ 支持配置文件持久化常用设置
- 🎯 专为 `toml` 配置文件优化

## 前置要求

安装以下工具：

- [Age](https://github.com/FiloSottile/age) - 加密工具
- [SOPS](https://github.com/getsops/sops) - 密钥管理
- [Remarshal](https://github.com/remarshal-project/remarshal) - TOML/YAML 格式转换工具

### Windows 安装（Scoop）

```powershell
scoop install age
scoop install sops
pipx install remarshal
```

### macOS 安装（Homebrew）

```bash
brew install age
brew install sops
pipx install remarshal
```

### Linux 安装

请参考各工具的官方文档。

## 安装

### 从源码编译

```bash
git clone https://github.com/YewFence/YewSeal
cd YewSeal
make build    # 生产构建 → build/yews.exe
# 或
make dev      # 开发构建 → test/yews.exe
```

### 直接下载

从 Releases 页面下载适合你系统的预编译二进制文件。

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
- `.yewseal.toml` - YewSeal 配置文件（包含公钥和文件路径）
- `.sops.yaml` - SOPS 配置文件（如果选择创建）
- `.gitignore` - 自动添加需要忽略的文件
- `*.example.toml` - 示例配置文件（如果选择创建）

> 如果选择创建了示例配置文件，记得在将其提交到版本控制系统前删除其中的敏感内容

> 📝 **关于 `.sops.yaml`**：虽然 `yews` 工具本身不强制要求此配置文件（通过环境变量传递密钥），但仍**强烈推荐**创建它，原因如下：
> - 🤝 便于团队协作，确保使用统一的加密配置
> - 🔧 支持直接使用原生 `sops` 命令操作加密文件
> - 📋 符合 SOPS 官方最佳实践
> - 🎯 明确定义哪些文件应该被加密

⚠️ **重要提示**：请妥善保管 `.age/keys.txt` 私钥文件，丢失后无法恢复！

### 2. 加密配置文件

将 TOML 配置文件加密为 YAML 格式：

```bash
yews encrypt
# 或者使用简写
yews e
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
yews decrypt
# 或者使用简写
yews d
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
# 需要加密的文件（TOML 格式）
input_file = "wrangler.toml"
# 加密后的输出文件（YAML 格式）
output_file = "wrangler.enc.toml.yaml"

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
# 使用配置文件中的设置
yews encrypt
yews decrypt

# 命令行参数会覆盖配置文件
yews encrypt -i custom.toml -o custom.enc.toml.yaml

# 环境变量也会覆盖配置文件
export SOPS_INPUT_FILE="custom.toml"
yews encrypt
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

## Git 工作流

### ✅ 应该提交到版本控制

```bash
git add .gitignore .yewseal.toml wrangler.enc.toml.yaml
git add .sops.yaml  # 可选但推荐
git commit -m "feat: 添加加密配置"
```

### ❌ 永远不要提交

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
      - uses: actions/checkout@v4

      - name: Install tools
        run: |
          # 安装 age
          curl -Lo age.tar.gz https://github.com/FiloSottile/age/releases/download/v1.2.0/age-v1.2.0-linux-amd64.tar.gz
          tar xf age.tar.gz
          sudo mv age/age /usr/local/bin/

          # 安装 sops
          curl -Lo sops https://github.com/getsops/sops/releases/download/v3.9.0/sops-v3.9.0.linux.amd64
          chmod +x sops
          sudo mv sops /usr/local/bin/

          # 安装 remarshal
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

确保 `age`、`sops`、`toml2yaml`、`yaml2toml` 在 PATH 中：

```bash
age --version
sops --version
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

工具使用 remarshal 进行 TOML/YAML 转换，应该支持所有标准 TOML 特性。如果遇到问题，请提交 Issue。

## 命令参考

### 全局选项

```
--key-file, -k   指定 Age 私钥文件路径
--help, -h       显示帮助信息
--version, -v    显示版本信息
```

### init

初始化项目，生成密钥和配置文件。

```bash
yews init [选项]
```

| 选项 | 简写 | 说明 |
|------|------|------|
| `--force` | `-f` | 强制覆盖已有配置 |
| `--input` | `-i` | 原始配置文件名（非交互模式） |
| `--output` | `-o` | 加密输出文件名（非交互模式） |
| `--create-example` | | 创建示例文件（非交互模式） |
| `--skip-sops-config` | | 跳过创建 .sops.yaml（非交互模式） |

### encrypt

加密 TOML 文件为 YAML 格式。

```bash
yews encrypt [选项]
yews e [选项]  # 简写
```

| 选项 | 简写 | 默认值 | 说明 |
|------|------|--------|------|
| `--input` | `-i` | `wrangler.toml` | 输入 TOML 文件 |
| `--output` | `-o` | `wrangler.enc.toml.yaml` | 输出加密 YAML 文件 |
| `--public-key` | `-p` | | Age 公钥 |
| `--verbose` | `-v` | | 显示详细输出 |

### decrypt

解密 YAML 文件为 TOML 格式。

```bash
yews decrypt [选项]
yews d [选项]  # 简写
```

| 选项 | 简写 | 默认值 | 说明 |
|------|------|--------|------|
| `--input` | `-i` | `wrangler.enc.toml.yaml` | 输入加密 YAML 文件 |
| `--output` | `-o` | `wrangler.toml` | 输出 TOML 文件 |
| `--verbose` | `-v` | | 显示详细输出 |

### edit

使用编辑器编辑加密文件。

```bash
yews edit [选项]
```

| 选项 | 简写 | 默认值 | 说明 |
|------|------|--------|------|
| `--file` | `-f` | `wrangler.enc.toml.yaml` | 要编辑的加密文件 |
| `--editor` | `-e` | `$EDITOR` | 编辑器命令 |

## 工作原理

```
加密流程: TOML → toml2yaml → YAML → sops encrypt → 加密 YAML
解密流程: 加密 YAML → sops decrypt → YAML → yaml2toml → TOML
```

YewSeal 本身不实现加密算法，而是编排三个成熟的外部工具：
- **Age** - 现代化的加密工具
- **SOPS** - Mozilla 出品的密钥管理工具
- **Remarshal** - 配置格式转换工具

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request！

## 相关链接

- [Age](https://github.com/FiloSottile/age)
- [SOPS](https://github.com/getsops/sops)
- [Remarshal](https://github.com/remarshal-project/remarshal)
- [Cloudflare Wrangler](https://developers.cloudflare.com/workers/wrangler/)
