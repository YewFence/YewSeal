# SOPS Config Tool

一个用于管理加密配置文件的 CLI 工具，使用 SOPS + Age 加密，专为 Cloudflare Wrangler 配置设计。

## 功能特性

- 🔐 使用 Age 加密算法保护敏感配置
- 📦 TOML ↔ YAML 格式自动转换
- 🔑 支持环境变量和文件两种密钥管理方式
- ✨ 简单易用的命令行界面
- 🎯 专为 `wrangler.toml` 配置文件优化

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
cd sops-config-tool
go build -o sops-config-tool.exe ./cmd
```

### 直接下载

从 Releases 页面下载适合你系统的预编译二进制文件。

## 使用指南

### 1. 初始化项目

首次使用时，在项目根目录运行：

```bash
./sops-config-tool init
```

这将会：
- 生成 Age 密钥对（存储在 `.age/keys.txt` 和 `.age.pub`）
- 创建 `.sops.yaml` 配置文件
- 创建 `.gitignore` 文件
- 生成 `wrangler.example.toml` 模板

⚠️ **重要提示**：请妥善保管 `.age/keys.txt` 私钥文件，丢失后无法恢复！

### 2. 加密配置文件

将 `wrangler.toml` 加密为 `wrangler.enc.yaml`：

```bash
./sops-config-tool encrypt
```

支持的选项：
```bash
./sops-config-tool encrypt --input custom.toml --output custom.enc.yaml
```

### 3. 解密配置文件

从加密文件恢复原始配置：

```bash
./sops-config-tool decrypt
```

支持的选项：
```bash
./sops-config-tool decrypt --input custom.enc.yaml --output custom.toml
```

### 4. 直接编辑加密文件

使用默认编辑器编辑加密文件（自动处理加密/解密）：

```bash
./sops-config-tool edit
```

指定编辑器：
```bash
./sops-config-tool edit --editor "code -w"
./sops-config-tool edit --editor vim
```

## 密钥管理

工具支持两种方式提供 Age 私钥，优先级从高到低：

### 1. 环境变量（推荐用于 CI/CD）

```bash
export SOPS_AGE_KEY="AGE-SECRET-KEY-..."
./sops-config-tool decrypt
```

### 2. 密钥文件（推荐用于本地开发）

默认从 `.age/keys.txt` 读取，或指定文件：

```bash
./sops-config-tool --key-file /path/to/keys.txt decrypt
```

## Git 工作流

### 提交到版本控制

```bash
git add .sops.yaml .gitignore wrangler.enc.yaml wrangler.example.toml
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
          ./sops-config-tool decrypt
      
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
├── .sops.yaml                # SOPS 配置
├── .gitignore                # Git 忽略规则
├── wrangler.toml             # 明文配置（不提交）
├── wrangler.enc.yaml         # 加密配置（提交）
├── wrangler.example.toml     # 配置模板（提交）
└── sops-config-tool.exe      # 工具二进制文件
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
sops-config-tool init [--force]
```

选项：
- `--force`: 强制覆盖已有配置

#### encrypt

加密 TOML 文件为 YAML 格式。

```bash
sops-config-tool encrypt [--input FILE] [--output FILE]
```

选项：
- `--input`: 输入 TOML 文件（默认：wrangler.toml）
- `--output`: 输出加密 YAML 文件（默认：wrangler.enc.yaml）

#### decrypt

解密 YAML 文件为 TOML 格式。

```bash
sops-config-tool decrypt [--input FILE] [--output FILE]
```

选项：
- `--input`: 输入加密 YAML 文件（默认：wrangler.enc.yaml）
- `--output`: 输出 TOML 文件（默认：wrangler.toml）

#### edit

使用编辑器编辑加密文件。

```bash
sops-config-tool edit [--file FILE] [--editor COMMAND]
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
