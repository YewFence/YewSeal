## 项目概述

YewSeal 是一个 Go CLI 工具，用于管理加密配置文件。核心功能是通过 SOPS + Age 加密，支持多种配置格式（TOML/YAML/JSON/ENV/INI）。TOML 由内置的原生 TOML store 直接支持（来自 `github.com/YewFence/sops/v3` fork），不再需要格式转换。

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

**内嵌库依赖**：`filippo.io/age`（密钥生成）、`github.com/YewFence/sops/v3`（加解密引擎，带原生 TOML store 的 fork）

**外部工具依赖**：无

**加密引擎边界**：`internal/sopsx` 是 engine facade，对外暴露 `Encrypt`/`Decrypt`/`Inspect`/`Rekey`/`ExtractAgeRecipients`，格式参数使用 YewSeal 命名（`toml/yaml/json/env/ini/binary`）。其余包只允许通过 facade 调用，不直接 import sops 类型。

**格式支持**：
- 所有格式（TOML/YAML/JSON/ENV/INI/binary）均由 sops store 原生加密
- TOML 加密文件协议与其他格式对齐：`wrangler.toml` → `wrangler.enc.toml`（旧版 `.enc.toml.yaml` 转换协议已废弃，遇到会报迁移提示）

**密钥同步**：通过 Provider 接口扩展，目前支持 Infisical

<!-- CODEGRAPH_START -->
## CodeGraph

This project has a CodeGraph MCP server (`codegraph_*` tools) configured. CodeGraph is a tree-sitter-parsed knowledge graph of every symbol, edge, and file. Reads are sub-millisecond and return structural information grep cannot.

### When to prefer codegraph over native search

Use codegraph for **structural** questions — what calls what, what would break, where is X defined, what is X's signature. Use native grep/read only for **literal text** queries (string contents, comments, log messages) or after you already have a specific file open.

| Question | Tool |
|---|---|
| "Where is X defined?" / "Find symbol named X" | `codegraph_search` |
| "What calls function Y?" | `codegraph_callers` |
| "What does Y call?" | `codegraph_callees` |
| "What would break if I changed Z?" | `codegraph_impact` |
| "Show me Y's signature / source / docstring" | `codegraph_node` |
| "Give me focused context for a task/area" | `codegraph_context` |
| "See several related symbols' source at once" | `codegraph_explore` |
| "What files exist under path/" | `codegraph_files` |
| "Is the index healthy?" | `codegraph_status` |

### Rules of thumb

- **Answer directly — don't delegate exploration.** For "how does X work" / architecture / trace questions, answer with 2-3 codegraph calls: `codegraph_context` first, then ONE `codegraph_explore` for the source of the symbols it surfaces. Codegraph IS the pre-built index, so spawning a separate file-reading sub-task/agent — or running a grep + read loop — repeats work codegraph already did and costs more for the same answer.
- **Trust codegraph results.** They come from a full AST parse. Do NOT re-verify them with grep — that's slower, less accurate, and wastes context.
- **Don't grep first** when looking up a symbol by name. `codegraph_search` is faster and returns kind + location + signature in one call.
- **Don't chain `codegraph_search` + `codegraph_node`** when you just want context — `codegraph_context` is one call.
- **Don't loop `codegraph_node` over many symbols** — one `codegraph_explore` call returns several symbols' source grouped in a single capped call, while each separate node/Read call re-reads the whole context and costs far more.
- **Index lag**: the file watcher debounces ~500ms behind writes; don't re-query immediately after editing a file in the same turn.

### If `.codegraph/` doesn't exist

The MCP server returns "not initialized." Ask the user: *"I notice this project doesn't have CodeGraph initialized. Want me to run `codegraph init -i` to build the index?"*
<!-- CODEGRAPH_END -->
