## 项目概述

YewSeal 是一个 Go CLI 工具，用于管理加密配置文件。核心功能是通过 SOPS + Age 加密，支持多种配置格式（TOML/YAML/JSON/ENV/INI）。TOML 由内置的原生 TOML store 直接支持（来自 `github.com/YewFence/sops/v3` fork），不需要格式转换。

## Principles

After changing Go code, run `mise run check` before finishing, make sure the check passes. If the check fails, try to run `mise run fix` to fix the issues automatically. If the check still fails, you may need to fix the issues manually.

This project is in early development and does not require backward compatibility yet. When a cleaner long-term design requires an incompatible change, make the change deliberately instead of preserving compatibility through extra complexity.

## 关键设计

**配置优先级**：CLI 参数 > 环境变量 > 配置文件 > 默认值

**内嵌库依赖**：`filippo.io/age`（密钥生成）、`github.com/YewFence/sops/v3`（加解密引擎——作者的个人 fork，带原生 TOML store；引擎层问题应在 fork 仓库开 Issue 或修复） 

**加密引擎边界**：`internal/sopsx` 是 engine facade，对外暴露 `Encrypt`/`Decrypt`/`Inspect`/`Rekey`/`ExtractAgeRecipients`，格式参数使用 YewSeal 命名（`toml/yaml/json/env/ini/binary`）。其余包只允许通过 facade 调用，不直接 import sops 类型。

**格式支持**：TOML/YAML/JSON/ENV/INI/binary 均由 sops store 原生加密

**密钥同步**：通过 Provider 接口扩展，目前支持 Infisical
