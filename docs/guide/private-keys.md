# 外部私钥来源

YewSeal 不负责私钥的远端存储、上传、下载或分发，不提供 `sync` 命令或 Provider 集成。解密所需的 Age identity bundle 由开发者、机器或部署环境自行提供；身份读取优先级见[配置说明](/guide/configuration#私钥读取)。

个人私钥路径通过 `--key-file` 或环境变量指定，不写入 `.yewseal.toml`。未显式提供身份时，YewSeal 最后尝试当前工作目录下的 `.age/keys.txt`；`init` 仍可在该位置生成初始密钥，但不会把路径登记进项目配置。

不同开发者、开发机和生产环境可以使用不同的密钥。项目中的 `[recipients.registry]` 和文件授权集合登记公开 recipient，不要求各环境共享同一把私钥。只向每个环境提供它实际需要的身份，私钥不要提交到版本控制，也不要输出到 CI 日志。

## Infisical 参考脚本

私钥托管在 Infisical 时，可以独立使用 [Infisical CLI](https://infisical.com/docs/cli/commands/secrets) 导出一个 secret 的完整值。先自行安装 CLI 并完成登录或机器身份认证；认证、访问权限和 secret 内容均由 Infisical 管理，YewSeal 不检查 `.infisical.json`，也不调用 Infisical CLI。

下面是 POSIX shell 参考脚本，不是 YewSeal 的内置功能。将 project ID、环境、路径和 secret 名称替换为当前机器对应的值；secret 内容应为完整的 Age 私钥文件，可以包含多把身份，而不是整个项目的环境变量导出。

```sh
#!/bin/sh
set -eu
umask 077

mkdir -p .age
tmp=$(mktemp .age/keys.txt.XXXXXX)
trap 'rm -f "$tmp"' EXIT
trap 'exit 1' HUP INT TERM

infisical secrets get AGE_KEY_FILE --plain --silent \
  --projectId 'your-project-id' \
  --env 'dev' \
  --path '/yewseal' > "$tmp"
test -s "$tmp"
mv -f "$tmp" .age/keys.txt

yews --key-file .age/keys.txt decrypt
```

脚本只在导出成功且内容非空后替换目标文件，避免远端命令失败时清空已有私钥；临时文件和替换后的文件权限为 `0600`。它不会验证私钥格式，格式由 YewSeal 读取时校验。请在自己控制的工作目录运行，并将 `.age/` 加入 `.gitignore`。已有私钥会被替换，解密失败不会回滚此次替换。

CI 中使用的 `INFISICAL_TOKEN` 或其他认证方式按 [Infisical 官方文档](https://infisical.com/docs/cli/overview) 配置。Docker 场景可在宿主机先导出私钥，再把文件只读挂载到容器，并通过 `--key-file` 指定容器内路径。

## 其他来源

密码管理器、云 secret manager、CI secret 和本地文件都适用相同的职责划分：外部工具提供身份，YewSeal 使用身份解密。不需要为每一种来源增加 Provider 或项目配置。
