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

## Per-identity secrets

The scripts above synchronize the whole bundle as one value, which couples every identity on the machine. The alternative is one secret per identity, named `YEWS_{alias}` after the registry alias: each environment pulls exactly the identities it needs, and each holder pushes only their own key — an overwrite can never clobber identities someone else pushed.

The push script consumes `yews identities --json --reveal`: YewSeal resolves the first-win identity chain itself, derives each identity's public key, and looks up the registry alias, so the script never parses key files or `.yewseal.toml`. The `# public key:` comment lines that `yews init` writes stay useful for humans reading `.age/keys.txt` — and the pull script recreates them — but nothing requires them anymore.

Each secret value stays a single bare `AGE-SECRET-KEY-1...` line. Comments never leave the local file, and Infisical's per-secret comment field is not used either: the CLI cannot set it, so anything stored there has to be maintained in the WebUI and no script can rely on it. The pull script reads `.yewseal.toml` through python3 (3.11+) and its standard `tomllib` — hand-rolled greps over TOML break on quoting styles and table layouts; the push script needs only plain python3 to read the identities JSON.

```sh
#!/bin/sh
set -eu
umask 077

config=$PWD/.yewseal.toml
test -s "$config" || { echo "no $config here" >&2; exit 1; }

tmp=$(mktemp)
report=$(mktemp)
pairs=$(mktemp)
trap 'rm -f "$tmp" "$report" "$pairs"' EXIT
trap 'exit 1' HUP INT TERM

yews identities --json --reveal > "$report"

python3 - "$report" > "$pairs" <<'PY'
import json, sys

with open(sys.argv[1]) as f:
    report = json.load(f)
for identity in report["identities"]:
    if identity.get("alias"):
        print(identity["alias"], identity["secret"])
PY

while read -r alias secret; do
  printf '%s\n' "$secret" > "$tmp"
  infisical secrets set "YEWS_$alias=@$tmp" \
    --projectId 'your-project-id' \
    --env 'dev' \
    --path '/yewseal' \
    --silent
  echo "pushed YEWS_$alias"
done < "$pairs"
```

The report flows through mode-`0600` temporary files, so secret keys never touch argv, shell history, or terminal output. An identity whose public key has no registry alias is warned on stderr by `yews identities` and skipped, so a borrowed or unregistered key is never uploaded under a wrong name. The script pushes whichever identities YewSeal would actually decrypt with — to push a different layer, point `--key-file` or `YEWSEAL_AGE_IDENTITIES` at it first.

The pull direction takes a comma-separated alias list and rebuilds `.age/keys.txt` — the path YewSeal reads by default, so no environment variable is needed afterwards. It uses the same registry in reverse: for each alias it looks up the public key in `.yewseal.toml` and fails immediately on an unregistered alias, then rewrites the `# public key:` comment line above the fetched secret key. The rebuilt file is exactly what the push script expects, so pull → push round-trips:

```sh
#!/bin/sh
set -eu
umask 077

: "${ALIASES:?set ALIASES to a comma-separated alias list, e.g. ALIASES=owner,ci $0}"

config=$PWD/.yewseal.toml
test -s "$config" || { echo "no $config here" >&2; exit 1; }

mkdir -p .age
one=$(mktemp)
bundle=$(mktemp .age/keys.txt.XXXXXX)
table=$(mktemp)
trap 'rm -f "$one" "$bundle" "$table"' EXIT
trap 'exit 1' HUP INT TERM

python3 - "$config" "$ALIASES" > "$table" <<'PY'
import sys, tomllib

config, aliases = sys.argv[1], sys.argv[2]
with open(config, "rb") as f:
    registry = tomllib.load(f)["recipients"]["registry"]
for alias in aliases.replace(",", " ").split():
    print(alias, registry.get(alias, ""))
PY

while read -r alias public_key; do
  if [ -z "$public_key" ]; then
    echo "alias $alias is not registered in $config" >&2
    exit 1
  fi
  infisical secrets get "YEWS_$alias" --plain --silent \
    --projectId 'your-project-id' \
    --env 'dev' \
    --path '/yewseal' > "$one"
  test -s "$one" || { echo "YEWS_$alias came back empty or missing" >&2; exit 1; }
  value=$(cat "$one")
  case $value in
    '# public key: '*) printf '%s\n' "$value" >> "$bundle" ;;
    *) printf '# public key: %s\n%s\n' "$public_key" "$value" >> "$bundle" ;;
  esac
done < "$table"

test -s "$bundle"
mv -f "$bundle" .age/keys.txt
```

The bundle file only appears after every fetch succeeded and is non-empty, and it is mode `0600`. YewSeal splits identities on commas, spaces, and newlines alike, so multi-line secret values or a hand-joined bundle all parse.

For a one-off shell or a CI runner, skip the file and join the per-identity values with commas straight into the environment variable — command substitution keeps the values out of shell history, and on GitHub Actions the repository secrets mirror the Infisical names upper-cased:

```bash
export YEWSEAL_AGE_IDENTITIES="$(infisical secrets get YEWS_owner --plain --silent --projectId your-project-id --env dev --path /yewseal),$(infisical secrets get YEWS_deploy --plain --silent --projectId your-project-id --env dev --path /yewseal)"
```

```yaml
env:
  YEWSEAL_AGE_IDENTITIES: ${{ secrets.YEWS_OWNER }},${{ secrets.YEWS_DEPLOY }}
```

## Other sources

Password managers, cloud secret managers, CI secrets, and local files all follow the same division of responsibility: the external tool provides the identity, YewSeal uses it to decrypt.
