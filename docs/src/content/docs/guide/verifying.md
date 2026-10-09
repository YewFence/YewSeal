---
title: Verifying repository health
---

Run `verify` before sharing configuration changes or deploying encrypted files. It checks selected mappings and reports problems without writing project files or printing plaintext or private keys. The checks, finding codes, and exit behavior are defined in [`yews verify --help`](/references/yews_verify).

Use the shared [target-selection rules](/guide/target-selection) to choose what to check. To inspect the configured mappings first, run [`plan`](/references/yews_plan). Both reports include each mapping's [plaintext classification](/guide/configuration#plaintext-classification) so delivery policies can be audited without writing plaintext.

## Choosing a decryption posture

For a local check, run:

```bash
yews verify
```

To inspect local edits before encrypting them, use `diff`, save the changes with `encrypt`, and verify again:

```bash
yews diff config.toml
yews encrypt config.toml
yews verify config.toml
```

For CI without access to private keys, use static checks:

```bash
yews verify --no-decrypt
```

For a gate that must also prove decryption and content integrity, provide an [identity](/guide/configuration#reading-private-keys):

```bash
yews verify --decrypt --key-file /run/secrets/yewseal-identities
```

To restore plaintext during deployment, follow verification with [`decrypt --strict`](/guide/decryption-results); for caller-owned temporary trees, use the [runtime delivery workflow](/guide/plaintext-delivery).

## What to do with a failing run

Read each finding's hint, then repair the affected mapping:

| Finding | Repair |
| --- | --- |
| `recipient_missing`, `recipient_extra` | Run `encrypt` after changing [recipient authorization](/guide/configuration#recipient-authorization). If only ciphertext remains, restore the plaintext with an authorized identity first. |
| `plaintext_drift` | Inspect the difference with `diff`, then encrypt local changes or restore the intended plaintext. |
| `plaintext_not_ignored` | Restore the [ignore rules](/guide/configuration#gitignore), then verify again. |
| `plaintext_tracked` | Save any local changes to ciphertext, remove the plaintext from version control, and optionally remove the local copy with [`clean`](/guide/plaintext-cleanup). Audit past commits separately for exposure. |
| `key_not_ignored`, `key_tracked` | Protect the key from version control. If it was exposed, generate a replacement, update the registry, and re-encrypt. |
| `decrypt_failed`, `mac_mismatch` | Check the identity source. For damaged ciphertext, recover plaintext from another source and re-encrypt. |
| `sops_config_drift` | Run `encrypt` to restore the [managed SOPS rules](/guide/configuration#sopsyaml). For manually maintained SOPS rules, follow the opt-out instructions there. |

For example, after rotating a registry key, verification may report:

```text
ERROR [recipient_missing] config.toml -> config.enc.toml: configured recipient owner cannot decrypt this file
  hint: run 'yews encrypt' to rewrap the data key for the configured recipients
```

Restore or retain the intended plaintext, then run:

```bash
yews encrypt config.toml
yews verify config.toml
```

## Exit codes and machine-readable output

When integrating verification into a script, use `--json` and handle the [documented exit codes](/references/yews_verify):

```bash
yews verify --no-decrypt --json > verify.json
```

Unignored plaintext and private-key files make verification fail with exit `1`. Warnings and skipped checks still allow exit `0`; see the [command contract](/references/yews_verify) for their meanings.

For CI setup, see [CI/CD integration - checking repository health](/guide/ci-cd#checking-repository-health).
