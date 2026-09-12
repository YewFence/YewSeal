---
title: Working with a team
---

The [Tutorial](/guide/tutorial) ended with one owner and one key. Real repositories have several people and machines, each with their own Age identity. This page continues the same project: register a teammate, authorize them per file, and see what happens when someone cannot decrypt everything.

## The model in one paragraph

Public keys live in the repository, private keys never do. `[recipients.registry]` maps a reviewable alias to one public key; each file's `recipients` picks from those aliases. A file is encrypted to *exactly* that set — no more, no less — and everyone not in the set simply cannot read it. There is no shared team key and no key server.

## The teammate generates their own key

Nothing special is needed — any fresh Age identity works. Running `yews init` in a scratch directory is one way to get a pair:

```text
# .age/keys.txt (on the teammate's machine)
# created: 2026-09-12T14:09:50Z
# public key: age15hp2gjg20zgkqgu7p56f2u8t6j3lzqwknxdls7jwhzqsn4ra4pzsk83w07
AGE-SECRET-KEY-15D2...   (private key elided)
```

The teammate sends the owner the `age15hp2...` line — a public key, safe to paste into any chat. The private key stays on their machine.

## The owner registers the key

The owner edits `.yewseal.toml` and commits. The registry gains the teammate, the shared config gains them as a recipient, and — to show per-file authorization — a new `ops.toml` secret stays owner-only:

```toml
[encryption]
[[encryption.files]]
plaintext = 'config.toml'
encrypted = 'config.enc.toml'
format = 'toml'
recipients = ['owner', 'teammate']

[[encryption.files]]
plaintext = 'ops.toml'
encrypted = 'ops.enc.toml'
format = 'toml'
recipients = ['owner']

[recipients]
defaults = ['owner']

[recipients.registry]
owner = 'age1rczdrfjqq8h9e8hpax33j8smdcvgjzspxzhy4afp2fqx8d4hxqjq89sv49'
teammate = 'age15hp2gjg20zgkqgu7p56f2u8t6j3lzqwknxdls7jwhzqsn4ra4pzsk83w07'
```

Check what the config now resolves to before writing anything:

```bash
$ yews plan
Loaded 1 config file
Command plan
Scope .
Selected 2 file pairs

Plaintext    P.Source             Encrypted        E.Source             Format  F.Source              Aliases         Recipients                                                                                                                     Authorization             Registry Sources                            Selected By
config.toml  .yewseal.toml exact  config.enc.toml  .yewseal.toml exact  toml    .yewseal.toml format  owner,teammate  age15hp2gjg20zgkqgu7p56f2u8t6j3lzqwknxdls7jwhzqsn4ra4pzsk83w07,age1rczdrfjqq8h9e8hpax33j8smdcvgjzspxzhy4afp2fqx8d4hxqjq89sv49  .yewseal.toml recipients  owner=.yewseal.toml,teammate=.yewseal.toml  current-directory
ops.toml     .yewseal.toml exact  ops.enc.toml     .yewseal.toml exact  toml    .yewseal.toml format  owner           age1rczdrfjqq8h9e8hpax33j8smdcvgjzspxzhy4afp2fqx8d4hxqjq89sv49                                                                 .yewseal.toml recipients  owner=.yewseal.toml                         current-directory
```

Two mappings, two different authorization sets — visible at a glance, reviewable in the commit that changed them. Re-encrypt so the ciphertext actually carries the new recipient, then push:

```bash
$ yews encrypt
Summary (encrypted): 2 succeeded, 0 skipped, 0 failed (2 selected)

$ git add .yewseal.toml .sops.yaml config.enc.toml ops.enc.toml
$ git commit -m "Grant teammate access to config"
$ git push
```

`.sops.yaml` follows along automatically — its rule for `config.enc.toml` now lists both public keys.

## The teammate pulls and decrypts

On the teammate's machine, `git pull` brings in the new ciphertext. They put their own key at `.age/keys.txt` (or pass `--key-file`, or any of the [identity sources](/guide/configuration#reading-private-keys)) and run the same command as everyone else:

```bash
$ yews decrypt
SKIPPED ops.enc.toml: no matching age identity; encrypted content was not verified
Summary (decrypted): 1 succeeded, 1 skipped, 0 failed (2 selected)
```

The exit code is `0`. `config.toml` was restored; `ops.enc.toml` was skipped — the teammate is simply not a recipient of that file, and no `ops.toml` appears on their disk. This is the default **lenient** behavior: in a shared repository, people are expected to hold different identities, and "not mine" is not an error.

For a deployment, or any context that must have *every* file, demand completeness:

```bash
$ yews decrypt --strict
SKIPPED ops.enc.toml: no matching age identity; encrypted content was not verified
Summary (decrypted): 1 succeeded, 1 skipped, 0 failed (2 selected)
Error: strict mode requires complete processing: 1 of 2 files skipped
```

Strict mode turns any skip into a failure — the same gate [CI/CD integration](/guide/ci-cd) uses before deploying. The full skip/failure taxonomy, including what `diff` does differently, is in [Decryption results and strict mode](/guide/decryption-results).

:::note
A skip means "no matching identity", never "corrupted file". Detected tampering, integrity failures, and read/write errors are always failures, in lenient mode too.
:::

## Removing access and rotating keys

Remove an alias from the file's `recipients` (or delete it from the registry), then run `yews encrypt` and commit. New ciphertext no longer carries the removed recipient — that is the whole act of revocation.

Two honest caveats:

- Anyone who *already* decrypted a value still knows it. Rotating access does not rotate the secret itself; change sensitive values in the same commit when it matters.
- Git history keeps the old ciphertext, which the old identity can still read. That is a property of encrypt-then-commit, not a YewSeal limitation; re-encrypting rewrites nothing.

Do not reach for `init --force` here — it rebuilds the owner keys and the whole config from scratch. Recipient changes are ordinary config edits: edit, `encrypt`, commit.

## Private keys are still your problem (by design)

YewSeal syncs no keys anywhere. Each person obtains their private key through whatever channel your team already trusts — a password manager, Infisical, a sealed envelope. Reference scripts and the division of responsibilities are in [External private key sources](/guide/private-keys).
