---
title: Decryption results and strict mode
---

Multiple developers and environments may hold different Age identities. `decrypt` and `diff` [skip](/guide/glossary#skip) files with no matching identity by default, but they never treat corrupted ciphertext, read/write failures, or overwrite conflicts as ordinary skips. `view` and `edit` have no lenient mode and still fail when the target cannot be decrypted; the failure rules for encryption are unchanged.

## Result classification

- Success: the decryption or comparison completed. Plaintext that is already identical, or a comparison with no differences, still counts as success; writing a file or producing a diff is not required.
- Skipped: no matching decryption identity; diff also skips mappings that are missing the plaintext or the ciphertext input. A skip is not a successful comparison and proves neither that contents are equal nor that the ciphertext is intact.
- Failure: detected ciphertext or data key anomalies, integrity check failures, read/write failures, and unauthorized plaintext overwrites. For decrypt, missing input remains a failure; diff classifies only a nonexistent file as a missing-input skip, not real faults such as permission errors.

After a single-file failure, the remaining selected files are still processed and summarized at the end. Pre-run errors — arguments, config, identity resolution, or an impossible scan — abort the run; output channel failures also fail the command.

## Strict mode

```bash
# lenient by default: process what the current identity can access
yews decrypt

# decrypt requires every selected file to complete
yews decrypt --strict

# the env var sets the default; an explicit flag wins
YEWSEAL_DECRYPT_STRICT=true yews decrypt
YEWSEAL_DECRYPT_STRICT=true yews decrypt --strict=false
```

Without `YEWSEAL_DECRYPT_STRICT` the default is lenient. The variable accepts the same boolean syntax as the flag. A non-empty value that cannot be parsed is an argument error; an empty value is treated as unset. An explicit `--strict` or `--strict=false` overrides the variable even when the variable itself is invalid.

The variable is read only during business argument validation of `decrypt`; it never affects version, help, completion, `init`, `encrypt`, `view`, `edit`, or `diff`, and it has no project config counterpart. Strict decrypt requires every selected item to complete; it never widens the selection, stops on first error, or rolls back the batch. `diff` has no strict mode at all — it is a development preview, not a completeness gate.

## Exit codes

| Situation | lenient decrypt | strict decrypt | diff |
| --- | --- | --- | --- |
| All succeeded, no differences | 0 | 0 | 0 |
| All compared, differences found | n/a | n/a | 0 |
| Partial success, rest skipped for missing identity | 0 | 1 | 0 |
| All skipped for missing identity | 1 | 1 | 0 |
| All unprocessed for missing input | 1 | 1 | 0 |
| Partial success, rest missing input | 1 | 1 | 0 |
| Any real error | 1 | 1 | 1 |

The table assumes selected mappings and successful pre-run validation. `diff` is a development preview command: finding and showing differences is not a failure, and its exit code must not be used to decide whether files are identical. Argument, config, identity source, and output channel errors return `1`. When a batch produces both differences and a real error, the diffs already obtained are still printed and the exit code is `1`. An explicitly selected single file with no matching identity still exits `0`.

## What diff compares

Groups discover mappings from the plaintext side using the config root and rules, keeping the real plaintext path — for example the correspondence between `config.yml` and `config.enc.yaml`. Group entries with only a ciphertext never become diff candidates; explicit file pairs are registered as configured and enter the selection even with missing inputs. The cwd and directory targets filter by the plaintext side; a file target may match either side of a discovered mapping; a directory target never rescans.

Selecting no mapping at all is a selection error, which differs from selecting files that all end up skipped. Once a selection succeeds, the identity bundle is resolved exactly once: an invalid explicit private key source fails even if every mapping ultimately misses its inputs.

Each mapping is input-checked first: when either the plaintext or the ciphertext side is missing, the mapping is skipped without decryption and without new/deleted patches. Only when both sides exist are decryption and comparison attempted. Missing inputs and identity mismatches are counted and reported separately. Detected ciphertext corruption and other per-file errors always fail.

diff keeps the historical-decrypt exception for stale aliases: after a warning, it still compares using the ciphertext metadata and the identity bundle, without requiring the current config recipients to match the ciphertext. Base config validation is unchanged.

The diff body goes to stdout only; when no diff is generated, stdout is empty. stderr lists every skipped or failed mapping with its reason by default, and summarizes the counts of compared, missing-input, identity-mismatched, and failed mappings; on identity mismatch or a real error it reports `Comparison incomplete`. Verbose output and alias warnings also go to stderr; no status markers are ever injected into the body.

A diff exit `0` neither means the compared files are identical nor guarantees that anything was compared at all; with all inputs missing it still succeeds. `diff` is not a CI gate for file equality or deployment input completeness.

## Files and metadata

A skip never creates, deletes, or updates the corresponding plaintext, nor creates its output directory. An existing stale plaintext is kept but does not count as processed; do not keep assuming it is up to date. `--force` only allows overwriting files that decrypt successfully and never deletes or modifies skipped files.

`decrypt` still updates `.gitignore` for the current project or target scope before processing files, including plaintext paths that may end up skipped, to prevent stale plaintext from being committed. If that update fails, writing plaintext never starts. Consequently, after an all-skipped failing run, `.gitignore` may already have changed. `diff` writes neither plaintext nor project metadata.

Files processed successfully are never rolled back because of later failures or strict incompleteness. In CI/CD, use `yews decrypt --strict` and deploy only after it exits successfully.
