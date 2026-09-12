---
title: Configuration
---

YewSeal uses `.yewseal.toml` to manage file mappings and per-file authorization, and keeps `.sops.yaml` in sync so the plain SOPS CLI can run with fully managed encryption rules. Decryption identities come from command-line flags, environment variables, or `.age/keys.txt` in the current directory; the private key location is not part of the project config, and remote storage or distribution of private keys is out of scope for YewSeal.

## Config loading order

YewSeal loads configuration from the root of the current Git repository down to the current directory, picking at most one config file per directory. Outside a Git repository only the current directory is searched — the loader never walks above it. Within a single directory the priority is `.yewseal/.yewseal.toml` over `.config/.yewseal.toml` over `.yewseal.toml`.

Child-directory configs override or extend parent ones: `[[encryption.files]]` entries replace their counterparts after deduplication by plaintext or encrypted path, and `[[encryption.groups]]` entries accumulate. Private key paths never participate in config inheritance or merging.

## When config is loaded

Version output, bare `yews`, help at any level, completion script generation, and the current static tab completion never discover, read, or validate project config. Even a config with syntax errors does not make these entry points fail or emit config warnings.

`init` does not parse an existing config; it performs its own existence checks and overwrite confirmation. `init --force` can rebuild a broken config, but it also rebuilds the keys, and existing ciphertext may become undecryptable.

`encrypt`, `decrypt`, `plan`, `edit`, `view`, and `diff` require a project config. The execution order is: parse flags, handle help or version, validate config-independent arguments, load the config exactly once, select targets and validate business conditions, and only then resolve identities for operations that need decryption.

- No config file: loading fails with `no YewSeal configuration found`; execution never continues with an empty config.
- Config exists but parsing, merging, or validation fails: the config error is reported; there is no fallback to defaults.
- Config is valid but empty: this is not treated as a missing file; file selection later reports that no configured files are eligible.
- Invalid color, worker count, argument count, and other self-contained option errors take precedence over config errors; `--parallel` must be at least `1`.

Help and version do not bypass flag parsing. `yews init --help --format invalid` shows help, but `yews decrypt --help --parallel nope` and `yews --version --unknown-option` fail with argument errors; none of these three load project config. Business commands have no `--format` flag; passing one is an unknown-option error even alongside `--help`.

## .yewseal.toml

### Editor completion and validation

YewSeal maintains a JSON Schema for `.yewseal.toml`, generated from `schema/config.cue` in the repository and kept in sync by CI. With Taplo or the Even Better TOML extension for VS Code, add a comment at the top of the config to get completion, hover docs, and instant validation:

```toml
#:schema https://raw.githubusercontent.com/YewFence/YewSeal/main/schema/yewseal.schema.json

[encryption]
```

