# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

YewSeal 是一个 Go CLI 工具，用于管理加密配置文件。核心功能是通过 SOPS + Age 加密，支持多种配置格式（TOML/YAML/JSON/ENV/INI）。由于 SOPS 不原生支持 TOML 格式，本工具通过 TOML ↔ YAML 转换来实现。

## 常用命令

```bash
# 构建
just build          # 构建 → build/yews.exe
just clean          # 清理构建产物

# 测试
just test           # 运行所有测试 (go test -v ./...)
go test -v ./internal/crypto/...  # 运行单个包的测试

# 跨平台构建
just build-all      # 构建所有平台 (linux/windows/darwin × amd64/arm64)
just release        # 构建并打包为发布归档
```

## 架构

```
cmd/yews/main.go         # CLI 入口 (urfave/cli/v2)
internal/
├── config/              # 配置管理
│   └── config.go        # 多路径查找 .yewseal.toml，配置优先级合并
├── crypto/              # 核心加密模块
│   ├── operations.go    # Encrypt/Decrypt/Edit 单文件操作
│   ├── sops_engine.go   # SOPS library 封装层（sopsEncryptData/sopsDecryptData）
│   ├── batch.go         # BatchEncrypt/BatchDecrypt 批量操作（支持并行）
│   ├── format.go        # 文件格式检测 (TOML/YAML/JSON/ENV/INI)
│   ├── converter.go     # TOML ↔ YAML 格式转换 (remarshal)
│   ├── init.go          # InitProject 项目初始化（使用 filippo.io/age 生成密钥）
│   ├── keys.go          # Age 密钥提取
│   ├── sops_config.go   # .sops.yaml 配置生成
│   └── yewseal_config.go # .yewseal.toml 配置读写
├── sync/                # 密钥同步模块
│   ├── provider.go      # Provider 接口定义
│   └── infisical.go     # Infisical 密钥管理服务集成
└── tools/               # 工具函数
    ├── checker.go       # 工具检查（内嵌库版本 + remarshal 可选工具）
    ├── executor.go      # 命令执行封装（仅保留 ExecCommand）
    └── input.go         # 交互式用户输入
```

## 关键设计

**配置优先级**：CLI 参数 > 环境变量 > 配置文件 > 默认值

**内嵌库依赖**：`filippo.io/age`（密钥生成）、`github.com/getsops/sops/v3`（加解密引擎）

**外部工具依赖**：仅 `toml2yaml`/`yaml2toml` (remarshal)，且仅 TOML 格式需要

**格式支持**：
- SOPS 原生格式（YAML/JSON/ENV/INI）：直接加密
- TOML：转换为 YAML 后加密（TOML → YAML → 加密）

**批量操作**：支持目录扫描（glob 模式）或配置文件中定义的文件对列表，可并行处理

**密钥同步**：通过 Provider 接口扩展，目前支持 Infisical

<!-- gitnexus:start -->
# GitNexus — Code Intelligence

This project is indexed by GitNexus as **yew-seal** (455 symbols, 1215 relationships, 38 execution flows). Use the GitNexus MCP tools to understand code, assess impact, and navigate safely.

> If any GitNexus tool warns the index is stale, run `npx gitnexus analyze` in terminal first.

## Always Do

- **MUST run impact analysis before editing any symbol.** Before modifying a function, class, or method, run `gitnexus_impact({target: "symbolName", direction: "upstream"})` and report the blast radius (direct callers, affected processes, risk level) to the user.
- **MUST run `gitnexus_detect_changes()` before committing** to verify your changes only affect expected symbols and execution flows.
- **MUST warn the user** if impact analysis returns HIGH or CRITICAL risk before proceeding with edits.
- When exploring unfamiliar code, use `gitnexus_query({query: "concept"})` to find execution flows instead of grepping. It returns process-grouped results ranked by relevance.
- When you need full context on a specific symbol — callers, callees, which execution flows it participates in — use `gitnexus_context({name: "symbolName"})`.

## When Debugging

1. `gitnexus_query({query: "<error or symptom>"})` — find execution flows related to the issue
2. `gitnexus_context({name: "<suspect function>"})` — see all callers, callees, and process participation
3. `READ gitnexus://repo/yew-seal/process/{processName}` — trace the full execution flow step by step
4. For regressions: `gitnexus_detect_changes({scope: "compare", base_ref: "main"})` — see what your branch changed

