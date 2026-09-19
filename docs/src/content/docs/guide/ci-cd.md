---
title: CI/CD integration
---

A CI environment has no interactive terminal and no local `.age/keys.txt`; the usual pattern is to inject the Age private key through an environment variable and run `yews decrypt` to restore the configuration files.

## GitHub Actions

After declaring `github:YewFence/YewSeal` in the repository's `mise.toml`, install it with [mise-action](https://github.com/jdx/mise-action) and inject the key from repository secrets:

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

Store the private key value from `.age/keys.txt` as a repository secret (`AGE_KEY` above); `YEWSEAL_AGE_IDENTITIES` then passes it directly to YewSeal — `gh secret set AGE_KEY < .age/keys.txt` does it in one command. See [Configuration - reading private keys](/guide/configuration#reading-private-keys) for the resolution order.

## With Infisical

If private keys are hosted in Infisical, use its CLI independently to export the current deployment environment's identity first, then hand it to `yews`. Reference scripts and authentication notes live in [External private key sources](/guide/private-keys#infisical-reference-script). YewSeal never calls Infisical and does not require production to share a key with development machines.

## Other CI systems

The same pattern works for any CI:

1. Install `yews` (mise, go install, a release binary, or the [Docker image](/guide/docker))
2. Provide the Age private key via an environment variable or a key file
3. Run `yews decrypt --strict` and deploy only after a successful exit

Development environments can tolerate files skipped for missing identities, but deployment usually requires every selected file to decrypt. Lenient mode exits `0` even when every selected file is skipped, so a plain `yews decrypt` never proves that anything was restored — use `--strict` (or `YEWSEAL_DECRYPT_STRICT=true`) whenever the job deploys what it decrypts. See [Decryption results and strict mode](/guide/decryption-results) for result classification and exit codes.

A repository partitioned by identity — one key for image scanning, another for deployment configs — gives each job one strict gate over its own scope. A job that treats one credential as optional stays lenient and probes the JSON report for that file instead; see [Probing and gating in scripts](/guide/decryption-results#probing-and-gating-in-scripts).
