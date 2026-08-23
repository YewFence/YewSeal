# CI/CD 集成

CI 环境中没有交互终端，也没有本地 `.age/keys.txt`，通常通过环境变量注入 Age 私钥，再运行 `yews decrypt` 还原配置文件。

## GitHub Actions

在仓库的 `mise.toml` 中声明 `github:YewFence/YewSeal` 依赖后，可以用 [mise-action](https://github.com/jdx/mise-action) 安装，配合仓库 Secret 注入私钥：

```yaml
name: Deploy
on: push

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7
        with:
          persist-credentials: false

      - uses: jdx/mise-action@v4

      - name: Decrypt configuration
        env:
          SOPS_AGE_KEY: ${{ secrets.AGE_KEY }}
        run: yews decrypt

      - name: Deploy
        run: wrangler deploy
```

把 `.age/keys.txt` 里的私钥值存为仓库 Secret（上面的 `AGE_KEY`），解密时 `SOPS_AGE_KEY` 环境变量会被直接使用，私钥解析优先级见[配置说明 - 私钥读取](/guide/configuration#私钥读取)。

## 配合 Infisical

如果私钥托管在 Infisical，可以在 CI 里用 [Infisical CLI](https://infisical.com/docs/cli/overview) 先导出私钥，再交给 `yews`：

```bash
infisical secrets get AGE_KEY_FILE --plain > .age/keys.txt
yews decrypt
```

Infisical CLI 在 CI 中需要通过服务令牌（`INFISICAL_TOKEN`）认证，具体见 Infisical 官方文档。

## 其他 CI 系统

任何 CI 都适用同样的模式：

1. 安装 `yews`（mise、go install 或下载 Release 二进制，也可以用 [Docker 镜像](/guide/docker)）
2. 通过环境变量或密钥文件提供 Age 私钥
3. 运行 `yews decrypt` 后执行部署
