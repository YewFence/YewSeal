---
title: CI/CD integration
---

A CI environment has no interactive terminal and no local `.age/keys.txt`; the usual pattern is to inject the Age private key through an environment variable and run `yews decrypt` to restore the configuration files.

## GitHub Actions

Two common ways to install `yews` on a runner: plain `actions/setup-go` plus `go install` needs no extra tooling, while [mise-action](https://github.com/jdx/mise-action) fits repositories that already declare `github:YewFence/YewSeal` in their `mise.toml`. Both examples below inject the key from repository secrets and gate the deploy on strict decryption.

### setup-go and go install

`go-version: stable` matters: YewSeal tracks a current Go toolchain, and runner defaults lag behind it.

```yaml
name: Deploy
on: push

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7
        with:
          persist-credentials: false

      - uses: actions/setup-go@v6
        with:
          go-version: stable

      - name: Install YewSeal
        run: go install github.com/YewFence/YewSeal/cmd/yews@latest

      - name: Decrypt deployment configuration
        env:
          YEWSEAL_AGE_IDENTITIES: ${{ secrets.AGE_KEY }}
          YEWSEAL_DECRYPT_STRICT: "true"
        run: yews decrypt ./deploy

      - name: Deploy
        uses: cloudflare/wrangler-action@v4
        with:
          apiToken: ${{ secrets.CLOUDFLARE_API_TOKEN }}
          command: deploy --config deploy/wrangler.toml
```

The deploy step uses `cloudflare/wrangler-action` rather than a bare `wrangler deploy` because runner images do not ship Wrangler: the action installs it and takes its Cloudflare token from a `CLOUDFLARE_API_TOKEN` repository secret.

### mise

After declaring `github:YewFence/YewSeal` in the repository's `mise.toml`, let the action install it:

```yaml
name: Deploy
on: push

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7
        with:
          persist-credentials: false

      - uses: jdx/mise-action@v4

      - name: Decrypt deployment configuration
        env:
          YEWSEAL_AGE_IDENTITIES: ${{ secrets.AGE_KEY }}
          YEWSEAL_DECRYPT_STRICT: "true"
        run: yews decrypt ./deploy

      - name: Deploy
        run: wrangler deploy --config deploy/wrangler.toml
```

A bare `wrangler deploy` is enough here because `mise-action` installs every tool the repository declares: add `wrangler = "4"` to `[tools]` next to `github:YewFence/YewSeal`, and the runner gets Wrangler from the same list with no separate install step. The token still has to come from the step: Wrangler reads `CLOUDFLARE_API_TOKEN` from the environment, where the action above takes it as its `apiToken` input.

In both examples, store the private key value from `.age/keys.txt` as a repository secret (`AGE_KEY` above); `YEWSEAL_AGE_IDENTITIES` then passes it directly to YewSeal — `gh secret set AGE_KEY < .age/keys.txt` does it in one command. See [Configuration - reading private keys](/guide/configuration#reading-private-keys) for the resolution order.

:::note[Version pinning]
A common best practice is pinning actions to commit SHAs and `go install` to a release tag, and letting a bot like [Renovate](https://docs.renovatebot.com/) keep them updated — general CI hygiene rather than YewSeal-specific guidance.
:::

## Checking repository health

Deploy jobs prove that ciphertext can be opened; a pull-request job can also prove that the repository itself is safe, without holding any private key:

```yaml
      - name: Verify encrypted configuration
        run: yews verify --no-decrypt
```

`--no-decrypt` keeps the job keyless and its check set reproducible; a runner that does receive a key can gate on `yews verify --decrypt` instead. What each layer checks, what the findings mean, and how to repair them is covered in [Verifying repository health](/guide/verifying).

## With Infisical

If private keys are hosted in Infisical, use its CLI independently to export the current deployment environment's identity first, then hand it to `yews`. Reference scripts and authentication notes live in [External private key sources](/guide/private-keys#infisical-reference-script). YewSeal never calls Infisical and does not require production to share a key with development machines.

## Other CI systems

The same pattern works for any CI:

1. Install `yews` (mise, go install, a release binary, or the [Docker image](/guide/docker))
2. Provide the Age private key via an environment variable or a key file
3. Run `yews decrypt --strict` and deploy only after a successful exit

Development environments can tolerate files skipped for missing identities, but deployment usually requires every selected file to decrypt. Lenient mode exits `0` even when every selected file is skipped, so a plain `yews decrypt` never proves that anything was restored — use `--strict` (or `YEWSEAL_DECRYPT_STRICT=true`) whenever the job deploys what it decrypts. See [Decryption results and strict mode](/guide/decryption-results) for result classification and exit codes.

A repository partitioned by identity — one key for image scanning, another for deployment configs — gives each job one strict gate over its own scope. A job that treats one credential as optional stays lenient and probes the JSON report for that file instead; see [Probing and gating in scripts](/guide/decryption-results#probing-and-gating-in-scripts).
