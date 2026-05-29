# 快速开始

YewSeal 是一个基于 SOPS 和 Age 的配置文件加密管理工具，支持 TOML、YAML、JSON、ENV、INI 等多种格式。

## 安装

### 从源码构建

```bash
git clone https://github.com/YewFence/YewSeal.git
cd YewSeal
mise run build
```

构建完成后，二进制文件位于 `build/yews`。

### 添加到 PATH

```bash
# 复制到系统路径
sudo cp build/yews /usr/local/bin/

# 或者添加到用户路径
mkdir -p ~/.local/bin
cp build/yews ~/.local/bin/
export PATH="$HOME/.local/bin:$PATH"
```

## 初始化项目

在项目目录中运行 `init` 命令初始化 Age 密钥和配置：

```bash
yews init
```

该命令会：
1. 生成 Age 密钥对（保存在 `.age/keys.txt`）
2. 创建 `.sops.yaml` 配置文件
3. 交互式添加第一个配置文件条目

### 非交互模式

如果需要在 CI/CD 或脚本中使用，可以使用非交互模式：

```bash
yews init \
  --input config.toml \
  --output config.enc.toml.yaml \
  --format toml \
  --create-example
```

## 基本使用

### 加密配置文件

```bash
# 加密单个文件
yews encrypt -i config.toml -o config.enc.toml.yaml

# 批量加密目录中的文件
yews encrypt --dir ./configs --pattern "*.toml"
```

### 解密配置文件

```bash
# 解密单个文件
yews decrypt -i config.enc.toml.yaml -o config.toml

# 批量解密
yews decrypt --dir ./configs --pattern "*.enc.toml.yaml"
```

### 编辑加密文件

```bash
# 使用默认编辑器
yews edit -f config.enc.toml.yaml

# 指定编辑器
yews edit -f config.enc.toml.yaml -e "code -w"
```

### 查看加密文件内容

```bash
# 输出到标准输出
yews view config.enc.toml.yaml
```

### 比较差异

```bash
# 比较明文文件和加密文件的差异
yews diff config.toml
```

## 下一步

- [命令参考](/commands/init) - 查看所有命令的详细说明
- [配置说明](/guide/configuration) - 了解如何配置 YewSeal
- [Shell 补全](/guide/completion) - 启用命令行自动补全
