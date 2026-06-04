# edit - 编辑加密文件

使用 SOPS 编辑加密的配置文件。

## 语法

```bash
yews edit [选项]
```

## 选项

### --file, -f

要编辑的加密文件路径。

默认值：`wrangler.enc.toml.yaml`

```bash
yews edit -f config.enc.toml.yaml
```

### --editor, -e

指定编辑器命令（例如 'code -w', 'vim'）。

```bash
yews edit -f config.enc.toml.yaml -e "code -w"
```

如果不指定，会使用以下顺序查找编辑器：
1. `--editor` 选项
2. `EDITOR` 环境变量
3. 系统默认编辑器

## 工作原理

`edit` 命令会：
1. 使用 SOPS 解密文件到临时文件
2. 在编辑器中打开临时文件
3. 保存后自动重新加密
4. 更新原始加密文件

## 示例

### 使用默认编辑器

```bash
yews edit -f config.enc.toml.yaml
```

### 使用 VS Code

```bash
yews edit -f config.enc.toml.yaml -e "code -w"
```

注意：`-w` 参数让 VS Code 等待文件关闭后再返回。

### 使用 Vim

```bash
yews edit -f config.enc.toml.yaml -e vim
```

### 使用 Nano

```bash
yews edit -f config.enc.toml.yaml -e nano
```

## 编辑器配置

### VS Code

```bash
# 等待文件关闭
yews edit -f config.enc.toml.yaml -e "code -w"

# 或设置环境变量
export EDITOR="code -w"
yews edit -f config.enc.toml.yaml
```

### Vim/Neovim

```bash
export EDITOR=vim
yews edit -f config.enc.toml.yaml
```

### Emacs

```bash
export EDITOR="emacs -nw"
yews edit -f config.enc.toml.yaml
```

## 注意事项

1. **编辑器等待**：某些编辑器（如 VS Code）需要 `-w` 参数才能等待文件关闭
2. **临时文件**：SOPS 会在系统临时目录创建解密后的临时文件，编辑完成后自动删除
3. **格式保持**：编辑时保持原始格式（YAML），不会转换为其他格式
4. **自动加密**：保存并关闭编辑器后，SOPS 会自动重新加密文件

## 相关命令

- [view](/commands/view) - 查看加密文件内容（只读）
- [decrypt](/commands/decrypt) - 解密文件到明文
- [encrypt](/commands/encrypt) - 加密明文文件
- [diff](/commands/diff) - 比较明文和加密文件
