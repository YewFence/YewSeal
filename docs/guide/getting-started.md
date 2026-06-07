# 快速开始

YewSeal 是一个基于 SOPS 和 Age 的配置文件加密管理工具，支持 TOML、YAML、JSON、ENV、INI 和二进制文件。TOML 文件会先转换为 YAML，再交给 SOPS 加密。

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
  --output config.enc.toml.yaml \
  --format toml \
  --create-example
```

## 基本使用

### 加密配置文件

```bash
# 加密配置里的所有文件
yews encrypt

# 加密单个明文文件，输出路径会按文件名推断
yews encrypt config.toml

# 加密单个明文文件，并指定输出路径
yews encrypt config.toml -o config.enc.toml.yaml

# 扫描目录并加密匹配的文件
yews encrypt ./configs --pattern "*.toml"
```

### 解密配置文件

```bash
# 解密配置里的所有文件
yews decrypt

# 解密单个加密文件，输出路径会按文件名推断
yews decrypt config.enc.toml.yaml

# 解密单个加密文件，并指定输出路径
yews decrypt config.enc.toml.yaml -o config.toml

# 扫描目录并解密匹配的加密文件
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

## 配置文件示例

初始化后会生成类似下面的 `.yewseal.toml`：

```toml
[key]
file_path = ".age/keys.txt"
public_key = "age1xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"

[[encryption.files]]
plaintext = "config.toml"
encrypted = "config.enc.toml.yaml"
```

也可以用分组配置让 YewSeal 扫描一批文件：

```toml
[[encryption.groups]]
patterns = ["*.toml", "*.yaml", "!*.enc.toml.yaml", "!*.enc.yaml"]
format_rules = [".dev.vars=env"]
unknown_as_binary = false
```

## 下一步

[命令参考](/commands/init) 可以查看所有命令选项，[配置说明](/guide/configuration) 可以了解 `.yewseal.toml`、`.sops.yaml` 和密钥同步的配置方式。
