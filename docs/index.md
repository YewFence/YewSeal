---
layout: home

hero:
  name: YewSeal
  text: 加密配置文件管理工具
  tagline: 通过 SOPS 和 Age 管理 TOML、YAML、JSON、ENV、INI 配置文件。
  actions:
    - theme: brand
      text: 快速开始
      link: /guide/getting-started
    - theme: alt
      text: 命令参考
      link: /commands/init

features:
  - title: 多格式配置
    details: 原生处理 YAML、JSON、ENV、INI，并通过 TOML 与 YAML 转换支持 TOML 文件。
  - title: 内嵌加密能力
    details: 使用 SOPS 加解密引擎和 Age 密钥生成能力，减少外部工具依赖。
  - title: 密钥同步
    details: 通过 Provider 接口扩展密钥同步能力，目前支持 Infisical。
---
