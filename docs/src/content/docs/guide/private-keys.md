---
title: External private key sources
---

YewSeal is not responsible for remote storage, upload, download, or distribution of private keys, and provides no `sync` command or provider integration. The Age identity bundle needed for decryption is supplied by developers, machines, or the deployment environment; see [Configuration - reading private keys](/guide/configuration#reading-private-keys) for the resolution order.

Personal private key paths are passed via `--key-file` or environment variables and never written into `.yewseal.toml`. Without an explicit identity, YewSeal finally tries `.age/keys.txt` under the current working directory; `init` can still generate the initial key there, but the path is never registered in project config.

Different developers, workstations, and production environments may use different keys. `[recipients.registry]` and the per-file authorization sets register public recipients only; environments are not required to share one private key. Give each environment only the identities it actually needs; never commit private keys to version control or print them into CI logs.

## Infisical reference script

When private keys are hosted in Infisical, you can independently use the [Infisical CLI](https://infisical.com/docs/cli/commands/secrets) to export a secret's full value. Install the CLI and complete a login or machine identity first; authentication, access control, and secret contents are all managed by Infisical. YewSeal neither inspects `.infisical.json` nor invokes the Infisical CLI.

The POSIX shell script below is a reference, not a built-in feature. Replace the project ID, environment, path, and secret name with the values for the current machine; the secret should be the complete Age private key file, which may contain multiple identities, not a dump of project-wide environment variables.

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

The script replaces the target only after a successful non-empty export, so a failing remote command never wipes an existing key; the temp file and the replaced file are mode `0600`. It does not validate the key format — YewSeal validates on read. Run it in a working directory you control and add `.age/` to `.gitignore`. An existing key gets overwritten, and a later decryption failure does not roll the replacement back.

Configure `INFISICAL_TOKEN` or another authentication method per the [official Infisical docs](https://infisical.com/docs/cli/overview). For Docker, export the key on the host first, mount the file read-only into the container, and point `--key-file` at the in-container path.

## Other sources

Password managers, cloud secret managers, CI secrets, and local files all follow the same division of responsibility: the external tool provides the identity, YewSeal uses it to decrypt. No per-source provider or project config is needed.