## When Refactoring

- **Renaming**: MUST use `gitnexus_rename({symbol_name: "old", new_name: "new", dry_run: true})` first. Review the preview — graph edits are safe, text_search edits need manual review. Then run with `dry_run: false`.
- **Extracting/Splitting**: MUST run `gitnexus_context({name: "target"})` to see all incoming/outgoing refs, then `gitnexus_impact({target: "target", direction: "upstream"})` to find all external callers before moving code.
- After any refactor: run `gitnexus_detect_changes({scope: "all"})` to verify only expected files changed.

## Never Do

- NEVER edit a function, class, or method without first running `gitnexus_impact` on it.
- NEVER ignore HIGH or CRITICAL risk warnings from impact analysis.
- NEVER rename symbols with find-and-replace — use `gitnexus_rename` which understands the call graph.
- NEVER commit changes without running `gitnexus_detect_changes()` to check affected scope.

## Tools Quick Reference

| Tool | When to use | Command |
|------|-------------|---------|
| `query` | Find code by concept | `gitnexus_query({query: "auth validation"})` |
| `context` | 360-degree view of one symbol | `gitnexus_context({name: "validateUser"})` |
| `impact` | Blast radius before editing | `gitnexus_impact({target: "X", direction: "upstream"})` |
| `detect_changes` | Pre-commit scope check | `gitnexus_detect_changes({scope: "staged"})` |
| `rename` | Safe multi-file rename | `gitnexus_rename({symbol_name: "old", new_name: "new", dry_run: true})` |
| `cypher` | Custom graph queries | `gitnexus_cypher({query: "MATCH ..."})` |

## Impact Risk Levels

| Depth | Meaning | Action |
|-------|---------|--------|
| d=1 | WILL BREAK — direct callers/importers | MUST update these |
| d=2 | LIKELY AFFECTED — indirect deps | Should test |
| d=3 | MAY NEED TESTING — transitive | Test if critical path |

## Resources

| Resource | Use for |
|----------|---------|
| `gitnexus://repo/yew-seal/context` | Codebase overview, check index freshness |
| `gitnexus://repo/yew-seal/clusters` | All functional areas |
| `gitnexus://repo/yew-seal/processes` | All execution flows |
| `gitnexus://repo/yew-seal/process/{name}` | Step-by-step execution trace |

## Self-Check Before Finishing

Before completing any code modification task, verify:
1. `gitnexus_impact` was run for all modified symbols
2. No HIGH/CRITICAL risk warnings were ignored
3. `gitnexus_detect_changes()` confirms changes match expected scope
4. All d=1 (WILL BREAK) dependents were updated

## Keeping the Index Fresh

After committing code changes, the GitNexus index becomes stale. Re-run analyze to update it:

```bash
npx gitnexus analyze
```

If the index previously included embeddings, preserve them by adding `--embeddings`:

```bash
npx gitnexus analyze --embeddings
```

To check whether embeddings exist, inspect `.gitnexus/meta.json` — the `stats.embeddings` field shows the count (0 means no embeddings). **Running analyze without `--embeddings` will delete any previously generated embeddings.**

> Claude Code users: A PostToolUse hook handles this automatically after `git commit` and `git merge`.

## CLI

| Task | Read this skill file |
|------|---------------------|
| Understand architecture / "How does X work?" | `.claude/skills/gitnexus/gitnexus-exploring/SKILL.md` |
| Blast radius / "What breaks if I change X?" | `.claude/skills/gitnexus/gitnexus-impact-analysis/SKILL.md` |
| Trace bugs / "Why is X failing?" | `.claude/skills/gitnexus/gitnexus-debugging/SKILL.md` |
| Rename / extract / split / refactor | `.claude/skills/gitnexus/gitnexus-refactoring/SKILL.md` |
| Tools, resources, schema reference | `.claude/skills/gitnexus/gitnexus-guide/SKILL.md` |
| Index, status, clean, wiki CLI commands | `.claude/skills/gitnexus/gitnexus-cli/SKILL.md` |

<!-- gitnexus:end -->
