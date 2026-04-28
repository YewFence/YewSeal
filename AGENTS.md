## 项目概述

YewSeal 是一个 Go CLI 工具，用于管理加密配置文件。核心功能是通过 SOPS + Age 加密，支持多种配置格式（TOML/YAML/JSON/ENV/INI）。由于 SOPS 不原生支持 TOML 格式，本工具通过 TOML ↔ YAML 转换来实现。

## 常用命令

```bash
# 构建
mise run build                    # 构建 → build/yews
mise run build -- --all           # 构建所有平台
mise run build -- --target linux/amd64  # 构建指定平台
mise run build -- --release       # 构建所有平台并打包
mise run clean                    # 清理构建产物

# 全量测试
mise run test       # 运行所有测试 (go test -v ./...)
```

## 关键设计

**配置优先级**：CLI 参数 > 环境变量 > 配置文件 > 默认值

**内嵌库依赖**：`filippo.io/age`（密钥生成）、`github.com/getsops/sops/v3`（加解密引擎）

**外部工具依赖**：仅 `toml2yaml`/`yaml2toml` (remarshal)，且仅 TOML 格式需要

**格式支持**：
- SOPS 原生格式（YAML/JSON/ENV/INI）：直接加密
- TOML：转换为 YAML 后加密（TOML → YAML → 加密）

**密钥同步**：通过 Provider 接口扩展，目前支持 Infisical
