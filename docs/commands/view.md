# view - 查看加密文件

`view` 会把加密文件解密后的明文写到标准输出，不会写入明文文件。

## 语法

```bash
yews view [command options] <target>
```

## 参数

`target` 必须是一个目标文件，可以是配置里声明的明文路径或加密路径，也可以是符合文件名协议的加密文件路径。

```bash
yews view config.enc.toml
```

## 选项

### --format

指定输出格式，支持 `toml`、`yaml`、`json`、`env`、`ini` 和 `binary`。

```bash
yews view config.enc.yaml --format toml
```

### --verbose, -v

输出详细的文件选择信息。

```bash
yews view config.enc.toml -v
```

## 示例

```bash
# 查看加密文件
yews view config.enc.toml

# 转换为 TOML 输出
yews view config.enc.yaml --format toml

# 转换为 JSON 后交给 jq
yews view config.enc.yaml --format json | jq '.database'
```

## 与 decrypt 的区别

`view` 只写标准输出，适合临时查看和管道处理。`decrypt` 会写入明文文件，并带有覆盖保护。

## 相关命令

[decrypt](/commands/decrypt) 可以解密文件到磁盘，[edit](/commands/edit) 可以直接编辑加密文件。
