# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

YewSeal 是一个 Go CLI 工具，用于管理加密的 TOML 配置文件。由于 SOPS 不原生支持 TOML 格式，本工具通过 TOML ↔ YAML 转换来实现加密功能，主要面向 Cloudflare Wrangler 等使用 TOML 配置的场景。

## 常用命令

```bash
# 构建
make build          # 生产构建 → build/yews.exe（带 -ldflags "-s -w" 优化）
make dev            # 开发构建 → test/yews.exe
make clean          # 清理构建产物

# 测试
make test           # 运行所有测试 (go test -v ./...)

# 依赖管理
make tidy           # 整理依赖

# 运行（开发）
make run            # 构建并运行
cd test && ./yews.exe init|encrypt|decrypt|edit
```

## 架构

```
cmd/main.go              # CLI 入口，使用 urfave/cli/v2 框架
internal/
├── config/config.go     # 配置管理，多路径查找 .yewseal.toml
├── crypto/              # 核心加密模块
│   ├── operations.go    # 加密/解密/编辑操作主流程
│   ├── init.go          # 项目初始化（生成密钥、配置）
│   ├── keys.go          # Age 密钥管理（提取公钥/私钥）
│   ├── converter.go     # TOML/YAML 格式转换
│   ├── sops_config.go   # .sops.yaml 配置管理
│   └── yewseal_config.go # .yewseal.toml 配置管理
└── tools/               # 工具函数
    ├── checker.go       # 外部工具依赖检查 (age/sops/toml2yaml/yaml2toml)
    ├── executor.go      # 命令执行封装
    └── input.go         # 交互式用户输入
```

## 关键设计

**配置优先级**：CLI 参数 > 环境变量 > 配置文件 > 默认值

**外部工具依赖**：`age`（密钥）、`sops`（加密）、`toml2yaml`/`yaml2toml`（remarshal 格式转换）

**工作流程**：
- 加密：TOML → toml2yaml → 临时 YAML → sops encrypt → 加密 YAML
- 解密：加密 YAML → sops decrypt → 临时 YAML → yaml2toml → TOML

**密钥管理**：
- 私钥存储在 `.age/keys.txt`，自动添加到 `.gitignore`
- 公钥存储在 `.yewseal.toml` 配置文件中

## 测试目录

`test/` 目录包含开发用的测试文件和构建产物，使用 `cd test && ./yews.exe` 进行本地测试。
