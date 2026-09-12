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

      - name: Decrypt configuration
        env:
          SOPS_AGE_KEY: ${{ secrets.AGE_KEY }}
        run: yews decrypt --strict

      - name: Deploy
        run: wrangler deploy
```

Store the private key value from `.age/keys.txt` as a repository secret (`AGE_KEY` above); the `SOPS_AGE_KEY` environment variable is then used directly. See [Configuration - reading private keys](/guide/configuration#reading-private-keys) for the resolution order.

## With Infisical

If private keys are hosted in Infisical, use its CLI independently to export the current deployment environment's identity first, then hand it to `yews`. Reference scripts and authentication notes live in [External private key sources](/guide/private-keys#infisical-reference-script). YewSeal never calls Infisical and does not require production to share a key with development machines.

## Other CI systems

The same pattern works for any CI:

1. Install `yews` (mise, go install, a release binary, or the [Docker image](/guide/docker))
2. Provide the Age private key via an environment variable or a key file
3. Run `yews decrypt --strict` and deploy only after a successful exit

Development environments can tolerate files skipped for missing identities, but deployment usually requires every selected file to decrypt. Strict mode can also be enabled with `YEWSEAL_STRICT=true`; see [Decryption results and strict mode](/guide/decryption-results) for result classification and exit codes.
