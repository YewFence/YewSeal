---
title: Target selection
---

Use positional selectors with `encrypt`, `decrypt`, `plan`, `diff`, `clean`, and `verify` to choose registered mappings. Selectors choose files for an operation; declare paths, formats, and recipients in `.yewseal.toml` using [file mappings](/guide/configuration#file-mappings) or [groups](/guide/configuration#group-scanning).

## Default selection

With no arguments, a command selects registered files under the current working directory and its subdirectories. This is the *current-directory scope*. The command determines whether a mapping's plaintext path, encrypted path, or either path must be inside it, as shown below.

## The two sides of a mapping

A mapping has a plaintext path and an encrypted path. Commands select by one or both paths. The following rules apply to default selection, directory selectors, and patterns:

| Command | Selects by | Group files eligible for selection |
| --- | --- | --- |
| `encrypt` | plaintext path | plaintext exists |
| `decrypt` | encrypted path | ciphertext exists |
| `plan` | either path | plaintext or ciphertext exists |
| `diff` | plaintext path | plaintext exists |
| `clean` | plaintext path | plaintext or ciphertext exists |
| `verify` | either path | plaintext or ciphertext exists |

Explicit file entries stay selectable even when neither file exists. Each command reports missing inputs according to its command contract; see the [command reference](/references/yews).

When both sides exist in a group, the existing plaintext name is used. For example, `config.yml` next to `config.enc.yaml` selects a mapping with `config.yml` as its plaintext path.

## What you can pass

### A registered file path

Either path selects the same mapping:

```bash
yews encrypt config.toml
yews encrypt config.enc.toml
```

Both commands encrypt the configured plaintext into the configured ciphertext path. A file selector does not change the operation or declare a new mapping.

### An existing directory

To decrypt the registered ciphertext under `configs`:

```bash
yews decrypt ./configs
```

The command's selection side determines which mappings are inside the directory. Groups retain their [configured discovery root and patterns](/guide/configuration#group-scanning).

### A pattern

Quote patterns to pass them to YewSeal without shell expansion:

```bash
yews plan './configs/*.toml'
```

An argument containing `*`, `?`, or `[` is a glob. It uses the gitignore dialect described in [group scanning](/guide/configuration#group-scanning), with two differences: positional patterns only include matches, and their root is the current working directory. Exclusions belong to group `patterns` in the config.

A pattern containing `/` is anchored to that root. A leading `/` also anchors a single-segment pattern: `/notes.toml` matches only the top-level file, while `notes.toml` can also match `a/b/notes.toml`.

### Multiple selectors

Multiple arguments take the union of their matches. Any selector that matches nothing is an error, even when the others match:

```bash
yews encrypt config.toml './secrets/*.yaml'
```

## Empty selections and errors

With a valid config and no eligible mapping in the current-directory scope, an unparameterized batch command succeeds with zero selected mappings. An explicit selector that matches nothing fails. Config errors follow the [configuration rules](/guide/configuration#when-config-is-loaded).

Conflicting group mappings or recipient sets produce an error unless an explicit file entry resolves the conflict; see [recipient authorization](/guide/configuration#recipient-authorization).

`decrypt --output` keeps these selection rules: selectors choose the mappings, while the output directory chooses where their [mirrored plaintext tree](/guide/plaintext-delivery) is written. A [delivery-classified mapping](/guide/configuration#plaintext-classification) still needs `--output` or explicit `--inplace` consent before writing.

A successful selection may still produce skipped files during processing. See [decryption results](/guide/decryption-results) for their meanings and exit behavior.

For `.gitignore` and `.sops.yaml` updates, see the [managed-file rules](/guide/configuration#managed-files).
