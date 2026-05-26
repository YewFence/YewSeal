# Protection Plan

目标是先解决误运行 `yews decrypt` 导致本地明文改动被覆盖的问题，并补上查看与审查能力，让用户能判断当前明文和密文哪边更新。这个计划只覆盖 `view`、`diff`、`decrypt --force`、`decrypt --backup` 和 Git textconv 配置，不处理 binary、exec 或 SOPS CLI 透传。

## Scope
- In: `decrypt` 默认拒绝覆盖不同内容的明文文件，新增只读 `view` 命令，新增当前明文对密文解密结果的 `diff` 命令，提供 Git textconv 配置入口，让密文历史可以按明文展示差异。
- Out: 不实现 Git 历史管理，不自动写入全局 Git 配置，不缓存解密后的 textconv 结果，不实现任务编排、exec 模式或 binary 格式。

## Action items
[ ] Add an internal read-only decrypt helper that returns plaintext bytes without writing files, and reuse it from `decrypt`, `view`, `diff`, and future Git textconv support.

[ ] Change `yews decrypt` so it decrypts into memory first, then refuses to overwrite an existing plaintext file when the current file content differs from the decrypted content.

[ ] Add `--force` to `yews decrypt` to explicitly allow overwriting a differing plaintext file, keeping non-interactive behavior suitable for scripts and continuous integration.

[ ] Add `--backup` to `yews decrypt` to save the existing plaintext file before overwrite, using a timestamped backup path beside the original file.

[ ] Add `yews view <target>` that resolves a configured plaintext or encrypted target, decrypts the matching encrypted file, converts TOML outputs back to TOML when applicable, and writes only to standard output.

[ ] Add `yews diff <target>` that compares the current plaintext file with the decrypted encrypted file, prints a unified diff, exits `0` when clean, exits `1` when different, and exits `2` on errors.

[ ] Make `diff` target resolution match normal config behavior, so `yews diff wrangler.toml`, `yews diff wrangler.enc.toml.yaml`, and `yews diff` for configured files all work predictably.

[ ] Keep Git textconv support read-only by documenting `.gitattributes` entries that map encrypted files to a YewSeal diff driver, and a Git config snippet that runs `yews view` as the `textconv` command.

[ ] Add a helper command or documentation snippet for repository-local Git setup, preferring local `.git/config` instructions over automatic global configuration.

[ ] Add tests for protected decrypt refusal, `--force`, `--backup`, `view` stdout behavior, `diff` exit codes, missing plaintext behavior, identical-file behavior, and TOML conversion in view or diff.

## Proposed command behavior

`yews decrypt` should become protective by default. If the target plaintext file does not exist, it writes normally. If the file exists and already matches the decrypted content, it succeeds without rewriting. If the file exists and differs, it fails with guidance to run `yews diff`, `yews decrypt --backup`, or `yews decrypt --force`.

```bash
yews decrypt
yews decrypt --force
yews decrypt --backup
```

`yews view` should never write plaintext to disk. It is the command used by humans and by Git textconv to render an encrypted file as plaintext.

```bash
yews view wrangler.toml
yews view wrangler.enc.toml.yaml
```

`yews diff` should compare the working plaintext file against the decrypted encrypted file, not against Git history.

```bash
yews diff
yews diff wrangler.toml
yews diff wrangler.enc.toml.yaml
```

## Git textconv

Repository attributes can opt encrypted files into a YewSeal diff driver.

```gitattributes
*.enc.yaml diff=yewseal
*.enc.yml diff=yewseal
*.enc.json diff=yewseal
*.enc.toml.yaml diff=yewseal
```

The repository-local Git config can point the driver at `yews view`.

```ini
[diff "yewseal"]
    textconv = yews view
```

This makes commands such as `git diff -- wrangler.enc.toml.yaml` and `git log -p -- wrangler.enc.toml.yaml` display decrypted plaintext diffs when the local machine has the required age key. Do not enable `cachetextconv` for this driver, because caching decrypted text would undermine the protection model.

## Open questions
- Should `yews view` accept only one target at a time, or should it support all configured files with separators for human use while keeping textconv single-file only?
- Should `yews diff` normalize line endings before comparison, or should it start as byte-exact text comparison to avoid hiding changes?
- Should `decrypt --backup` imply overwrite, or should it still require `--force` for maximum explicitness?
