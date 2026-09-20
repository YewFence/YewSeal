---
title: Cleaning local plaintext
---

`clean` removes registered local plaintext files that the ciphertext can already give back. It is the closing step of a working session after `decrypt`: you delete the plaintext by hand — or you forget, and the secrets stay on disk. `clean` turns that step into one governed command that refuses to delete anything it cannot prove is recoverable.

```bash
# clean everything in the current directory scope
$ yews clean
REMOVED config.toml
RETAINED wip.toml: plaintext differs from encrypted content
Summary (cleaned): 1 removed, 0 already absent, 1 retained, 0 failed (2 selected)

# clean one mapping (either side locates it)
$ yews clean config.toml

# keep work-in-progress differences without prompting
$ yews clean --skip-different

# irreversibly remove differences that decrypt fine
$ yews clean --force
```

`clean` never encrypts, never updates recipients or data keys, and never touches `.gitignore` or `.sops.yaml`. Use `encrypt` to save local changes first; use `diff` to inspect a difference before deciding.

## What "safe to delete" means

A plaintext is only removed after the same batch proves it can be recovered:

1. The logical plaintext path resolves through its complete symlink chain to a regular file.
2. The corresponding ciphertext exists and is readable.
3. At least one Age identity in the bundle decrypts the ciphertext's data key, and SOPS integrity checks pass.
4. By default, the plaintext bytes equal the decrypted bytes exactly — no normalization, so whitespace, comments, and layout differences all count as differences.

Immediately before the actual removal, `clean` resolves the symlink chain again and re-reads the target. If the link was retargeted, the target type changed, or the content no longer matches the snapshot the decision was based on, the file is kept and the item fails. This catches ordinary editor autosaves while a prompt is waiting. The check is deliberately conservative: what it detects always fails, it never retries, and it never deletes the previously resolved target. It does not lock out an external writer racing the final re-read and removal.

After an interactive Yes for differing content, `clean` also decrypts the ciphertext again. If it no longer decrypts to the same bytes that the prompt decision was based on, the plaintext is kept and the item fails.

`--force` and an interactive Yes skip only the byte-equality requirement. Every existing plaintext must be decrypted before removal and therefore requires a usable identity. Missing or corrupted ciphertext, no usable or matching identity, non-regular targets, and I/O errors mark the item `FAILED` and retain it in every mode, so a batch can never report success while unproven plaintext lingers.

## Differences and the three strategies

| Mode | Matching content | Different content | Reads stdin |
| --- | --- | --- | --- |
| default | removed automatically | prompted per file | only when a difference exists |
| `--skip-different` | removed automatically | kept automatically | never |
| `--force` | removed automatically | removed automatically | never |

The default prompt looks like this; empty input and anything other than `y`/`yes` means No:

```text
Plaintext differs from encrypted content: wip.toml
Hint: run yews diff -- 'wip.toml' to view the diff.
Delete the local plaintext anyway? [y/N]:
```

`clean` itself never prints plaintext or diff bodies. A layout-only difference (semantically equal TOML, different bytes) still prompts — stores normalize on decryption, so run the suggested `diff` and decide. For scripts and CI, pick a strategy explicitly: `--skip-different` keeps your workspace changes, `--force` is the irreversible end-of-session sweep. The two flags are mutually exclusive, flag/env combinations included, and the conflict is rejected before the config loads.

A prompt that ends in EOF or a broken channel is not a No: the file is kept, the item fails, and the command exits non-zero. Removals that already happened are never rolled back.

## Results and exit codes

Each selected mapping ends as one of:

- `REMOVED` — deleted after full verification.
- `ALREADY ABSENT` — the plaintext is missing or its symlink chain is broken; counted in the summary and shown per file only with `--verbose`. No key is required for a batch that is entirely absent.
- `RETAINED` — a difference kept by policy or by an explicit No. This is a completed decision, not an error, and contributes to exit 0.
- `FAILED` — anything that could not be proven recoverable, including prompt EOF and pre-removal recheck failures.

| Situation | Exit code |
| --- | --- |
| All removed or already absent, retained differences allowed | 0 |
| Any real failure, undecided prompt, or output channel fault | 1 |
| No usable identity for plaintext that needs verification | 1 |
| Calling errors: arguments, config, selection, unreadable explicit key file, failed key command | 2 |

Success therefore means "the chosen policy ran to completion", not "zero plaintext remains". The summary line is the authoritative record of what is left on disk; if you require zero residue, combine a strategy flag with the summary (or check it from `--verbose` output).

Dynamic groups are discovered from both sides, like `plan`: a mapping whose plaintext was already cleaned is found again through its ciphertext and reported as already absent, so repeating `yews clean` is idempotent. Scope and pattern filtering use the logical plaintext path — the same [selection rules](/guide/target-selection) as the other commands.

## Symlinks and files outside the project

`clean` follows the symlink model of `encrypt` and `decrypt`: the config authorizes the logical plaintext path, the command resolves the full chain and deletes only the final regular file, and every link in the chain stays in place so a later `decrypt` writes back to the same target. A broken chain counts as already absent and the link is kept; loops, unresolvable links, and non-regular targets fail and are kept.

This also covers the recommended layout for files outside the repository: register a relative symlink inside the project and keep the real file elsewhere. `clean` deletes the outside target when it is verified and leaves the link for the next `decrypt`. The pre-removal recheck exists exactly because the link can change while you are being prompted.
