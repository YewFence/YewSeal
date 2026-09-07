# diff - 比较文件差异

`diff` 用于开发时预览本地明文与历史密文解出内容的差异。展示差异本身不算失败；宽松模式允许没有任何实际比较的空预览，未比较的原因会写入 stderr。

## 语法

```bash
yews diff [command options] [target]
```

## 目标选择

不传 `target` 时，`diff` 按明文路径选择当前目录范围内的映射。目录 target 也只筛选该目录明文侧的已登记映射，不重新扫描；文件 target 可以匹配已发现映射的明文路径或加密路径。

Group 从明文侧发现，只有密文的 Group 条目不进入候选集合；显式 FilePair 不依赖文件存在即可登记。没有选中映射仍然报选择错误，与已选中但全部跳过不同。

选中映射的明文或密文任一侧不存在时，跳过并报告未比较，不输出新增或删除补丁，也不算内容相同。两侧存在才尝试解密；身份不匹配也可跳过。已检测到的密文损坏、权限错误及其他读写失败仍是错误，单文件错误不阻断其他文件的比较。

格式与映射来自配置；当前 recipient alias 失效时警告后继续，解密依据历史密文 metadata 和当前 Identity bundle，不以当前配置授权替代历史授权。选择成功后始终解析一次身份来源，即使所有映射最终都因缺少输入而跳过，无效的 `--key-file` 仍会失败。

```bash
yews diff config.toml
```

## 选项

### --strict

要求所有两侧输入均存在的选中映射完成比较，不要求内容一致。缺少输入的跳过不影响 strict 成功；两侧存在但身份不匹配则使 strict 失败。默认宽松模式下，即使全部跳过也可返回 `0`，但 stderr 会明确列出未比较原因和汇总数量。

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

向 stderr 额外输出文件选择信息和逐文件比较完成提示。

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

宽松模式下，只要没有真正错误就退出 `0`，无论有无差异或是否全部跳过。strict 模式下，缺输入不影响成功，但可比较映射因身份不匹配跳过时退出 `1`。真正错误、参数、配置、身份来源和输出通道错误均退出 `1`；已经获得的差异不会撤回。

宽松模式的 `0` 既不代表文件一致，也不保证实际比较过任何文件。`--strict` 只要求可比较映射完成比较，不会把内容差异视为失败，也不要求缺失文件出现。`diff` 没有“差异即失败”的开关，不用于 CI 判定。

## 输出

stdout 只包含 diff 正文，没有实际差异时为空。警告、逐文件跳过或失败的原因和最终汇总写入 stderr，不需要 `--verbose`；使用 `--verbose` 时还会显示文件选择信息和逐文件比较完成提示。

```bash
# changes.diff 仅包含 diff 正文，诊断仍显示在终端
yews diff --color never > changes.diff

# 分开保存正文与诊断
yews diff --color never > changes.diff 2> diagnostics.log
```

## 相关命令

[encrypt](/commands/encrypt) 可以重新加密明文文件，[view](/commands/view) 可以查看加密文件内容。
