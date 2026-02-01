# YewSeal TODO

> 代码审查生成于 2026-01-31

## 🔴 高优先级

### 1. 修复默认密钥路径不一致 (已完成)
- **位置**: `internal/config/config.go:160`
- **问题**: `GetKeyFile()` 返回 `.age/key.txt`，但 `keys.go:61` 使用 `.age/keys.txt`
- **修复**: 统一改为 `.age/keys.txt` (已修复)

### 2. 修复临时文件权限过宽 (已完成)
- **位置**: `internal/crypto/operations.go:67`
- **问题**: 临时文件使用 `0644` 权限（world-readable），多用户系统存在泄露风险
- **修复**: 改为 `0600` (已修复)

### 3. 统一 FilePair 类型定义 (已完成)
- **位置**: `internal/config/config.go:30-35` 和 `internal/crypto/batch.go:26-29`
- **问题**: 两处定义了相同的 `FilePair` 结构体，导致类型转换和维护问题
- **修复**: 提取到公共位置或使用类型别名 (已修复，统一使用 `config.FilePair`)

## 🟡 中优先级

### 4. 重构 SOPS 命令构建逻辑(done)
- **位置**: `internal/crypto/operations.go`
- **问题**: `encryptTOML()` 和 `encryptNative()` 有大量重复的 SOPS 参数构建代码
- **修复**: 提取 `buildSopsEncryptArgs()` 辅助函数

### 5. 为 sync 模块添加测试
- **位置**: `internal/sync/`
- **问题**: 该模块无测试覆盖
- **修复**: 添加 `infisical_test.go` 和 `provider_test.go`

### 6. 构建时版本注入 (已完成)
- **位置**: `cmd/main.go:23`
- **问题**: 版本号 `1.0.0` 硬编码
- **修复**: 使用 `ldflags` 在构建时注入，更新 Makefile (已修复)

## 🟢 低优先级

### 7. 优化配置加载性能
- **位置**: `internal/crypto/keys.go:92`
- **问题**: `GetPublicKey()` 每次调用都重新加载配置文件，批量操作时有性能损耗
- **修复**: 在 `BatchOptions` 中传递已加载的配置

### 8. 添加结构化错误类型(done)
- **位置**: 全局
- **问题**: 错误使用 `fmt.Errorf` 字符串拼接，难以程序化处理
- **修复**: 定义自定义错误类型如 `KeyNotFoundError`、`EncryptionError` 等
