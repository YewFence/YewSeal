---
title: Target selection
---

`encrypt`, `decrypt`, `plan`, and `diff` share one selection model: every positional argument is a selector over the mappings registered in `.yewseal.toml`, and the config alone governs paths, formats, and authorization. Arguments never register new files, never redefine a group's discovery root, and never declare new authorization.

Each command's `--help` carries a condensed version of these rules; this page is the complete reference, including the per-command differences.

## Selector kinds

- A registered file path — plaintext or encrypted side — selects that single mapping. Either side locates it; the command's own direction decides what happens next.
- An existing directory selects the mappings whose relevant side lies inside it. Directories filter registered mappings; groups are never rescanned with the target directory as a new root.
- An argument containing metacharacters such as `*` or `?` is a pattern, matched against the registered paths that matter for the command (see the table below).

Multiple arguments take the union of their selections. Patterns only include — exclusions come from group `patterns` in the config — and any argument that matches nothing is an error, even when other arguments match.

## Per-command scope

| Command | No-argument scope | Directory selects by | Patterns match |
| --- | --- | --- | --- |
| `encrypt` | every file and group in the current directory scope | plaintext side | registered plaintext paths |
| `decrypt` | registered files and group results with the ciphertext side in scope | encrypted side | registered encrypted paths |
| `plan` | mappings with either side in scope | either side | either side of registered mappings |
| `diff` | mappings by plaintext path in scope | plaintext side | either side of discovered mappings |

Patterns naturally cannot double-encrypt: `encrypt` patterns match plaintext paths only, and ciphertext files are excluded from group discovery anyway.

## Pattern syntax

Selection patterns support `*`, `?`, and `**`, and a leading `/` anchors the pattern to the current working directory; patterns are otherwise relative to it. This is the same glob dialect used for CLI-side selection; group scanning in the config uses the gitignore dialect instead (see [Configuration - group scanning](/guide/configuration#group-scanning)).

```bash
# registered .toml plaintext under ./configs
yews encrypt './configs/*.toml'

# registered ciphertext under ./configs
yews decrypt './configs/*.enc.toml'

# filter registered mappings for a plan report
yews plan './configs/*.toml'
```

## Groups

Groups always discover files by the directory of the config that owns them and by their own rules:

- `encrypt` discovers from the plaintext side; the format protocol generates the `.enc.*` ciphertext paths.
- `decrypt` works from the ciphertext-side discovery results.
- `plan` uses the union of both sides, so files present on only one side still appear — a new plaintext-only file and a deployment-only ciphertext both show up; when both sides exist, the real discovered plaintext path is kept (for example `config.yml` next to `config.enc.yaml`).
- `diff` discovers from the plaintext side; group entries with only a ciphertext never enter the candidate set.

Explicit `[[encryption.files]]` entries do not require the files to exist on disk to be selectable (notably for `plan` and `diff`). When several groups resolve different canonical recipient sets for the same path, an explicit file pair must arbitrate, or the run reports a conflict.

## Empty selections and errors

With no config file, an empty config, or no eligible mapping in the current scope, selection fails: there is no implicit fallback to "everything in the directory". The same applies when every argument fails to match. Distinguish this from a successful selection whose files are all skipped later — skips are a decryption-time concept, not a selection error (see [Decryption results and strict mode](/guide/decryption-results)).
