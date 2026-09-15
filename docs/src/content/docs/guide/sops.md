---
title: Interop with SOPS
---

YewSeal targets projects that manage the encryption and decryption of many files through project-level configuration: register plaintext/ciphertext mappings, file groups, and public recipient authorization in `.yewseal.toml`, then select the scope to process per invocation. For one-off single-file tasks that do not need project-level management, use the standalone [SOPS CLI](https://getsops.io/docs/) directly.

YewSeal itself uses the embedded engine: installing an external `sops` is never required, and YewSeal does not invoke one when config is missing. Running the examples on this page requires your own installation, though.

## How to choose

| Scenario | Recommended approach |
| --- | --- |
| Managing many files, groups, and recipient authorization through config | YewSeal |
| Touching one already-registered file of the project | `yews encrypt <path>` or `yews decrypt <path>` |
| Handling one file ad hoc without setting up project config | SOPS CLI directly |

## YewSeal's rules

Files and encryption authorization always come from `.yewseal.toml`:

- A `path` argument selects registered files; mappings or authorization are never auto-created for unregistered files.
- `--output` only names the output location for an explicit single-file target, and positional arguments (files, directories, or patterns) only select registered files; neither declares new authorization.
- The format comes from the file's `format`, the group's `format_rules`, or inference from the registered path. Business commands have no `--format` flag; only `init --format` exists, for declaring a format at creation time.
- `--key-file` supplies a decryption identity, not an encryption authorization; encryption recipients are resolved solely from the alias sets in the config.

See [Configuration](/guide/configuration) for the full rules.

## Handling a single file ad hoc

The example below uses YAML and needs neither `.yewseal.toml` nor `.sops.yaml`. Set `AGE_RECIPIENT` to your actual age public key first; the variable is a shell example, not a YewSeal setting.

```bash
export AGE_RECIPIENT='age1...'

# pass the public key explicitly and write the new ciphertext elsewhere
sops --encrypt --age "$AGE_RECIPIENT" \
  --output config.enc.yaml config.yaml

# decrypt with a local private key; recipients come from the
# ciphertext metadata
SOPS_AGE_KEY_FILE=./.age/keys.txt \
  sops --decrypt --output config.decrypted.yaml config.enc.yaml
```

Encryption needs no private key. Decryption needs no repeated `--age`, but the identities required by the ciphertext must be reachable. Never commit private keys or decrypted plaintext; also confirm output paths before running — SOPS has no counterpart to YewSeal `decrypt`'s plaintext overwrite protection.

SOPS uses its own flags, environment variables, and identity discovery rules, so the example sets `SOPS_AGE_KEY_FILE` explicitly. See [External private key sources](/guide/private-keys) for provisioning options.

## Format compatibility boundary

YewSeal ciphertext in YAML, JSON, ENV, INI, and binary uses the corresponding SOPS stores and can be handled by a SOPS CLI that supports the format. When the extension alone cannot identify the format, pass SOPS's own `--input-type` / `--output-type`; for example, ENV is named `dotenv` in SOPS.

Native TOML is the exception that needs a separate check: YewSeal uses the native TOML store of the [YewFence/sops fork](https://github.com/YewFence/sops). Handling such TOML ciphertext directly requires a SOPS CLI that includes the same store — a build of that fork, for example.

Or, handing a plain TOML file to SOPS as binary (whole-file encryption) also works.

## SOPS in an existing project

YewSeal can generate `.sops.yaml` for convenient direct SOPS usage, but the two configs keep independent responsibilities: YewSeal's encryption authorization lives in `.yewseal.toml`, and SOPS never reads its registry or aliases.

Calling SOPS directly neither registers YewSeal file mappings nor updates `.gitignore` for you. When using SOPS on project files, keep formats and recipient authorization consistent with the project policy yourself; to bring an ad hoc file under batch management later, register its mapping and authorization explicitly in `.yewseal.toml`.
