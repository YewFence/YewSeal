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

输出详细的文件选择信息。

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

`view` 只按文件自身的格式输出，不进行跨格式转换；需要转换时，将解密后的内容交给相应格式转换工具。

## 相关命令

[decrypt](/commands/decrypt) 可以解密文件到磁盘，[edit](/commands/edit) 可以直接编辑加密文件。
