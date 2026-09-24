---
title: Verifying repository health
---

Encrypting a file once does not keep a repository healthy. Recipients rotate, configs get edited, plaintext gets committed by mistake — and none of it fails until a deployment cannot decrypt. `verify` catches these states early: it re-reads the repository the way `encrypt` and `decrypt` would, and reports every inconsistency — without writing any file, and without printing plaintext, private keys, or data keys.

It complements the other read-only commands: `plan` previews what the config resolves to, `diff` compares one plaintext against its ciphertext, and `verify` checks the repository's standing state. Target selection works exactly like `plan`'s — either side of a mapping selects it — so the same [selection rules](/guide/target-selection) apply.

## What each layer catches

Each selected mapping is checked in four layers, and every layer exists because of a failure mode that stays invisible until it hurts.

**Ciphertext integrity.** The encrypted file must exist, be a regular file, parse as SOPS, and carry exactly one key group of Age recipients. SOPS keeps this metadata unencrypted, so none of it needs a private key: a deleted, moved, or truncated ciphertext fails here, before any deployment tries to open it.

**Authorization consistency.** The recipients embedded in the ciphertext must equal the set `.yewseal.toml` resolves. The usual cause of divergence is a recipient change that was never followed by `yews encrypt` — here, the registry key was rotated but the file was not re-encrypted:

```bash
$ yews verify
Error: verify completed with error findings
Summary: 5 passed, 0 warnings, 2 errors, 0 skipped
ERROR [recipient_extra] config.toml -> config.enc.toml: recipient age178zp…js7u0kue can decrypt this file but is not configured
  hint: run 'yews encrypt' to rewrap the data key for the configured recipients
ERROR [recipient_missing] config.toml -> config.enc.toml: configured recipient owner cannot decrypt this file
  hint: run 'yews encrypt' to rewrap the data key for the configured recipients
```

Only key sets are compared — renaming an alias without changing its key is not drift.

**Version-control exposure.** Selected plaintext paths and file-backed Age keys must not be tracked by the nearest git or jj repository. This layer reads the current state — the git index, or jj's parent commit — not the whole history: a `plaintext_tracked` finding can be a file that is only staged, and untracking it (`git rm --cached`) clears the finding even when earlier commits still hold the bytes. Ignoring never untracks, so "committed, then ignored" keeps failing. Verify does not scan past commits — audit history separately when exposure is suspected. A file one commit away from leaking — untracked and not ignored — is a warning.

**`.sops.yaml` drift.** The managed file is compared against the complete resolved policy, because a hand edit here would silently change what the standalone `sops` CLI may decrypt while `yews` keeps following `.yewseal.toml`. `yews encrypt` re-synchronizes it after processing; the comparison is governed by the one `--sync-sops-config` policy that `init`, `encrypt`, and `verify` share ([Configuration - .sops.yaml](/guide/configuration#sops-yaml)).

## Choosing a decryption posture

The decryption layer is the only tunable one, and its three postures match three workflows:

- **Keyless and reproducible** — `verify --no-decrypt` reads no identity source at all, so the check set no longer depends on whether the runner happens to hold a key. That is what a pull-request job wants.
- **Local and opportunistic** — the default decrypts when an identity is available, checks each file's MAC, and compares an existing local plaintext with the decrypted content by parsed structure. Without an identity the layer is skipped and the output says so. This fits a quick check before pushing.
- **Gate with a key** — `verify --decrypt` requires an identity and treats any file no identity can open as an error. It is to `verify` what `--strict` is to `decrypt` ([Decryption results and strict mode](/guide/decryption-results)): a deployment job still uses `decrypt --strict`, because it must restore files, while a verification job only proves they can be opened.

## What to do with a failing run

Every finding carries a hint; the common repairs:

- **Recipient drift** (`recipient_missing`, `recipient_extra`) — `yews encrypt` repairs this only while the plaintext exists locally: with an identity matching the old recipients it rewraps the existing data key, without one it encrypts the plaintext fresh. With ciphertext only, nothing happens — the file is skipped — so restore the plaintext first with `yews decrypt`, which requires an identity among the old recipients. Holding neither is a dead end inherent to the SOPS + Age model: the data key is wrapped for the configured recipients only, so the finding stays red until one of them decrypts or re-encrypts the file.
- **Plaintext exposure** (`plaintext_not_ignored` warning, `plaintext_tracked` error) — a warning means the ignore rule vanished; restore it (`decrypt` maintains these rules automatically). An error means the file is tracked now, possibly only staged: get any unsaved change into ciphertext first (`diff`, then `encrypt`), untrack the file, and remove the plaintext with [`clean`](/guide/plaintext-cleanup). Whether earlier commits also hold the plaintext is a separate history audit — ignoring the file is not a repair.
- **Private key exposure** (`key_not_ignored` warning, `key_tracked` error) — treat the key as leaked: rotate to a new key pair, update the registry, and re-encrypt.
- **Corrupt or undecryptable ciphertext** (`decrypt_failed`, `mac_mismatch`) — the data key no longer opens the file or its integrity check failed; recover the plaintext from another source and re-encrypt.

## Exit codes and machine-readable output

`verify` separates "the repository needs fixing" from "verify could not run": exit `1` means at least one error finding — the repository is the problem; exit `2` means the run itself failed — an invalid argument, a broken config, `--decrypt` without an identity. In CI, `2` points at the workflow, `1` at the pull request.

Finding codes are stable across releases, and `--json` prints the same report for scripts to consume:

```json
{
  "ok": false,
  "summary": {
    "pass": 5,
    "warning": 0,
    "error": 1,
    "skipped": 0
  },
  "skipped": [],
  "findings": [
    {
      "code": "plaintext_tracked",
      "severity": "error",
      "plaintext_path": "config.toml",
      "encrypted_path": "config.enc.toml",
      "message": "plaintext file config.toml is tracked by version control",
      "hint": "remove it from version control and review repository history; ignoring it now does not remove it from history"
    }
  ]
}
```

The complete list of checks, finding codes, and exit behavior is part of the command contract: [`yews verify --help`](/references/yews_verify). For wiring it into CI, see [CI/CD integration - checking repository health](/guide/ci-cd#checking-repository-health).
