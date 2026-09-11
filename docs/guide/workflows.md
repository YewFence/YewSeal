# Workflows

The commands are small and composable; this page shows how they fit together in daily use. Each command's `--help` documents its full semantics, and the generated [CLI reference](/references/yews) mirrors the same content on the site.

## The daily edit loop

```bash
# edit an encrypted file in place (decrypt → editor → re-encrypt)
yews edit -f config.enc.toml

# or edit the plaintext and re-encrypt it
$EDITOR config.toml
yews encrypt config.toml

# review what changed before committing
yews diff config.toml
git add config.enc.toml && git commit
```

`edit` never touches your plaintext tree; editing the plaintext and running `encrypt` produces the same ciphertext with the configured mapping and authorization. `diff` shows the pending plaintext-vs-ciphertext changes; showing differences is never an error.

## Auditing the configuration

```bash
# what is registered, in which format, authorized for whom
yews plan

# machine-readable report (errors stay on stderr)
yews plan --json > plan.json
```

Run `plan` after editing `.yewseal.toml` — adding aliases, files, or groups — to catch unknown aliases, empty authorization sets, and group conflicts before any encryption run. It checks every mapping resolved from the loaded config, including ones you did not select, but it reads no private keys and is not a decrypt dry run.

## Inspecting secrets

```bash
# print one file's plaintext without touching disk
yews view config.enc.toml

# pipe into structured tools
yews view config.enc.json | jq '.database'
```

`view` writes plaintext to stdout only; warnings and verbose detail stay on stderr, so the output remains pipeable. When a tool needs real files on disk, use `decrypt` instead.

## Restoring plaintext locally

```bash
# restore every registered file the current identity can read
yews decrypt

# require completeness — recommended before running the app
yews decrypt --strict
```

Skipping is lenient by default so multi-identity teams can share one repo; strict mode turns any skip into a failure without rolling back successes. See [Decryption results and strict mode](/guide/decryption-results).

## CI/CD

```bash
yews plan --json   # fail fast on config or authorization drift
yews decrypt --strict
```

Use `plan` as a cheap config gate and `decrypt --strict` before deploying, with the key injected through an environment variable. Full examples: [CI/CD integration](/guide/ci-cd) and [Running with Docker](/guide/docker).

## Onboarding and key rotation

```bash
# initial scaffold: keys, config, .sops.yaml, .gitignore
yews init

# after changing recipients in .yewseal.toml, refresh ciphertext and .sops.yaml
yews encrypt
```

`init --force` rebuilds keys and config from scratch — old ciphertext may become undecryptable, so prefer editing `.yewseal.toml` and re-running `encrypt` for recipient changes. Private keys stay out of YewSeal's scope; see [External private key sources](/guide/private-keys).
