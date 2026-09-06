# Docker 运行

如果你的环境里已经有 Docker，但不想额外安装 `yews`，可以直接跑容器镜像 `ghcr.io/yewfence/yew-seal:latest`。可以根据需要自行替换示例命令的镜像标签。

## 文件权限

Linux/macOS 下，所有会写入挂载目录 `/work` 的命令（例如 `init`、`encrypt`、`decrypt`）都建议显式传入宿主用户 ID，避免生成的文件属于 `root`。Windows 一般可以省略 `--user`：

```bash
docker run --rm -it \
  --user "$(id -u):$(id -g)" \
  -v "$PWD:/work" \
  ghcr.io/yewfence/yew-seal:latest --help
```

## 初始化

```bash
docker run --rm -it \
  --user "$(id -u):$(id -g)" \
  -v "$PWD:/work" \
  ghcr.io/yewfence/yew-seal:latest init
```

## 加密 / 解密

```bash
# 加密
docker run --rm \
  --user "$(id -u):$(id -g)" \
  -v "$PWD:/work" \
  ghcr.io/yewfence/yew-seal:latest encrypt

# 解密
docker run --rm \
  --user "$(id -u):$(id -g)" \
  -v "$PWD:/work" \
  ghcr.io/yewfence/yew-seal:latest decrypt
```

## 通过环境变量注入私钥

如果私钥通过环境变量注入（例如 CI 环境），挂载密钥文件不是必须的：

```bash
docker run --rm \
  -e SOPS_AGE_KEY="$SOPS_AGE_KEY" \
  --user "$(id -u):$(id -g)" \
  -v "$PWD:/work" \
  ghcr.io/yewfence/yew-seal:latest decrypt
```

## 从 Infisical 导出私钥

在宿主机独立使用 Infisical CLI 导出当前环境的私钥，再挂载给容器。参考脚本见[外部私钥来源](/guide/private-keys#infisical-参考脚本)，YewSeal 镜像不负责 Infisical 的认证或远端操作。

## 限制

`edit` 不建议通过 Docker 运行，因为它依赖宿主编辑器。
