---
layout: home

hero:
  name: YewSeal
  text: 加密配置文件管理工具
  tagline: 通过 SOPS 和 Age 加密仓库中的 TOML、YAML、JSON、ENV、INI 配置文件，让敏感配置可以安全提交。
  actions:
    - theme: brand
      text: 快速开始
      link: /guide/getting-started
    - theme: alt
      text: 命令参考
      link: /commands/init

features:
  - title: 多格式配置
    details: 原生处理 TOML、YAML、JSON、ENV、INI，无需任何格式转换或外部工具。
  - title: 内嵌加密能力
    details: 使用 SOPS 加解密引擎和 Age 密钥生成能力，减少外部工具依赖。
  - title: 配置化批量处理
    details: 通过 .yewseal.toml 配置精确文件映射和分组扫描，并可用 plan 预览选择结果。
  - title: 按文件授权
    details: 通过公开 recipient 配置每个文件的授权集合，解密身份由开发者或部署环境自行提供。
---
