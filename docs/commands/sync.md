# sync - 同步密钥

`sync` 用来把本地 Age 私钥文件推送到密钥管理服务，`sync pull` 用来把远端密钥拉回本地。目前支持 Infisical。

## 语法

```bash
yews sync [command options]
yews sync pull [command options]
```

## 选项

`sync` 和 `sync pull` 使用同一组选项。

### --provider, -p

密钥管理服务提供商，默认值是 `infisical`。

```bash
yews sync --provider infisical
```

### --key-file, -k

本地 Age 私钥文件路径，默认值是 `.age/keys.txt`。

```bash
yews sync --key-file .age/keys.txt
```

### --name, -n

远端密钥名称，默认值是 `AGE_KEY_FILE`。

```bash
yews sync --name AGE_KEY_FILE
```

### --project-id

Infisical 项目编号。

```bash
yews sync --project-id <project-id>
```

### --path

Infisical 中的路径或文件夹。

```bash
yews sync --path /yewseal
```

### --env, --environment

Infisical 环境名称。

```bash
yews sync --env dev
```

## 配置文件

这些选项也可以写入 `.yewseal.toml`。

```toml
[sync]
provider = "infisical"
project_id = "your-project-id"
environment = "dev"
path = "/yewseal"
secret_name = "AGE_KEY_FILE"
```

命令行参数优先于配置文件。

## 示例

```bash
# 推送本地私钥到 Infisical
yews sync \
  --project-id <project-id> \
  --env dev \
  --path /yewseal \
  --name AGE_KEY_FILE \
  --key-file .age/keys.txt

# 从 Infisical 拉取私钥
yews sync pull \
  --project-id <project-id> \
  --env dev \
  --path /yewseal \
  --name AGE_KEY_FILE \
  --key-file .age/keys.txt
```

## 团队协作

项目负责人可以运行 `yews init` 后用 `yews sync` 推送 `.age/keys.txt`，团队成员克隆仓库后用 `yews sync pull` 拉取私钥，再运行 `yews decrypt` 解密配置文件。

## 前置条件

使用 Infisical 前需要安装并登录 Infisical CLI。

```bash
infisical login
```

## 相关命令

[init](/commands/init) 会生成 Age 私钥，[decrypt](/commands/decrypt) 会使用私钥解密文件。
