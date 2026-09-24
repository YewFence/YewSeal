---
title: Target selection
---

`encrypt`, `decrypt`, `plan`, and `diff` accept optional positional arguments, and those arguments do one thing only: they choose among the [mappings](/guide/glossary#mapping) registered in `.yewseal.toml`. Paths, formats, and authorization always come from the config — an argument never registers a new file, never redefines a group's [discovery root](/guide/glossary#discovery-root), and never declares recipients.

Each command's `--help` carries a condensed version of this page.

## The two sides of a mapping

Every [mapping](/guide/glossary#mapping) has two paths, and every command looks at one of them — or both — when interpreting your arguments:

```text
plaintext side                encrypted side

config.toml   ── encrypt ──▶  config.enc.toml
config.yml    ◀── decrypt ──  config.enc.yaml

                plan: either side
```

- `encrypt` selects from the plaintext side and writes the encrypted counterpart; ciphertext-only group mappings remain in the complete metadata policy but are not encryption tasks
- `decrypt` works from the encrypted side and writes the plaintext counterpart
- `plan` accepts either side — it unions both to report every mapping
- `diff` matches by the plaintext side

This is the key to the whole page: which side a command "looks at" decides what a file argument locates, what a directory contains, and what a pattern matches.

## What you can pass

### A registered file path

```bash
$ yews encrypt config.toml      # locate the mapping by its plaintext side
$ yews decrypt config.enc.toml  # ...or by its encrypted side
```

Either side locates the same mapping, and the command's own direction decides what happens next — `encrypt config.enc.toml` still writes the configured encrypted path.

### An existing directory

```bash
$ yews decrypt ./configs
```

Selects the registered mappings whose relevant side (see above) lies inside the directory. A directory filters what the config already registered; it never rescans the directory as a fresh source of files.

### A pattern

```bash
$ yews plan './configs/*.toml'
```

An argument containing `*`, `?`, or `[` is a glob, matched against the registered paths that matter for the command — plaintext paths for `encrypt` and `diff`, encrypted paths for `decrypt`, and either for `plan`. Patterns support `*`, `?`, and `**`; a pattern containing a `/` anywhere is anchored to the current working directory, and a leading `/` makes that anchoring explicit for single-segment patterns (`/notes.toml` matches only a top-level `notes.toml`, while `notes.toml` also matches `a/b/notes.toml`). This is the same gitignore dialect that group `patterns` use — what is CLI-side is the composition and the anchor: a positional pattern only includes (no `!` exclusions) and always anchors to the current working directory, while group `patterns` may exclude with `!` and anchor at their [discovery root](/guide/glossary#discovery-root) (see [Configuration - group scanning](/guide/configuration#group-scanning)).

`encrypt` patterns match plaintext paths only. Group discovery still excludes [protocol files](/guide/glossary#protocol-file) from the plaintext side, so a pattern can never double-encrypt.

Multiple arguments take the union of their selections. Patterns only include — exclusions belong to group `patterns` in the config — and any argument that matches nothing is an error, even when other arguments match.

## Per-command reference

| Command | No arguments selects | Directory selects by | Patterns match |
| --- | --- | --- | --- |
| `encrypt` | every file and group in the current directory [scope](/guide/glossary#scope) | plaintext side | registered plaintext paths |
| `decrypt` | registered mappings with the encrypted side in scope | encrypted side | registered encrypted paths |
| `plan` | mappings with either side in scope | either side | either side |
| `diff` | mappings by plaintext path in scope | plaintext side | registered plaintext paths |
| `clean` | mappings by plaintext path in scope | plaintext side | registered plaintext paths |

Two structural rules complete the picture:

- Groups discover files under their own [discovery root](/guide/glossary#discovery-root), by their own rules. `encrypt` discovers from the plaintext side and derives the `.enc.*` protocol paths; its metadata policy additionally includes the bidirectional group result so ciphertext-only mappings remain represented in `.sops.yaml`; `decrypt` reads the ciphertext-side results; `plan` unions both sides, so a new plaintext-only file and a deployment-only ciphertext both show up — and when both exist, the real plaintext path is kept (`config.yml` next to `config.enc.yaml`); `clean` unions both sides the same way but filters by the logical plaintext path, so a mapping whose plaintext was already removed is still discovered through its ciphertext and reported as [already absent](/guide/plaintext-cleanup); `diff` discovers from the plaintext side, so a group entry with only a ciphertext never becomes a candidate.
- Explicit `[[encryption.files]]` pairs stay selectable even when their files do not exist on disk — notably for `plan` and `diff`. When several groups resolve different recipient sets for the same path, an explicit pair must arbitrate, or the run reports a conflict.

## Empty selections and errors

With no config file or an empty config, loading fails. With a valid config but no eligible mapping in the current scope, an unparameterized batch command succeeds with zero selected mappings; an explicit file, directory, or pattern that matches nothing still fails.

This differs from a successful selection whose files are all [skipped](/guide/glossary#skip) later: a skip is a decryption-time concept, not a selection error. See [Decryption results and strict mode](/guide/decryption-results).
