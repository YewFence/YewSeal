# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

YewSeal 是一个 Go CLI 工具，用于管理加密配置文件。核心功能是通过 SOPS + Age 加密，支持多种配置格式（TOML/YAML/JSON/ENV/INI）。由于 SOPS 不原生支持 TOML 格式，本工具通过 TOML ↔ YAML 转换来实现。

## 常用命令

```bash
# 构建
make build          # 生产构建 → build/yews.exe
make dev            # 开发构建 → test/yews.exe
make clean          # 清理构建产物

# 测试
make test           # 运行所有测试 (go test -v ./...)
go test -v ./internal/crypto/...  # 运行单个包的测试

# 跨平台构建
make build-all      # 构建所有平台 (linux/windows/darwin × amd64/arm64)
make release        # 构建并打包为发布归档

# 运行（开发）
cd test && ./yews.exe init|encrypt|decrypt|edit|sync
```

## 架构

```
cmd/main.go              # CLI 入口 (urfave/cli/v2)
internal/
├── config/              # 配置管理
│   └── config.go        # 多路径查找 .yewseal.toml，配置优先级合并
├── crypto/              # 核心加密模块
│   ├── operations.go    # Encrypt/Decrypt/Edit 单文件操作
│   ├── batch.go         # BatchEncrypt/BatchDecrypt 批量操作（支持并行）
│   ├── format.go        # 文件格式检测 (TOML/YAML/JSON/ENV/INI)
│   ├── converter.go     # TOML ↔ YAML 格式转换 (remarshal)
│   ├── init.go          # InitProject 项目初始化
│   ├── keys.go          # Age 密钥提取
│   ├── sops_config.go   # .sops.yaml 配置生成
│   └── yewseal_config.go # .yewseal.toml 配置读写
├── sync/                # 密钥同步模块
│   ├── provider.go      # Provider 接口定义
│   └── infisical.go     # Infisical 密钥管理服务集成
└── tools/               # 工具函数
    ├── checker.go       # 外部工具检查 (age/sops/toml2yaml/yaml2toml)
    ├── executor.go      # 命令执行封装
    └── input.go         # 交互式用户输入
```

## 关键设计

**配置优先级**：CLI 参数 > 环境变量 > 配置文件 > 默认值

**外部工具依赖**：`age`、`sops`、`toml2yaml`/`yaml2toml` (remarshal)

**格式支持**：
- SOPS 原生格式（YAML/JSON/ENV/INI）：直接加密
- TOML：转换为 YAML 后加密（TOML → YAML → 加密）

**批量操作**：支持目录扫描（glob 模式）或配置文件中定义的文件对列表，可并行处理

**密钥同步**：通过 Provider 接口扩展，目前支持 Infisical

## 测试目录

`test/` 目录用于开发测试，使用 `cd test && ./yews.exe` 进行本地测试。