[schema/example.yewseal.toml](https://github.com/YewFence/YewSeal/blob/main/schema/example.yewseal.toml) in the repository is a complete example covering every field, also kept fresh by CI.

### Basic structure

```toml
[recipients]
defaults = ["owner"]

[recipients.registry]
owner = "age1xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"

[[encryption.files]]
plaintext = "config.toml"
encrypted = "config.enc.toml"
```

Every path processed at runtime must come from an explicit file pair or a group. A missing config, an empty config, or an unregistered target never auto-generates a file pair; the default private key file remains `.age/keys.txt`.

Pair paths resolve at load time: a relative `plaintext` or `encrypted` path is interpreted against the directory of the config file that declares it — the same base as a group's [discovery root](/guide/glossary#discovery-root). 

:::note
An absolute path is taken literally and may point anywhere on disk. That works but is discouraged: it binds the config to a single machine, and clones, CI checkouts, or a moved project directory break it — keep everything outside the project on a symlink and register the relative link instead. `~` is never expanded, and on Windows an absolute path must include a drive letter.
:::

Files and encryption authorization are declared centrally in the project config. For one-off single-file tasks that do not need project-level management, use SOPS directly; see [Interop with SOPS](/guide/sops).

### Recipient authorization

`[recipients.registry]` maps reviewable aliases to single public Age recipients. Aliases are case-sensitive, must start with an ASCII letter, and may contain letters, digits, underscores, or hyphens; duplicate aliases, one public key under multiple aliases, and invalid keys are all errors. Private keys cannot live in the registry.

`recipients` on file pairs and groups accepts aliases only. The effective set is chosen as explicit file pair over matching group over `recipients.defaults`; each level fully replaces the previous one rather than merging. An explicit empty array clears inheritance, but encrypt and plan then fail on the empty set.

When one path matches multiple groups, the resolved canonical recipient sets must be identical, otherwise a conflict is reported. An explicit file pair for the same path is the final arbiter. Recipients are sorted by public key before being handed to SOPS; alias order in the config never changes authorization semantics.

### File mappings

`[[encryption.files]]` declares the correspondence between one plaintext file and one encrypted file — a [mapping](/guide/glossary#mapping):

```toml
[[encryption.files]]
plaintext = "config.toml"
encrypted = "config.enc.toml"

[[encryption.files]]
plaintext = ".dev.vars"
encrypted = ".dev.enc.env"
format = "env"
```

`format` is optional and accepts `toml`, `yaml`, `json`, `env`, `ini`, and `binary` (aliases `yml`, `dotenv`, and `bin` are normalized at runtime). It suits files like `.dev.vars` whose format cannot be inferred from the extension.

### Group scanning

`[[encryption.groups]]` scans a batch of files by patterns:

```toml
[[encryption.groups]]
patterns = [
  "*.toml",
  "*.yaml",
  "secrets/**/*.json",
  "!*.enc.toml",
  "!*.enc.yaml",
  "!*.enc.json",
]
format_rules = [
  ".dev.vars=env",
  "secrets/*.conf=ini",
]
unknown_as_binary = false
```

`patterns` is required, which uses the gitignore dialect: `*`, `?`, `**`, and `!` exclusions; a leading `/` anchors a rule to the group's [discovery root](/guide/glossary#discovery-root) — never to a filesystem or repository root — and a trailing `/` restricts a rule to directories. Path separators are `/` on every platform including Windows; `\` retains its glob escape meaning. Encryption groups always exclude files in the [YewSeal protocol files](/guide/glossary#protocol-file) plus the `encrypted` paths of explicit file pairs in the config. Decryption discovers ciphertext by those same protocol suffixes and filters it through your `patterns` applied to the logical plaintext paths.

`format_rules` uses `<pattern>=<format>` entries; the first matching rule decides the format, and the values are the same as `format`. When `unknown_as_binary` is `true`, files whose format cannot be recognized during group encryption are treated as binary.

A group's discovery root is always the directory of the config that owns it. CLI arguments (files, directories, or patterns) only filter registered mappings; they never redefine the root or override group rules.

## .sops.yaml

`.sops.yaml` is SOPS's own config; YewSeal's `init` and `encrypt` keep it in sync with the current file mappings. Skipping `.sops.yaml` does not affect YewSeal's own encryption through the embedded SOPS engine, but having it makes direct `sops` usage more convenient.

```yaml
creation_rules:
  - path_regex: ^config\.enc\.toml$
    age: age1xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

YewSeal generates exact-match rules per encrypted file.

## Age key management

### Reading private keys

At decryption time, the Age private key resolves in this order (highest first):

1. The explicit global flag `--key-file` / `-k`, or `AGE_KEY_FILE`
2. A comma-separated identity bundle in `YEWSEAL_AGE_IDENTITIES`
3. A full multi-line bundle in `SOPS_AGE_KEY`
4. `SOPS_AGE_KEY_FILE`
5. `SOPS_AGE_KEY_CMD`
6. The default path `.age/keys.txt` under the current working directory

```bash
yews --key-file ~/.age/my-key.txt decrypt config.enc.toml
```

```bash
export SOPS_AGE_KEY_FILE="$HOME/.age/my-key.txt"
yews decrypt config.enc.toml
```

Relative paths given by flag or environment variable, like the default `.age/keys.txt`, are resolved against the current directory of the invocation, not the directory containing `.yewseal.toml`; use an absolute path when running from a subdirectory.

Encryption uses only the canonical Age recipients resolved from `[recipients.registry]` and the file pair, group, or `recipients.defaults` alias sets. Private key files, `SOPS_AGE_RECIPIENTS`, and `.sops.yaml` never determine YewSeal's encryption authorization.

### Identity bundles

One key file may contain multiple Age private keys; YewSeal ignores comments, blank lines, and irrelevant lines, and deduplicates by first occurrence. CI can also pass a comma-separated list of private keys through the dedicated variable:

```bash
YEWSEAL_AGE_IDENTITIES='AGE-SECRET-KEY-1...,AGE-SECRET-KEY-1...' yews decrypt config.enc.toml
```

## External private key sources

YewSeal provides no `sync`, `sync pull`, or `[sync]` configuration. Private keys are supplied by developers or deployment environments; reference scripts for external tools such as Infisical live in [External private key sources](/guide/private-keys).

## Environment variables

| Variable | Purpose |
| --- | --- |
| `AGE_KEY_FILE` | Default for the global `--key-file`, treated as an explicit key file |
| `YEWSEAL_AGE_IDENTITIES` | Comma-separated Age identity bundle |
| `SOPS_AGE_KEY` | Full multi-line Age identity bundle |
| `SOPS_AGE_KEY_FILE` | Path to an Age private key file |
| `SOPS_AGE_KEY_CMD` | Command whose output provides an Age identity bundle |
| `SOPS_OUTPUT_FILE` | `--output` value for `encrypt` and `decrypt`; plan ignores it |
| `YEWSEAL_STRICT` | Strict-mode default for `decrypt` and `diff`; an explicit `--strict` / `--strict=false` wins |
| `EDITOR` | Editor used by `edit` when `VISUAL` is unset |
| `VISUAL` | Editor preferred by `edit` |

## Best practices

### Key safety

Never commit private key files. Different developers and environments may hold independent identities; register their public recipients in the project registry and set authorization per file. Leave private key storage and distribution to each environment, and re-run `encrypt` after changing recipient configuration to sync `.sops.yaml` and the encrypted files.

### File naming conventions

The recommended naming convention:

| Plaintext format | Encrypted file |
| --- | --- |
| `config.toml` | `config.enc.toml` |
| `config.yaml` | `config.enc.yaml` |
| `config.json` | `config.enc.json` |
| `.env` | `.env.enc.env` |
| `config.ini` | `config.enc.ini` |
| `secret.bin` | `secret.enc.bin` |

### Version control

`init` (and `decrypt`) maintain `.gitignore` for you: each registered plaintext path is added with a `# YewSeal - Decrypted configuration files` header, and the default key file is excluded under `# YewSeal - Age private keys`. This is the exact content generated for the [Tutorial](/guide/tutorial) project:

```ini
# YewSeal - Decrypted configuration files
config.toml

# YewSeal - Age private keys
.age/keys.txt
```

If you prefer to ignore by pattern instead of per file — useful with [group scanning](#group-scanning), where plaintext files are discovered dynamically — extend the same structure with format-wide rules and re-include the protocol files:

```ini
# YewSeal - Decrypted configuration files
*.toml
*.yaml
*.json
*.env
*.ini
!*.enc.toml
!*.enc.yaml
!*.enc.json
!*.enc.env
!*.enc.ini
!*.enc.bin

# YewSeal - Age private keys
.age/
```

YewSeal only ever appends its own entries; hand-written rules in the same file are preserved.
