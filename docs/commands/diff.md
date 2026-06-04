# diff - 比较文件差异

比较明文文件与解密后的加密文件内容。

## 语法

```bash
yews diff [选项] [target]
```

## 参数

### target

要比较的明文文件路径。如果不指定，会根据配置自动查找。

```bash
yews diff config.toml
```

## 选项

### --format

为选定的目标指定格式覆盖（toml/yaml/json/env/ini）。

```bash
yews diff config.txt --format toml
```

### --verbose, -v

启用详细输出。

```bash
yews diff config.toml -v
```

## 工作原理

`diff` 命令会：
1. 读取明文文件
2. 根据 `.sops.yaml` 配置找到对应的加密文件
3. 解密加密文件
4. 比较两者内容
5. 显示差异（如果有）

## 示例

### 基本比较

```bash
# 比较明文和加密文件
yews diff config.toml
```

### 指定格式

```bash
# 指定文件格式
yews diff config.txt --format toml
```

### 详细输出

```bash
# 显示详细信息
yews diff config.toml -v
```

## 输出格式

如果文件内容相同，输出：
```text
Files are identical
```

如果文件内容不同，输出差异：
```diff
--- config.toml (plaintext)
+++ config.enc.toml.yaml (decrypted)
@@ -1,3 +1,3 @@
 [database]
 host = "localhost"
-password = "old_password"
+password = "new_password"
```

## 使用场景

### 验证加密

在加密文件后，验证加密是否正确：

```bash
# 加密文件
yews encrypt -i config.toml -o config.enc.toml.yaml

# 验证内容一致
yews diff config.toml
```

### 检查同步

检查本地明文文件是否与加密文件同步：

```bash
# 修改明文文件后
yews diff config.toml

# 如果有差异，重新加密
yews encrypt -i config.toml -o config.enc.toml.yaml
```

### CI/CD 验证

在 CI/CD 流程中验证配置文件：

```bash
#!/bin/bash
# 检查所有配置文件是否同步
for file in *.toml; do
  if ! yews diff "$file"; then
    echo "Error: $file is out of sync"
    exit 1
  fi
done
```

## 退出码

- `0` - 文件内容相同
- `1` - 文件内容不同或发生错误

可以在脚本中使用退出码：

```bash
if yews diff config.toml; then
  echo "Files are in sync"
else
  echo "Files differ, re-encrypting..."
  yews encrypt -i config.toml -o config.enc.toml.yaml
fi
```

## 注意事项

1. **格式规范化**：比较前会规范化格式，忽略空白和注释的差异
2. **密钥要求**：需要对应的 Age 私钥才能解密
3. **配置依赖**：依赖 `.sops.yaml` 配置来定位加密文件

## 相关命令

- [view](/commands/view) - 查看加密文件内容
- [decrypt](/commands/decrypt) - 解密文件
- [encrypt](/commands/encrypt) - 加密文件
