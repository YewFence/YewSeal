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
        run: yews decrypt --strict

      - name: Deploy
        run: wrangler deploy
```

把 `.age/keys.txt` 里的私钥值存为仓库 Secret（上面的 `AGE_KEY`），解密时 `SOPS_AGE_KEY` 环境变量会被直接使用，私钥解析优先级见[配置说明 - 私钥读取](/guide/configuration#私钥读取)。

## 配合 Infisical

如果私钥托管在 Infisical，可以独立使用其 CLI 先导出当前部署环境的身份，再交给 `yews`。参考脚本和认证说明见[外部私钥来源](/guide/private-keys#infisical-参考脚本)。YewSeal 不调用 Infisical，也不要求生产环境与开发机共用私钥。

## 其他 CI 系统

任何 CI 都适用同样的模式：

1. 安装 `yews`（mise、go install 或下载 Release 二进制，也可以用 [Docker 镜像](/guide/docker)）
2. 通过环境变量或密钥文件提供 Age 私钥
3. 运行 `yews decrypt --strict`，仅在成功退出后执行部署

开发环境可以容忍部分文件因身份不匹配而跳过，但部署通常要求全部选中文件成功解密。严格模式也可以通过 `YEWSEAL_STRICT=true` 启用；结果分类和退出码见[解密结果与严格模式](/guide/decryption-results)。
