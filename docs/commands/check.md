# check - 检查依赖

检查是否安装了所需的外部工具。

## 语法

```bash
yews check
yews doctor
```

别名：`doctor`

## 功能

`check` 命令会检查以下工具是否已安装：

1. **SOPS** - 用于加密和解密配置文件
2. **Age** - 用于密钥生成和管理

## 示例

```bash
# 检查依赖
yews check

# 使用别名
yews doctor
```

## 输出示例

### 所有依赖已安装

```
✓ SOPS is installed (version 3.8.1)
✓ Age is installed (version 1.1.1)

All required tools are installed!
```

### 缺少依赖

```
✗ SOPS is not installed
✓ Age is installed (version 1.1.1)

Missing required tools. Please install them before using YewSeal.
```

## 安装依赖

### SOPS

#### macOS

```bash
brew install sops
```

#### Linux

```bash
# 下载最新版本
wget https://github.com/getsops/sops/releases/download/v3.8.1/sops-v3.8.1.linux.amd64
sudo mv sops-v3.8.1.linux.amd64 /usr/local/bin/sops
sudo chmod +x /usr/local/bin/sops
```

#### Windows

```powershell
# 使用 Chocolatey
choco install sops

# 或使用 Scoop
scoop install sops
```

### Age

#### macOS

```bash
brew install age
```

#### Linux

```bash
# 下载最新版本
wget https://github.com/FiloSottile/age/releases/download/v1.1.1/age-v1.1.1-linux-amd64.tar.gz
tar xzf age-v1.1.1-linux-amd64.tar.gz
sudo mv age/age /usr/local/bin/
sudo mv age/age-keygen /usr/local/bin/
sudo chmod +x /usr/local/bin/age*
```

#### Windows

```powershell
# 使用 Chocolatey
choco install age.portable

# 或使用 Scoop
scoop install age
```

## 为什么需要这些工具

### SOPS

SOPS (Secrets OPerationS) 是 Mozilla 开发的加密工具，用于：
- 加密和解密配置文件
- 支持多种密钥管理系统（Age, PGP, AWS KMS, GCP KMS, Azure Key Vault）
- 保持文件格式可读性（只加密值，不加密键）

### Age

Age 是一个现代化的文件加密工具，用于：
- 生成密钥对
- 提供简单安全的加密方案
- 替代传统的 PGP/GPG

## 注意事项

1. **版本要求**：建议使用最新稳定版本
2. **PATH 配置**：确保工具在系统 PATH 中可访问
3. **权限问题**：Linux/macOS 上可能需要 sudo 权限安装

## 相关命令

- [init](/commands/init) - 初始化项目（需要 Age）
- [encrypt](/commands/encrypt) - 加密文件（需要 SOPS）
- [decrypt](/commands/decrypt) - 解密文件（需要 SOPS）
- [edit](/commands/edit) - 编辑加密文件（需要 SOPS）
