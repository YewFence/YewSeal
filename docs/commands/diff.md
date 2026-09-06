# diff - 比较文件差异

`diff` 用于开发时预览明文文件与解密后内容的差异。成功展示差异是正常结果，不会因此以失败退出；退出码只表示本次预览是否成功完成。

## 语法

```bash
yews diff [command options] [target]
```

## 目标选择

不传 `target` 时，`diff` 会比较配置中当前目录范围内的文件。传入 `target` 时，可以使用明文文件路径或加密文件路径。

Group 的候选集合取明文侧与密文侧发现结果的并集。没有匹配身份的文件默认跳过，并在 stderr 逐文件提示；能解密但明文缺失，或密文缺失，仍算错误。单文件错误不会阻断其他文件的比较。

```bash
yews diff config.toml
```

## 选项

### --strict

要求所有选中文件完成比较，不要求文件内容一致。默认宽松模式下，部分文件因身份不匹配跳过、其余文件比较成功时仍返回 `0`，但 stderr 会明确标注比较不完整并汇总数量。

```bash
yews diff --strict
YEWSEAL_STRICT=true yews diff --strict=false
```

环境变量只在没有显式 flag 时生效。完整规则见[解密结果与严格模式](/guide/decryption-results)。

### --color

控制差异输出的颜色，支持 `auto`、`always` 和 `never`，默认值是 `auto`。

```bash
yews diff config.toml --color always
```

### --verbose, -v

输出详细的文件选择信息。

```bash
yews diff config.toml -v
```

## 示例

```bash
# 比较配置中的所有文件
yews diff

# 比较单个明文文件
yews diff config.toml

# 比较单个加密文件对应的明文
yews diff config.enc.toml

# 在脚本中禁用颜色
yews diff --color never
```

## 退出码

成功完成预览时退出 `0`，无论是否发现差异。发生真正错误、全部跳过或严格模式下比较不完整时退出 `1`；已经获得的差异仍会输出。参数、配置和身份来源错误也退出 `1`。

宽松模式的 `0` 既不代表文件一致，也不保证所有选中文件都完成了比较。`--strict` 只要求完整比较，不会把内容差异视为失败。`diff` 没有“差异即失败”的开关，其退出码只表示预览是否完成，不用于 CI 判定。

stdout 只包含 diff 内容，跳过提示、错误和汇总写入 stderr，不需要 `--verbose`。

## 相关命令

[encrypt](/commands/encrypt) 可以重新加密明文文件，[view](/commands/view) 可以查看加密文件内容。
