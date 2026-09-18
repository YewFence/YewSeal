## 项目概述

YewSeal 是一个 Go CLI 工具，用于管理加密配置文件。核心功能是通过 SOPS + Age 加密，支持多种配置格式（TOML/YAML/JSON/ENV/INI）。TOML 由内置的原生 TOML store 直接支持（来自 `github.com/YewFence/sops/v3` fork），不需要格式转换。

## Principles

After changing Go code, run `mise run check` before finishing, make sure the check passes. If the check fails, try to run `mise run fix` to fix the issues automatically. If the check still fails, you may need to fix the issues manually.

This project is in early development and does not require backward compatibility yet. When a cleaner long-term design requires an incompatible change, make the change deliberately instead of preserving compatibility through extra complexity.

## 关键设计

**配置优先级**：CLI 参数 > 环境变量 > 配置文件 > 默认值

**配置加载时机**：命令构建不读配置；版本、帮助、静态补全和 `init` 不加载旧配置。业务命令先做不依赖配置的参数校验，再由 CLI 统一加载一次配置并传给应用层；找不到配置文件即失败，不返回空配置兜底。

**配置 Schema**：`.yewseal.toml` 的权威 schema 是 `schema/config.cue`（CUE），`schema/yewseal.schema.json` 由它导出（不要手改，用 `mise run schema:export` 重新导出）。修改 `internal/config` 的配置 struct 时，必须同步更新 `schema/config.cue` 和全字段锚点 `schema/example.yewseal.toml` 并重新导出；`internal/config/schema_sync_test.go` 的 tripwire 测试和 `mise run schema:check`（已含在 `mise run check`）会强制这一约定。

**CLI 帮助是命令契约的唯一来源**：每个业务命令的 `--help`（Cobra 的 `Long`、`Example` 和 flag usage）必须自带完整契约——用途、目标选择、每个 flag、退出码、输出通道和可运行示例，并以文档站 deep-link 收尾。改命令语义就在同一提交改 help，再跑 `mise run cli:docs` 重新生成 `docs/src/content/docs/references/`（生成物不手改；`cmd/gendocs/main_test.go` 的 tripwire 会比对生成物与已提交页面）。不再写按 flag 罗列的手写命令参考页（`docs/commands/` 已删除）：命令出现在 guide 里必须处于工作流或概念语境，概念页可以比 help 更详细，help 用浓缩版加链接回指它。文档站 URL 集中在 `internal/cli/doclinks.go`，`internal/cli/help_tripwire_test.go` 强制每个业务命令的 `Long`、`Example` 和 deep-link 非空。guide 与 help 或 `docs/design/` 冲突时以后者为准，改 guide。

**内嵌库依赖**：`filippo.io/age`（密钥生成）、`github.com/YewFence/sops/v3`（加解密引擎——作者的个人 fork，带原生 TOML store；引擎层问题应在 fork 仓库开 Issue 或修复） 

**加密引擎边界**：`internal/sopsx` 是 engine facade，对外暴露 `Encrypt`/`Decrypt`/`Inspect`/`Rekey`/`ExtractAgeRecipients`，格式参数使用 YewSeal 命名（`toml/yaml/json/env/ini/binary`）。其余包只允许通过 facade 调用，不直接 import sops 类型。

**格式支持**：TOML/YAML/JSON/ENV/INI/binary 均由 sops store 原生加密

**私钥职责**：不提供密钥同步命令或 Provider 集成；私钥的远端存储与分发由开发者或部署环境自行管理，外部工具的参考脚本见 `docs/guide/private-keys.md`。
