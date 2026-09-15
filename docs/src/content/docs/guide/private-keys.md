---
title: External private key sources
---

YewSeal is not responsible for remote storage, upload, download, or distribution of private keys. The Age identity bundle needed for decryption is supplied by developers, machines, or the deployment environment; see [Configuration - reading private keys](/guide/configuration#reading-private-keys) for the resolution order.

Personal private key paths are passed via `--key-file` or environment variables and never written into `.yewseal.toml`. Without an explicit identity, YewSeal finally tries `.age/keys.txt` under the current working directory; `init` can still generate the initial key there, but the path is never registered in project config.

Different developers, workstations, and production environments may use different keys. `[recipients.registry]` and the per-file authorization sets register public recipients only; environments are not required to share one private key. Give each environment only the identities it actually needs; never commit private keys to version control or print them into CI logs.

## Infisical reference script

When private keys are hosted in Infisical, you can independently use the [Infisical CLI](https://infisical.com/docs/cli/commands/secrets) to export a secret's full value. Install the CLI and complete a login or machine identity first; authentication, access control, and secret contents are all managed by Infisical.

The POSIX shell script below is a reference to sync your age key via Infisical in different environments. Replace the project ID, environment, path, and secret name with the values for the current machine.

```sh
#!/bin/sh
set -eu
umask 077

mkdir -p .age
tmp=$(mktemp .age/keys.txt.XXXXXX)
trap 'rm -f "$tmp"' EXIT
trap 'exit 1' HUP INT TERM

infisical secrets get AGE_KEY_FILE --plain --silent \
  --projectId 'your-project-id' \
  --env 'dev' \
  --path '/yewseal' > "$tmp"
test -s "$tmp"
mv -f "$tmp" .age/keys.txt

yews --key-file .age/keys.txt decrypt
```

The script replaces the target only after a successful non-empty export, so a failing remote command never wipes an existing key; the temp file and the replaced file are mode `0600`. It does not validate the key format — YewSeal validates on read. Run it in a working directory you control and add `.age/` to `.gitignore`.

Configure `INFISICAL_TOKEN` or another authentication method per the [official Infisical docs](https://infisical.com/docs/cli/overview). For Docker, export the key on the host first, mount the file read-only into the container, and point `--key-file` at the in-container path.

## Exporting a local key

The other direction is just as common: after `yews init` generates a fresh identity on a new machine, back the key up to the channel you trust, or hand it to the deployment environment. Export the complete `.age/keys.txt` file — it may hold several identities — not a single line of it.

The reference script below mirrors the Infisical download above and pushes the local bundle into the same secret:

```sh
#!/bin/sh
set -eu

keyfile=$PWD/.age/keys.txt
test -s "$keyfile"

infisical secrets set "AGE_KEY_FILE=@$keyfile" \
  --projectId 'your-project-id' \
  --env 'dev' \
  --path '/yewseal' \
  --silent
```

`set` creates or overwrites the secret, and the `@file` form makes the CLI read the value from the file directly, so the key never appears in argv, shell history, or terminal output. Overwriting replaces the remote bundle wholesale: if the stored value held identities other than yours, merge them into the local file before pushing.

For a CI runner that consumes the key through an environment variable, the GitHub counterpart of the same export is one command — this stores the value the [CI/CD integration](/guide/ci-cd) page consumes as `AGE_KEY`:

```bash
gh secret set AGE_KEY < .age/keys.txt
```

## Other sources

Password managers, cloud secret managers, CI secrets, and local files all follow the same division of responsibility: the external tool provides the identity, YewSeal uses it to decrypt.
