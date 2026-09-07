# view - 查看加密文件

`view` 会把加密文件解密后的明文写到标准输出，不会写入明文文件。

## 语法

```bash
yews view [command options] <target>
```

## 参数

`target` 必须命中已登记文件的明文路径或加密路径，格式使用项目配置或已登记路径的推断结果。

```bash
yews view config.enc.toml
```

## 选项

### --verbose, -v

向 stderr 输出详细的文件选择与解密信息，stdout 始终只包含明文。

```bash
yews view config.enc.toml -v
```

## 示例

```bash
# 查看加密文件
yews view config.enc.toml

# 将已登记的 JSON 文件解密后交给 jq
yews view config.enc.json | jq '.database'
```

## 与 decrypt 的区别

`view` 只写标准输出，适合临时查看和管道处理。`decrypt` 会写入明文文件，并带有覆盖保护。

两者共用完整的配置选择与历史解密语义：格式来自配置或已登记路径，当前 recipient alias 失效时警告后继续，实际解密依据密文 metadata 与当前 Identity bundle。`view` 保持单目标；无法解密时失败，不生成空明文冒充成功，也不更新 `.gitignore` 或 `.sops.yaml`。

`view` 只按文件自身的格式输出，不进行跨格式转换；需要转换时，将解密后的内容交给相应格式转换工具。

## 相关命令

[decrypt](/commands/decrypt) 可以解密文件到磁盘，[edit](/commands/edit) 可以直接编辑加密文件。
