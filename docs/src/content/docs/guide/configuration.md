---
title: Configuration
---

YewSeal uses `.yewseal.toml` to declare [file mappings](#file-mappings) and [recipient authorization](#recipient-authorization). Configure public keys and file paths here, then use the [managed files](#managed-files) with your version-control and SOPS workflows. For decryption, supply a private Age key using the [private-key sources](#reading-private-keys).

## Config loading order

YewSeal loads configuration from the root of the current Git repository down to the current directory, picking at most one config file per directory. Outside a Git repository only the current directory is searched — the loader never walks above it. Within a single directory the priority is `.yewseal/.yewseal.toml` over `.config/.yewseal.toml` over `.yewseal.toml`.

Child-directory configs override or extend parent ones: an `[[encryption.files]]` entry replaces a parent entry with the same plaintext or encrypted path, and `[[encryption.groups]]` entries accumulate.

## When config is loaded

Help, version output, bare `yews`, and static completion work without a project config, including when an existing config is invalid. Invalid flags can still fail these commands.

Other commands require a valid `.yewseal.toml`, except for `init`. Missing, malformed, or invalid configs produce an error. Argument errors such as an invalid worker count are reported before config errors.

To rebuild an invalid config, use `init --force`. This also replaces the owner key; existing ciphertext may become undecryptable. See [`yews init --help`](/references/yews_init) before using it.

## .yewseal.toml

### Editor completion and validation

With Taplo or the Even Better TOML extension for VS Code, add this comment at the top of `.yewseal.toml` for completion, hover docs, and validation:

```toml
#:schema https://raw.githubusercontent.com/YewFence/YewSeal/main/schema/yewseal.schema.json

[encryption]
```

[schema/example.yewseal.toml](https://github.com/YewFence/YewSeal/blob/main/schema/example.yewseal.toml) contains an example covering every field.

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

Every path processed at runtime must come from an explicit file pair or a group. A missing or empty config, or an unregistered target, never auto-generates a file pair; the default private key file remains `.age/keys.txt`.

Relative `plaintext` and `encrypted` paths are resolved against the directory of the config that declares them.

:::note
An absolute path is taken literally and may point anywhere on disk. That works but is discouraged: it binds the config to a single machine, and clones, CI checkouts, or a moved project directory break it — keep everything outside the project on a symlink and register the relative link instead. `~` is never expanded, and on Windows an absolute path must include a drive letter.
:::

Symlinked plaintext paths are followed transparently: `encrypt` and `decrypt` write through the complete link chain to the final regular file, and `clean` deletes that final target while leaving every link in place, so a later `decrypt` writes back to the same location. A broken chain counts as [already absent](/guide/plaintext-cleanup) for `clean` and keeps the link; unresolvable chains and non-regular targets fail rather than being treated as absent.

Files and encryption authorization are declared centrally in the project config. For one-off single-file tasks that do not need project-level management, use SOPS directly; see [Interop with SOPS](/guide/sops).

### Recipient authorization

`[recipients.registry]` maps reviewable aliases to single public Age recipients. Aliases are case-sensitive, must start with an ASCII letter, and may contain letters, digits, underscores, or hyphens; duplicate aliases, one public key under multiple aliases, and invalid keys are all errors. Private keys cannot live in the registry.

`recipients` on file pairs and groups accepts aliases only. The effective set is chosen as explicit file pair over matching group over `recipients.defaults`; each level fully replaces the previous one rather than merging. Recipient arrays must contain at least one alias — an empty array is rejected — and omitting the field is how you inherit from the matching group or `recipients.defaults`.

When one path matches multiple groups, their recipient key sets must be identical, otherwise a conflict is reported. An explicit file pair for the same path takes precedence. Alias order does not affect authorization.

### File mappings

`[[encryption.files]]` pairs one plaintext path with one encrypted path. This pair is called a *mapping*:

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

`[[encryption.groups]]` scans a batch of files by patterns. Its discovery root is the directory of the config that declares the group:

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

`patterns` is required and uses the gitignore dialect: `*`, `?`, `**`, and `!` exclusions; a leading `/` anchors a rule to the group's discovery root, and a trailing `/` restricts a rule to directories. Path separators are `/` on every platform including Windows; `\` retains its glob escape meaning. Encryption groups exclude [protocol files](#file-naming-conventions) and the `encrypted` paths of explicit file pairs. Decryption discovers ciphertext by those suffixes and filters it through `patterns` applied to the logical plaintext paths.

`format_rules` uses `<pattern>=<format>` entries; the first matching rule decides the format, and the values are the same as `format`. When `unknown_as_binary` is `true`, files whose format cannot be recognized during group encryption are treated as binary.

To select part of a group for an operation, use the [target-selection rules](/guide/target-selection).

## Managed files

YewSeal maintains `.gitignore` and, when enabled, `.sops.yaml` in the current working directory. Entries use relative paths and cover only that directory and its subdirectories, including when an operation explicitly selects an external file. `.gitignore` uses plaintext paths; `.sops.yaml` uses encrypted paths.

### .sops.yaml

Use `.sops.yaml` for [direct SOPS commands](/guide/sops). It contains exact-match rules for all configured ciphertext paths allowed by the managed-file rules, including ciphertext-only group results. Target selectors and temporary `--output` paths do not change these rules.

```yaml
creation_rules:
  - path_regex: ^config\.enc\.toml$
    age: age1xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

`init` creates this file by default; interactive initialization asks whether to create it. `encrypt` updates it after encryption, and `verify` checks for drift. To manage SOPS rules yourself, set `YEWSEAL_SYNC_SOPS_CONFIG=false` or use `--sync-sops-config=false` for these commands. Existing `.sops.yaml` content is then left untouched and its drift check is skipped. YewSeal encryption continues to use the authorization declared in `.yewseal.toml`.

For synchronization failures and exit codes, see [`yews encrypt --help`](/references/yews_encrypt) and [`yews init --help`](/references/yews_init).

### .gitignore

Run `init`, `encrypt`, or `decrypt` to add plaintext ignore entries and the default `.age/keys.txt` entry. Existing rules are preserved. For which mappings each command adds, see [`encrypt`](/references/yews_encrypt) and [`decrypt`](/references/yews_decrypt).

For the [Tutorial](/guide/tutorial) project, the generated entries are:

```ini
# YewSeal - Decrypted configuration files
config.toml

# YewSeal - Age private keys
.age/keys.txt
```

With [group scanning](#group-scanning), you can add format-wide rules and re-include [protocol files](#file-naming-conventions):

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

## Age key management

### Reading private keys

At decryption time, the Age private key resolves in this order (highest first):

1. The explicit global flag `--key-file` / `-k`, or `YEWSEAL_KEY_FILE`
2. A comma-separated, whitespace-separated, or multi-line bundle in `YEWSEAL_AGE_IDENTITIES`
3. The compatible alias `SOPS_AGE_KEY`
4. `SOPS_AGE_KEY_FILE`
5. `YEWSEAL_AGE_KEY_CMD`
6. `SOPS_AGE_KEY_CMD`
7. The default path `.age/keys.txt` under the current working directory

Sources never merge across levels: the first present source wins outright, even when it contains no valid identity, and everything below it is not read — with `--key-file` set, the environment variables and `.age/keys.txt` are ignored entirely. Only identities within the winning source combine into one bundle. An unset source does not participate; a missing `SOPS_AGE_KEY_FILE` also falls through to the next source.

`yews identities` prints exactly this resolution: the winning source, the present sources it shadowed, and every identity with its derived public key and registry alias. With no source it returns an empty source and identity list; a present but empty source keeps its source label. It warns when a public key is not registered, and with `--reveal` also includes the secret keys — mind terminal scrollback and CI logs when you use it.

```bash
yews --key-file ~/.age/my-key.txt decrypt config.enc.toml
```

```bash
export SOPS_AGE_KEY_FILE="$HOME/.age/my-key.txt"
yews decrypt config.enc.toml
```

Relative paths given by flag or environment variable, like the default `.age/keys.txt`, are resolved against the current directory of the invocation, not the directory containing `.yewseal.toml`; use an absolute path when running from a subdirectory.

Private-key sources provide decryption access; encryption uses the [recipient authorization](#recipient-authorization) in `.yewseal.toml`. After changing recipients, run [`encrypt`](/references/yews_encrypt) to update the ciphertext.

### Identity bundles

One key file may contain multiple Age private keys; YewSeal ignores comments and blank lines, deduplicates valid identities by first occurrence, and reports malformed items on stderr using their line and item positions plus a redacted preview. A source that parses to no valid identities is a valid empty bundle, including an empty successful key-command response. An unreadable explicit key file or a key command that exits unsuccessfully remains a calling error. CI can also pass multiple private keys through YewSeal's environment variable:

```bash
YEWSEAL_AGE_IDENTITIES='AGE-SECRET-KEY-1...,AGE-SECRET-KEY-1...' yews decrypt config.enc.toml
```

`SOPS_AGE_KEY` is a lower-priority alias with the same bundle syntax, so existing SOPS-oriented environments work unchanged.

## External private key sources

YewSeal provides no `sync`, `sync pull`, or `[sync]` configuration. Private keys are supplied by developers or deployment environments; reference scripts for external tools such as Infisical live in [External private key sources](/guide/private-keys).

## Environment variables

YewSeal flags have environment variables unless their help explicitly marks them CLI-only. Global flags use `YEWSEAL_<FLAG>`; command flags use `YEWSEAL_<COMMAND>_<FLAG>`, with names uppercased and hyphens replaced by underscores. For example, `--key-file` uses `YEWSEAL_KEY_FILE`, `encrypt --parallel` uses `YEWSEAL_ENCRYPT_PARALLEL`, and `decrypt --strict` uses `YEWSEAL_DECRYPT_STRICT`. Each environment-enabled flag's `--help` entry shows its exact variable. The dangerous `clean --force` mode is CLI-only and deliberately ignores `YEWSEAL_CLEAN_FORCE`.

Explicit flags override environment variables, which override flag defaults. Empty environment values are treated as unset; invalid non-empty booleans and integers fail before project configuration is loaded. Environment variables for other commands are ignored during the current invocation.

Non-flag integration variables remain owned by their respective identity or editor modules:

| Variable | Purpose |
| --- | --- |
| `YEWSEAL_AGE_IDENTITIES` | Preferred inline Age identity bundle |
| `SOPS_AGE_KEY` | SOPS-compatible alias for `YEWSEAL_AGE_IDENTITIES` |
| `SOPS_AGE_KEY_FILE` | Path to an Age private key file |
| `YEWSEAL_AGE_KEY_CMD` | Preferred command whose output provides an Age identity bundle |
| `SOPS_AGE_KEY_CMD` | Command whose output provides an Age identity bundle |
| `EDITOR` | Editor used by `edit` when `VISUAL` is unset |
| `VISUAL` | Editor preferred by `edit` |

## Best practices

### Key safety

Never commit private key files. Different developers and environments may hold independent identities; register their public recipients in the project registry and set authorization per file. Leave private key storage and distribution to each environment, and re-run `encrypt` after changing recipient configuration to sync `.sops.yaml` and the encrypted files.

### File naming conventions

Use a protocol-file suffix for encrypted group files: `.enc.toml`, `.enc.yaml`, `.enc.json`, `.enc.env`, `.enc.ini`, or `.enc.bin`, according to the configured format. For example, `config.toml` becomes `config.enc.toml` and `.env` becomes `.env.enc.env`. Group scanning recognizes these names as ciphertext.
